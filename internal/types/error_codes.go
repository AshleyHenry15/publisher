package types

// Copyright (C) 2023 by Posit Software, PBC.

const (
	NoError                               ErrorCode = ""
	UnknownErrorCode                      ErrorCode = "unknown"
	InternalErrorCode                     ErrorCode = "internalErr"                       // Bug or other error. Not user actionable.
	AuthenticationFailedCode              ErrorCode = "authFailedErr"                     // Couldn't authenticate to publishing server
	PermissionsCode                       ErrorCode = "permissionErr"                     // Server responded with 403 forbidden
	OperationTimedOutCode                 ErrorCode = "timeoutErr"                        // HTTP request to publishing server timed out
	ConnectionFailedCode                  ErrorCode = "connectionFailed"                  // Couldn't connect to Connect
	TimeoutErrorCode                      ErrorCode = "serverTimeout"                     // Timeout from publishing server
	ServerErrorCode                       ErrorCode = "serverErr"                         // HTTP 5xx code from publishing server
	AccountLockedCode                     ErrorCode = "accountLocked"                     // Connect user acccount is locked
	AccountNotConfirmedCode               ErrorCode = "accountNotConfirmed"               // Connect user acccount is not confirmed
	AccountNotPublisherCode               ErrorCode = "accountNotPublisher"               // Connect user acccount is not a publisher or admin
	VanityURLNotAvailableCode             ErrorCode = "vanityURLNotAvailableErr"          // Vanity URL already in use
	FileNotFoundCode                      ErrorCode = "fileNotFound"                      // A file couldn't be found; refine to a more specific code when possible
	ConfigurationNotFoundCode             ErrorCode = "configurationNotFoundErr"          // Could not find the named configuration
	DeploymentNotFoundCode                ErrorCode = "deploymentNotFoundErr"             // Could not find deployment to update
	AccountNotFoundCode                   ErrorCode = "accountNotFoundErr"                // Account not found
	ServerURLMismatchCode                 ErrorCode = "ServerURLMismatchErr"              // Redeployments must go to the same server
	RequirementFileMissingCode            ErrorCode = "requirementFileMissing"            // requirements.txt file is missing
	DeploymentFailedCode                  ErrorCode = "deployFailed"                      // generic deployment failure; make more specific
	DescriptionTooLongCode                ErrorCode = "descriptionTooLong"                // the description cannot be longer than 4096 characters
	CurrentUserExecutionNotLicensedCode   ErrorCode = "currentUserExecutionNotLicensed"   // run-as-current-user is not licensed on this Connect server
	CurrentUserExecutionNotConfiguredCode ErrorCode = "currentUserExecutionNotConfigured" // run-as-current-user is not configured on this Connect server
	OnlyAppsCanRACUCode                   ErrorCode = "onlyAppsCanRACU"                   // run-as-current-user can only be used with application types, not APIs or reports
	APIsNotLicensedCode                   ErrorCode = "apisNotLicensed"                   // API deployment is not licensed on this Connect server
	KubernetesNotLicensedCode             ErrorCode = "kubernetesNotLicensed"             // off-host execution with Kubernetes is not licensed on this Connect server
	KubernetesNotConfiguredCode           ErrorCode = "kubernetesNotConfigured"           // off-host execution with Kubernetes is not configured on this Connect server
	ImageSelectionNotEnabledCode          ErrorCode = "imageSelectionNotEnabled"          // default image selection is not enabled on this Connect server
	RuntimeSettingsForStaticContentCode   ErrorCode = "runtimeSettingsForStaticContent"   // runtime settings cannot be applied to static content
	PythonNotAvailableCode                ErrorCode = "pythonNotAvailable"                // Specified Python version is not available on this Connect server
	AdminPrivilegesRequiredCode           ErrorCode = "adminPrivilegesRequired"           // Admin privileges are required to perform this operation
	ValueOutOfRangeCode                   ErrorCode = "valueOutOfRange"                   // Value must be within the configured range
	MinGreaterThanMaxCode                 ErrorCode = "minGreaterThanMax"                 // Minimum value must be less than or equal to the maximum value
)

type PathDetails struct {
	Path string `mapstructure:"path"`
}

type AccountErrorDetails struct {
	AccountName string `mapstructure:"accountName"`
}

type VersionNotAvailableDetails struct {
	Requested string   `mapstructure:"requested"`
	Available []string `mapstructure:"available"`
}

type ConfigKeyDetails struct {
	ConfigKey string `mapstructure:"key"`
}

type ValueRangeDetails struct {
	ConfigKey string  `mapstructure:"key"`
	Value     float64 `mapstructure:"value"`
	Min       float64 `mapstructure:"min"`
	Max       float64 `mapstructure:"max"`
}

type MinGreaterThanMaxDetails struct {
	ConfigKey    string  `mapstructure:"key"`
	Value        float64 `mapstructure:"value"`
	MaxConfigKey string  `mapstructure:"maxKey"`
	Max          float64 `mapstructure:"max"`
}
