package connect

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"errors"
	"strings"

	"github.com/rstudio/connect-client/internal/clients/connect/server_settings"
	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/types"
	"github.com/rstudio/connect-client/internal/util"
)

type allSettings struct {
	user        UserDTO
	general     server_settings.ServerSettings
	application server_settings.ApplicationSettings
	scheduler   server_settings.SchedulerSettings
	python      server_settings.PyInfo
	r           server_settings.RInfo
	quarto      server_settings.QuartoInfo
}

var errRequirementsFileMissing = errors.New(
	`can't find the package file in the project directory.
Create the file, listing the packages your project depends on.
Or scan your project dependencies using the publisher UI or
the 'publisher requirements create' command`)

type RequirementsFileMissingDetails struct {
	Path string `mapstructure:"path"`
}

func checkRequirementsFile(base util.AbsolutePath, requirementsFilename string) *types.AgentError {
	packageFile := base.Join(requirementsFilename)
	exists, err := packageFile.Exists()
	if err != nil {
		return types.AsAgentError(err)
	}
	if !exists {
		return types.NewAgentError(types.RequirementFileMissingCode,
			errRequirementsFileMissing,
			types.PathDetails{Path: packageFile.String()})
	}
	return nil
}

func (c *ConnectClient) CheckCapabilities(base util.AbsolutePath, cfg *config.Config, log logging.Logger) *types.AgentError {
	if cfg.Python != nil {
		err := checkRequirementsFile(base, cfg.Python.PackageFile)
		if err != nil {
			return err
		}
	}
	settings, err := c.getSettings(cfg, log)
	if err != nil {
		return types.AsAgentError(err)
	}
	return settings.checkConfig(cfg)
}

func (c *ConnectClient) getSettings(cfg *config.Config, log logging.Logger) (*allSettings, error) {
	settings := &allSettings{}

	err := c.client.Get("/__api__/v1/user", &settings.user, log)
	if err != nil {
		return nil, err
	}
	err = c.client.Get("/__api__/server_settings", &settings.general, log)
	if err != nil {
		return nil, err
	}
	err = c.client.Get("/__api__/server_settings/applications", &settings.application, log)
	if err != nil {
		return nil, err
	}

	schedulerPath := ""
	appMode := AppModeFromType(cfg.Type)
	if !appMode.IsStaticContent() {
		// Scheduler settings don't apply to static content,
		// and the API will err if you try.
		schedulerPath = "/" + string(appMode)
	}
	err = c.client.Get("/__api__/server_settings/scheduler"+schedulerPath, &settings.scheduler, log)
	if err != nil {
		return nil, err
	}
	err = c.client.Get("/__api__/v1/server_settings/python", &settings.python, log)
	if err != nil {
		return nil, err
	}
	err = c.client.Get("/__api__/v1/server_settings/r", &settings.r, log)
	if err != nil {
		return nil, err
	}
	err = c.client.Get("/__api__/v1/server_settings/quarto", &settings.quarto, log)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

var (
	errDescriptionTooLong                = errors.New("the description cannot be longer than 4096 characters")
	errCurrentUserExecutionNotLicensed   = errors.New("run-as-current-user is not licensed on this Connect server")
	errCurrentUserExecutionNotConfigured = errors.New("run-as-current-user is not configured on this Connect server")
	errOnlyAppsCanRACU                   = errors.New("run-as-current-user can only be used with application types, not APIs or reports")
	errAPIsNotLicensed                   = errors.New("API deployment is not licensed on this Connect server")
	errKubernetesNotLicensed             = errors.New("off-host execution with Kubernetes is not licensed on this Connect server")
	errKubernetesNotConfigured           = errors.New("off-host execution with Kubernetes is not configured on this Connect server")
	errImageSelectionNotEnabled          = errors.New("default image selection is not enabled on this Connect server")
	errRuntimeSettingsForStaticContent   = errors.New("runtime settings cannot be applied to static content")
)

var errAdminPrivilegesRequired = errors.New("this operation requires administrator privileges")

func adminError(attr string) *types.AgentError {
	return types.NewAgentError(types.AdminPrivilegesRequiredCode,
		errAdminPrivilegesRequired,
		&types.ConfigKeyDetails{ConfigKey: attr})
}

func majorMinorVersion(version string) string {
	return strings.Join(strings.Split(version, ".")[:2], ".")
}

var errPythonNotAvailable = errors.New("the configured Python version is not available on the server. Consider editing your configuration file to request one of the available versions")

func newPythonNotAvailableErr(requested string, installations []server_settings.PyInstallation) *types.AgentError {
	available := make([]string, 0, len(installations))
	for _, inst := range installations {
		available = append(available, inst.Version)
	}
	return types.NewAgentError(types.PythonNotAvailableCode,
		errPythonNotAvailable,
		&types.VersionNotAvailableDetails{
			Requested: requested,
			Available: available,
		})
}

func (a *allSettings) checkMatchingPython(version string) *types.AgentError {
	if version == "" {
		// This is prevented by version being mandatory in the schema.
		return nil
	}
	requested := majorMinorVersion(version)
	for _, inst := range a.python.Installations {
		if majorMinorVersion(inst.Version) == requested {
			return nil
		}
	}
	return newPythonNotAvailableErr(requested, a.python.Installations)
}

func (a *allSettings) checkKubernetes(cfg *config.Config) *types.AgentError {
	k := cfg.Connect.Kubernetes
	if k == nil {
		// No kubernetes config present
		return nil
	}
	if !a.general.License.LauncherEnabled {
		return types.NewAgentError(types.KubernetesNotLicensedCode, errKubernetesNotLicensed, nil)
	}
	if a.general.ExecutionType != server_settings.ExecutionTypeKubernetes {
		return types.NewAgentError(types.KubernetesNotConfiguredCode, errKubernetesNotConfigured, nil)
	}
	if k.DefaultImageName != "" && !a.general.DefaultImageSelectionEnabled {
		return types.NewAgentError(types.ImageSelectionNotEnabledCode, errImageSelectionNotEnabled, nil)
	}
	if k.ServiceAccountName != "" && !a.user.CanAdmin() {
		return adminError("service-account-name")
	}

	s := a.scheduler
	if err := checkMaxFloat("cpu-request", k.CPURequest, s.MaxCPURequest); err != nil {
		return err
	}
	if err := checkMaxFloat("cpu-limit", k.CPULimit, s.MaxCPULimit); err != nil {
		return err
	}
	if err := checkMaxInt("memory-request", k.MemoryRequest, s.MaxMemoryRequest); err != nil {
		return err
	}
	if err := checkMaxInt("memory-limit", k.MemoryLimit, s.MaxMemoryLimit); err != nil {
		return err
	}
	if err := checkMaxInt("amd-gpu-limit", k.AMDGPULimit, s.MaxAMDGPULimit); err != nil {
		return err
	}
	if err := checkMaxInt("nvidia-gpu-limit", k.NvidiaGPULimit, s.MaxNvidiaGPULimit); err != nil {
		return err
	}

	// Requests cannot be > limits
	if err := checkMinMaxFloatWithDefaults(
		"cpu-request", k.CPURequest, s.CPURequest,
		"cpu-limit", k.CPULimit, s.CPULimit,
	); err != nil {
		return err
	}
	if err := checkMinMaxIntWithDefaults(
		"memory-request", k.MemoryRequest, s.MemoryRequest,
		"memory-limit", k.MemoryLimit, s.MemoryLimit,
	); err != nil {
		return err
	}
	return nil
}

func (a *allSettings) checkRuntime(cfg *config.Config) *types.AgentError {
	r := cfg.Connect.Runtime
	if r == nil {
		// No runtime configuration present
		return nil
	}
	appMode := AppModeFromType(cfg.Type)
	if appMode.IsStaticContent() {
		return types.NewAgentError(types.RuntimeSettingsForStaticContentCode, errRuntimeSettingsForStaticContent, nil)
	}
	s := a.scheduler

	if err := checkMaxInt("max-processes", r.MaxProcesses, int32(s.MaxProcessesLimit)); err != nil {
		return err
	}
	if err := checkMaxInt("min-processes", r.MinProcesses, int32(s.MinProcessesLimit)); err != nil {
		return err
	}
	// min/max values for timeouts are validate by the schema
	if err := checkMinMaxIntWithDefaults(
		"min-processes", r.MinProcesses, int32(s.MinProcesses),
		"max-processes", r.MaxProcesses, int32(s.MaxProcesses),
	); err != nil {
		return err
	}
	return nil
}

func (a *allSettings) checkAccess(cfg *config.Config) *types.AgentError {
	if cfg.Connect.Access == nil {
		// No access configuration present
		return nil
	}
	racu := cfg.Connect.Access.RunAsCurrentUser
	if racu != nil && *racu {
		if !a.general.License.CurrentUserExecution {
			return types.NewAgentError(types.CurrentUserExecutionNotLicensedCode, errCurrentUserExecutionNotLicensed, nil)
		}
		if !a.application.RunAsCurrentUser {
			return types.NewAgentError(types.CurrentUserExecutionNotConfiguredCode, errCurrentUserExecutionNotConfigured, nil)
		}
		if !a.user.CanAdmin() {
			return adminError("run-as-current-user")
		}
		if !cfg.Type.IsAppContent() {
			return types.NewAgentError(types.OnlyAppsCanRACUCode, errOnlyAppsCanRACU, nil)
		}
	}

	if cfg.Connect.Access.RunAs != "" && !a.user.CanAdmin() {
		return adminError("run-as")
	}
	return nil
}

func (a *allSettings) checkConfig(cfg *config.Config) *types.AgentError {
	if cfg.Type.IsAPIContent() {
		if !a.general.License.AllowAPIs {
			return types.NewAgentError(types.APIsNotLicensedCode, errAPIsNotLicensed, nil)
		}
	}
	if len(cfg.Description) > 4096 {
		return types.NewAgentError(types.DescriptionTooLongCode, errDescriptionTooLong, nil)
	}
	// we don't upload thumbnails yet, but when we do, we will check MaximumAppImageSize

	if cfg.Python != nil {
		err := a.checkMatchingPython(cfg.Python.Version)
		if err != nil {
			return err
		}
	}
	if cfg.Connect != nil {
		err := a.checkAccess(cfg)
		if err != nil {
			return err
		}
		err = a.checkRuntime(cfg)
		if err != nil {
			return err
		}
		err = a.checkKubernetes(cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

var errValueOutOfRange = errors.New("value is out of range")

func checkMaxInt[T int32 | int64](attr string, valuePtr *T, limit T) *types.AgentError {
	if valuePtr == nil {
		return nil
	}
	if limit == 0 {
		return nil
	}
	value := *valuePtr
	if value < 0 || value > limit {

		return types.NewAgentError(
			types.ValueOutOfRangeCode,
			errValueOutOfRange,
			&types.ValueRangeDetails{
				ConfigKey: attr,
				Value:     float64(value),
				Min:       0,
				Max:       float64(limit),
			})
	}
	return nil
}

func checkMaxFloat(attr string, valuePtr *float64, limit float64) *types.AgentError {
	if valuePtr == nil {
		return nil
	}
	if limit == 0 {
		return nil
	}
	value := *valuePtr
	if value < 0 || value > limit {
		return types.NewAgentError(
			types.ValueOutOfRangeCode,
			errValueOutOfRange,
			&types.ValueRangeDetails{
				ConfigKey: attr,
				Value:     value,
				Min:       0,
				Max:       limit,
			})
	}
	return nil
}

func checkMinMaxIntWithDefaults[T int32 | int64](
	minAttr string, cfgMin *T, defaultMin T,
	maxAttr string, cfgMax *T, defaultMax T) *types.AgentError {

	minValue := defaultMin
	if cfgMin != nil {
		minValue = *cfgMin
	}
	maxValue := defaultMax
	if cfgMax != nil {
		maxValue = *cfgMax
	}
	if maxValue == 0 {
		// no limit
		return nil
	}
	if minValue > maxValue {
		return types.NewAgentError(
			types.MinGreaterThanMaxCode,
			errValueOutOfRange,
			&types.MinGreaterThanMaxDetails{
				ConfigKey:    minAttr,
				MaxConfigKey: maxAttr,
				Value:        float64(minValue),
				Max:          float64(maxValue),
			})
	}
	return nil
}

func checkMinMaxFloatWithDefaults(
	minAttr string, cfgMin *float64, defaultMin float64,
	maxAttr string, cfgMax *float64, defaultMax float64) *types.AgentError {

	minValue := defaultMin
	if cfgMin != nil {
		minValue = *cfgMin
	}
	maxValue := defaultMax
	if cfgMax != nil {
		maxValue = *cfgMax
	}
	if maxValue == 0.0 {
		// no limit
		return nil
	}
	if minValue > maxValue {
		return types.NewAgentError(
			types.MinGreaterThanMaxCode,
			errValueOutOfRange,
			&types.MinGreaterThanMaxDetails{
				ConfigKey:    minAttr,
				MaxConfigKey: maxAttr,
				Value:        float64(minValue),
				Max:          float64(maxValue),
			})
	}
	return nil
}
