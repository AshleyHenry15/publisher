package types

// Copyright (C) 2023 by Posit Software, PBC.

const (
	NoError                   ErrorCode = ""
	InternalErrorCode         ErrorCode = "internalErr"              // Bug or other error. Not user actionable.
	AuthenticationFailedCode  ErrorCode = "authFailedErr"            // Couldn't authenticate to publishing server
	PermissionsCode           ErrorCode = "permissionErr"            // Server responded with 403 forbidden
	OperationTimedOutCode     ErrorCode = "timeoutErr"               // HTTP request to publishing server timed out
	ConnectionFailedCode      ErrorCode = "connectionFailed"         // Couldn't connect to Connect
	ServerErrorCode           ErrorCode = "serverErr"                // HTTP 5xx code from publishing server
	VanityURLNotAvailableCode ErrorCode = "vanityURLNotAvailableErr" // Vanity URL already in use
	FileNotFoundCode          ErrorCode = "fileNotFound"             // A file couldn't be found; refine to a more specific code when possible
	ConfigurationNotFoundCode ErrorCode = "configurationNotFoundErr" // Could not find the named configuration
	DeploymentNotFoundCode    ErrorCode = "deploymentNotFoundErr"    // Could not find deployment to update
	AccountNotFoundCode       ErrorCode = "accountNotFoundErr"       // Account not found
	ServerURLMismatchCode     ErrorCode = "ServerURLMismatchErr"     // Redeployments must go to the same server
	DeploymentFailedCode      ErrorCode = "deployFailed"             // generic deployment failure; make more specific
)

type FileNotFoundDetails struct {
	Path string `mapstructure:"path"`
}
