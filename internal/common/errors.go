package common

import (
	"errors"
	"fmt"
)

const (
	ErrCodeConfig    = "CONFIG"
	ErrCodeChain     = "CHAIN"
	ErrCodeService   = "SERVICE"
	ErrCodeContainer = "CONTAINER"
	ErrCodeSnapshot  = "SNAPSHOT"
	ErrCodeAuth      = "AUTH"
	ErrCodeAPI       = "API"
	ErrCodePlugin    = "PLUGIN"
	ErrCodeNetwork   = "NETWORK"
)

const (
	ErrConfigParseFailed     = "CONFIG_PARSE_FAILED"
	ErrConfigValidation      = "CONFIG_VALIDATION_FAILED"
	ErrConfigNotFound        = "CONFIG_NOT_FOUND"
	ErrConfigInterpolation   = "CONFIG_INTERPOLATION_FAILED"
	ErrChainStartFailed      = "CHAIN_START_FAILED"
	ErrChainStopFailed       = "CHAIN_STOP_FAILED"
	ErrChainHealthFailed     = "CHAIN_HEALTH_FAILED"
	ErrChainForkFailed       = "CHAIN_FORK_FAILED"
	ErrChainRPCFailed        = "CHAIN_RPC_FAILED"
	ErrChainNotFound         = "CHAIN_NOT_FOUND"
	ErrServiceStartFailed    = "SERVICE_START_FAILED"
	ErrServiceStopFailed     = "SERVICE_STOP_FAILED"
	ErrServiceHealthFailed   = "SERVICE_HEALTH_FAILED"
	ErrServiceNotFound       = "SERVICE_NOT_FOUND"
	ErrContainerCreateFailed = "CONTAINER_CREATE_FAILED"
	ErrContainerStartFailed  = "CONTAINER_START_FAILED"
	ErrContainerStopFailed   = "CONTAINER_STOP_FAILED"
	ErrContainerNotFound     = "CONTAINER_NOT_FOUND"
	ErrContainerPullFailed   = "CONTAINER_PULL_FAILED"
	ErrContainerExecFailed   = "CONTAINER_EXEC_FAILED"
	ErrSnapshotSaveFailed    = "SNAPSHOT_SAVE_FAILED"
	ErrSnapshotRestoreFailed = "SNAPSHOT_RESTORE_FAILED"
	ErrSnapshotNotFound      = "SNAPSHOT_NOT_FOUND"
	ErrAuthFailed            = "AUTH_FAILED"
	ErrAuthTokenExpired      = "AUTH_TOKEN_EXPIRED"
	ErrAuthUnauthorized      = "AUTH_UNAUTHORIZED"
	ErrAPIRequestFailed      = "API_REQUEST_FAILED"
	ErrPluginLoadFailed      = "PLUGIN_LOAD_FAILED"
	ErrPluginNotFound        = "PLUGIN_NOT_FOUND"
	ErrNetworkCreateFailed   = "NETWORK_CREATE_FAILED"
	ErrDependencyCycle       = "DEPENDENCY_CYCLE"
	ErrPortConflict          = "PORT_CONFLICT"
)

type DokryptError struct {
	Code       string         // Machine-readable error code, e.g. "CHAIN_START_FAILED"
	Message    string         // Human-readable error message
	Suggestion string         // Actionable suggestion for the user
	Cause      error          // Wrapped underlying error
	Context    map[string]any // Additional context for debugging
}

func (e *DokryptError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DokryptError) Unwrap() error {
	return e.Cause
}

func NewError(code, message string) *DokryptError {
	return &DokryptError{
		Code:    code,
		Message: message,
		Context: make(map[string]any),
	}
}

func Wrap(err error, code, message string) *DokryptError {
	return &DokryptError{
		Code:    code,
		Message: message,
		Cause:   err,
		Context: make(map[string]any),
	}
}

func (e *DokryptError) WithSuggestion(suggestion string) *DokryptError {
	e.Suggestion = suggestion
	return e
}

func (e *DokryptError) WithContext(key string, value any) *DokryptError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

func IsDokryptError(err error) bool {
	var de *DokryptError
	return errors.As(err, &de)
}

func AsDokryptError(err error) (*DokryptError, bool) {
	var de *DokryptError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}

func ErrorCode(err error) string {
	if de, ok := AsDokryptError(err); ok {
		return de.Code
	}
	return ""
}
