package connect

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"strings"
	"testing"

	"github.com/rstudio/connect-client/internal/clients/connect/server_settings"
	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/types"
	"github.com/rstudio/connect-client/internal/util/utiltest"
	"github.com/stretchr/testify/suite"
)

type CapabilitiesSuite struct {
	utiltest.Suite
}

func TestCapabilitiesSuite(t *testing.T) {
	suite.Run(t, new(CapabilitiesSuite))
}

func (s *CapabilitiesSuite) TestEmptyConfig() {
	a := allSettings{}
	cfg := &config.Config{}
	s.Nil(a.checkConfig(cfg))
}

func makePythonConfig(version string) *config.Config {
	return &config.Config{
		Python: &config.Python{
			Version: version,
		},
	}
}

func (s *CapabilitiesSuite) TestCheckMatchingPython() {
	a := allSettings{
		python: server_settings.PyInfo{
			Installations: []server_settings.PyInstallation{
				{Version: "3.10.1"},
				{Version: "3.11.2"},
			},
		},
	}
	s.Nil(a.checkConfig(makePythonConfig("3.10.1")))
	s.Nil(a.checkConfig(makePythonConfig("3.11.1")))
	err := a.checkConfig(makePythonConfig("3.9.1"))
	s.NotNil(err)
	s.Equal(err.Code, types.PythonNotAvailableCode)
	s.ErrorIs(err.Err, errPythonNotAvailable)
	s.Equal("3.9", err.Data["requested"])
	s.Equal([]string{"3.10.1", "3.11.2"}, err.Data["available"])
}

func makeMinMaxProcs(min, max int32) *config.Config {
	return &config.Config{
		Type: config.ContentTypePythonShiny,
		Connect: &config.Connect{
			Runtime: &config.ConnectRuntime{
				MinProcesses: &min,
				MaxProcesses: &max,
			},
		},
	}
}

func (s *CapabilitiesSuite) TestMinMaxProcs() {
	a := allSettings{
		scheduler: server_settings.SchedulerSettings{
			MinProcesses:      0,
			MaxProcesses:      3,
			MinProcessesLimit: 10,
			MaxProcessesLimit: 20,
		},
	}
	s.Nil(a.checkConfig(makeMinMaxProcs(1, 5)))
	s.Nil(a.checkConfig(makeMinMaxProcs(5, 5)))

	err := a.checkConfig(makeMinMaxProcs(11, 11))
	s.Equal(types.ValueOutOfRangeCode, err.Code)
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "min-processes",
		"value": 11.0,
		"min":   0.0,
		"max":   10.0,
	}, err.Data)

	err = a.checkConfig(makeMinMaxProcs(0, 21))
	s.Equal(types.ValueOutOfRangeCode, err.Code)
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "max-processes",
		"value": 21.0,
		"min":   0.0,
		"max":   20.0,
	}, err.Data)

	err = a.checkConfig(makeMinMaxProcs(5, 1))
	s.Equal(types.MinGreaterThanMaxCode, err.Code)
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":    "min-processes",
		"value":  5.0,
		"maxKey": "max-processes",
		"max":    1.0,
	}, err.Data)

	err = a.checkConfig(makeMinMaxProcs(-1, 5))
	s.Equal(types.ValueOutOfRangeCode, err.Code)
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "min-processes",
		"value": -1.0,
		"min":   0.0,
		"max":   10.0,
	}, err.Data)
}

func (s *CapabilitiesSuite) TestRuntimeNonWorker() {
	cfg := &config.Config{
		Type: config.ContentTypeHTML,
		Connect: &config.Connect{
			Runtime: &config.ConnectRuntime{},
		},
	}
	a := allSettings{}
	err := a.checkConfig(cfg)
	s.Equal(types.RuntimeSettingsForStaticContentCode, err.Code)
	s.ErrorIs(err.Err, errRuntimeSettingsForStaticContent)
}

func (s *CapabilitiesSuite) TestRunAs() {
	adminSettings := allSettings{
		user: UserDTO{
			UserRole: AuthRoleAdmin,
		},
	}
	publisherSettings := allSettings{
		user: UserDTO{
			UserRole: AuthRolePublisher,
		},
	}
	cfg := &config.Config{
		Connect: &config.Connect{
			Access: &config.ConnectAccess{
				RunAs: "someuser",
			},
		},
	}
	s.Nil(adminSettings.checkConfig(cfg))

	err := publisherSettings.checkConfig(cfg)
	s.Equal(types.AdminPrivilegesRequiredCode, err.Code)
	s.ErrorIs(err.Err, errAdminPrivilegesRequired)
	s.Equal(types.ErrorData{
		"key": "run-as",
	}, err.Data)
}

func (s *CapabilitiesSuite) TestRunAsCurrentUser() {
	goodSettings := allSettings{
		user: UserDTO{
			UserRole: AuthRoleAdmin,
		},
		general: server_settings.ServerSettings{
			License: server_settings.LicenseStatus{
				CurrentUserExecution: true,
			},
		},
		application: server_settings.ApplicationSettings{
			RunAsCurrentUser: true,
		},
	}
	truth := true
	cfg := config.Config{
		Type: config.ContentTypePythonDash,
		Connect: &config.Connect{
			Access: &config.ConnectAccess{
				RunAsCurrentUser: &truth,
			},
		},
	}
	s.Nil(goodSettings.checkConfig(&cfg))

	noLicense := goodSettings
	noLicense.general.License.CurrentUserExecution = false
	err := noLicense.checkConfig(&cfg)
	s.ErrorIs(err.Err, errCurrentUserExecutionNotLicensed)

	noConfig := goodSettings
	noConfig.application.RunAsCurrentUser = false
	err = noConfig.checkConfig(&cfg)
	s.Equal(types.CurrentUserExecutionNotConfiguredCode, err.Code)
	s.ErrorIs(err.Err, errCurrentUserExecutionNotConfigured)

	notAdmin := goodSettings
	notAdmin.user.UserRole = AuthRolePublisher
	err = notAdmin.checkConfig(&cfg)
	s.Equal(types.AdminPrivilegesRequiredCode, err.Code)
	s.ErrorIs(err.Err, errAdminPrivilegesRequired)
	s.Equal(types.ErrorData{"key": "run-as-current-user"}, err.Data)

	notAnApp := cfg
	notAnApp.Type = config.ContentTypeJupyterNotebook
	err = goodSettings.checkConfig(&notAnApp)
	s.Equal(types.OnlyAppsCanRACUCode, err.Code)
	s.ErrorIs(err.Err, errOnlyAppsCanRACU)
}

func (s *CapabilitiesSuite) TestAPILicense() {
	allowed := allSettings{
		general: server_settings.ServerSettings{
			License: server_settings.LicenseStatus{
				AllowAPIs: true,
			},
		},
	}
	notAllowed := allSettings{
		general: server_settings.ServerSettings{
			License: server_settings.LicenseStatus{
				AllowAPIs: false,
			},
		},
	}
	missing := allSettings{}
	cfg := &config.Config{
		Type: config.ContentTypePythonFlask,
	}
	s.Nil(allowed.checkConfig(cfg))

	err := missing.checkConfig(cfg)
	s.ErrorIs(err.Err, errAPIsNotLicensed)
	s.Equal(types.APIsNotLicensedCode, err.Code)

	err = notAllowed.checkConfig(cfg)
	s.ErrorIs(err.Err, errAPIsNotLicensed)
	s.Equal(types.APIsNotLicensedCode, err.Code)
}

func (s *CapabilitiesSuite) TestFieldLengths() {
	a := allSettings{}
	tooLong := strings.Repeat("spam", 10000)
	cfg := &config.Config{
		Description: tooLong,
	}
	err := a.checkConfig(cfg)
	s.Equal(types.DescriptionTooLongCode, err.Code)
	s.ErrorIs(err.Err, errDescriptionTooLong)
}

func (s *CapabilitiesSuite) TestKubernetesEnablement() {
	goodSettings := allSettings{
		user: UserDTO{
			UserRole: AuthRoleAdmin,
		},
		general: server_settings.ServerSettings{
			ExecutionType:                server_settings.ExecutionTypeKubernetes,
			DefaultImageSelectionEnabled: true,
			License: server_settings.LicenseStatus{
				LauncherEnabled: true,
			},
		},
	}

	cfg := config.Config{
		Connect: &config.Connect{
			Kubernetes: &config.ConnectKubernetes{
				DefaultImageName:   "image",
				ServiceAccountName: "account",
			},
		},
	}
	s.Nil(goodSettings.checkConfig(&cfg))

	noLicense := goodSettings
	noLicense.general.License.LauncherEnabled = false
	err := noLicense.checkConfig(&cfg)
	s.Equal(types.KubernetesNotLicensedCode, err.Code)
	s.ErrorIs(err.Err, errKubernetesNotLicensed)

	noConfig := goodSettings
	noConfig.general.ExecutionType = server_settings.ExecutionTypeLocal
	err = noConfig.checkConfig(&cfg)
	s.Equal(types.KubernetesNotConfiguredCode, err.Code)
	s.ErrorIs(err.Err, errKubernetesNotConfigured)

	noImageSelection := goodSettings
	noImageSelection.general.DefaultImageSelectionEnabled = false
	err = noImageSelection.checkConfig(&cfg)
	s.Equal(types.ImageSelectionNotEnabledCode, err.Code)
	s.ErrorIs(err.Err, errImageSelectionNotEnabled)

	noAdmin := goodSettings
	noAdmin.user.UserRole = AuthRolePublisher
	err = noAdmin.checkConfig(&cfg)
	s.Equal(types.AdminPrivilegesRequiredCode, err.Code)
	s.ErrorIs(err.Err, errAdminPrivilegesRequired)
}

func makeCpuRequestLimit(req, limit float64) *config.Config {
	return &config.Config{
		Connect: &config.Connect{
			Kubernetes: &config.ConnectKubernetes{
				CPURequest: &req,
				CPULimit:   &limit,
			},
		},
	}
}

func makeMemoryRequestLimit(req, limit int64) *config.Config {
	return &config.Config{
		Connect: &config.Connect{
			Kubernetes: &config.ConnectKubernetes{
				MemoryRequest: &req,
				MemoryLimit:   &limit,
			},
		},
	}
}

func makeGPURequest(amd, nvidia int64) *config.Config {
	return &config.Config{
		Connect: &config.Connect{
			Kubernetes: &config.ConnectKubernetes{
				AMDGPULimit:    &amd,
				NvidiaGPULimit: &nvidia,
			},
		},
	}
}

var kubernetesEnabledSettings = allSettings{
	general: server_settings.ServerSettings{
		ExecutionType: server_settings.ExecutionTypeKubernetes,
		License: server_settings.LicenseStatus{
			LauncherEnabled: true,
		},
	},
}

func (s *CapabilitiesSuite) TestKubernetesRuntimeCPU() {
	a := kubernetesEnabledSettings
	a.scheduler = server_settings.SchedulerSettings{
		CPURequest:    1.0,
		CPULimit:      2.0,
		MaxCPURequest: 3.0,
		MaxCPULimit:   4.0,
	}
	s.Nil(a.checkConfig(makeCpuRequestLimit(1.0, 2.0)))
	s.Nil(a.checkConfig(makeCpuRequestLimit(3.0, 3.0)))
	s.Nil(a.checkConfig(makeCpuRequestLimit(3.0, 0.0)))

	err := a.checkConfig(makeCpuRequestLimit(-1.0, 2.0))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "cpu-request",
		"value": -1.0,
		"min":   0.0,
		"max":   3.0,
	}, err.Data)

	err = a.checkConfig(makeCpuRequestLimit(1.0, -2.0))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "cpu-limit",
		"value": -2.0,
		"min":   0.0,
		"max":   4.0,
	}, err.Data)

	err = a.checkConfig(makeCpuRequestLimit(4.0, 4.0))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "cpu-request",
		"value": 4.0,
		"min":   0.0,
		"max":   3.0,
	}, err.Data)

	err = a.checkConfig(makeCpuRequestLimit(1.0, 10.0))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "cpu-limit",
		"value": 10.0,
		"min":   0.0,
		"max":   4.0,
	}, err.Data)

	err = a.checkConfig(makeCpuRequestLimit(3.0, 2.0))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":    "cpu-request",
		"value":  3.0,
		"maxKey": "cpu-limit",
		"max":    2.0,
	}, err.Data)
}

func (s *CapabilitiesSuite) TestKubernetesRuntimeNoConfiguredLimits() {
	a := kubernetesEnabledSettings
	a.scheduler = server_settings.SchedulerSettings{
		CPURequest:    1.0,
		CPULimit:      2.0,
		MaxCPURequest: 0.0,
		MaxCPULimit:   0.0,
	}
	s.Nil(a.checkConfig(makeCpuRequestLimit(10.0, 10.0)))
}

func (s *CapabilitiesSuite) TestKubernetesRuntimeMemory() {
	a := kubernetesEnabledSettings
	a.scheduler = server_settings.SchedulerSettings{
		MemoryRequest:    1000,
		MemoryLimit:      2000,
		MaxMemoryRequest: 3000,
		MaxMemoryLimit:   4000,
	}
	s.Nil(a.checkConfig(makeMemoryRequestLimit(1000, 2000)))
	s.Nil(a.checkConfig(makeMemoryRequestLimit(3000, 3000)))
	s.Nil(a.checkConfig(makeMemoryRequestLimit(3000, 0)))

	err := a.checkConfig(makeMemoryRequestLimit(-1000, 2000))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "memory-request",
		"value": float64(-1000),
		"min":   float64(0),
		"max":   float64(3000),
	}, err.Data)

	err = a.checkConfig(makeMemoryRequestLimit(1000, -2000))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "memory-limit",
		"value": float64(-2000),
		"min":   float64(0),
		"max":   float64(4000),
	}, err.Data)

	err = a.checkConfig(makeMemoryRequestLimit(4000, 4000))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "memory-request",
		"value": float64(4000),
		"min":   float64(0),
		"max":   float64(3000),
	}, err.Data)

	err = a.checkConfig(makeMemoryRequestLimit(1000, 10000))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":   "memory-limit",
		"value": float64(10000),
		"min":   float64(0),
		"max":   float64(4000),
	}, err.Data)

	err = a.checkConfig(makeMemoryRequestLimit(3000, 2000))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(types.ErrorData{
		"key":    "memory-request",
		"value":  float64(3000),
		"maxKey": "memory-limit",
		"max":    float64(2000),
	}, err.Data)
}

func (s *CapabilitiesSuite) TestKubernetesGPULimits() {
	a := kubernetesEnabledSettings
	a.scheduler = server_settings.SchedulerSettings{
		MaxAMDGPULimit:    1,
		MaxNvidiaGPULimit: 2,
	}
	s.Nil(a.checkConfig(makeGPURequest(0, 1)))
	s.Nil(a.checkConfig(makeGPURequest(1, 0)))
	s.Nil(a.checkConfig(makeGPURequest(1, 1)))

	err := a.checkConfig(makeGPURequest(5, 0))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(err.Data, types.ErrorData{
		"key":   "amd-gpu-limit",
		"value": float64(5),
		"min":   float64(0),
		"max":   float64(1),
	})
	err = a.checkConfig(makeGPURequest(0, 5))
	s.ErrorIs(err.Err, errValueOutOfRange)
	s.Equal(err.Data, types.ErrorData{
		"key":   "nvidia-gpu-limit",
		"value": float64(5),
		"min":   float64(0),
		"max":   float64(2),
	})
}
