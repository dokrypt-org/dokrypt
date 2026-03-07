package common

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewError_FieldsAreSet(t *testing.T) {
	err := NewError(ErrConfigParseFailed, "could not parse config")

	require.NotNil(t, err)
	assert.Equal(t, ErrConfigParseFailed, err.Code)
	assert.Equal(t, "could not parse config", err.Message)
	assert.Nil(t, err.Cause)
	assert.NotNil(t, err.Context)
	assert.Empty(t, err.Suggestion)
}

func TestNewError_ContextMapIsInitialized(t *testing.T) {
	err := NewError(ErrChainStartFailed, "chain failed")

	require.NotNil(t, err.Context)
	err.Context["key"] = "value"
	assert.Equal(t, "value", err.Context["key"])
}

func TestNewError_ImplementsErrorInterface(t *testing.T) {
	var e error = NewError(ErrServiceStartFailed, "service failed")
	assert.NotNil(t, e)
}

func TestWrap_WrapsUnderlying(t *testing.T) {
	cause := errors.New("underlying cause")
	err := Wrap(cause, ErrConfigParseFailed, "wrapped message")

	require.NotNil(t, err)
	assert.Equal(t, ErrConfigParseFailed, err.Code)
	assert.Equal(t, "wrapped message", err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.NotNil(t, err.Context)
}

func TestWrap_UnwrapReturnsUnderlying(t *testing.T) {
	cause := errors.New("root cause")
	wrapped := Wrap(cause, ErrChainRPCFailed, "rpc failed")

	assert.True(t, errors.Is(wrapped, cause))
}

func TestWrap_WithNilCause(t *testing.T) {
	err := Wrap(nil, ErrConfigNotFound, "not found")

	require.NotNil(t, err)
	assert.Nil(t, err.Cause)
	assert.Nil(t, err.Unwrap())
}

func TestErrorString_WithoutCause(t *testing.T) {
	err := NewError(ErrConfigValidation, "validation failed")
	got := err.Error()

	assert.Equal(t, "[CONFIG_VALIDATION_FAILED] validation failed", got)
}

func TestErrorString_WithCause(t *testing.T) {
	cause := errors.New("disk full")
	err := Wrap(cause, ErrSnapshotSaveFailed, "snapshot save failed")
	got := err.Error()

	assert.Equal(t, "[SNAPSHOT_SAVE_FAILED] snapshot save failed: disk full", got)
}

func TestErrorString_WithWrappedDokryptError(t *testing.T) {
	inner := NewError(ErrChainStartFailed, "chain did not start")
	outer := Wrap(inner, ErrServiceStartFailed, "service could not start")

	got := outer.Error()
	assert.Contains(t, got, "SERVICE_START_FAILED")
	assert.Contains(t, got, "service could not start")
	assert.Contains(t, got, "CHAIN_START_FAILED")
}

func TestWithSuggestion_SetsSuggestion(t *testing.T) {
	err := NewError(ErrConfigNotFound, "config not found").
		WithSuggestion("run 'dokrypt init' to create a config")

	assert.Equal(t, "run 'dokrypt init' to create a config", err.Suggestion)
}

func TestWithSuggestion_Chaining(t *testing.T) {
	err := NewError(ErrAuthFailed, "auth failed").
		WithSuggestion("check your credentials").
		WithContext("user", "alice")

	assert.Equal(t, "check your credentials", err.Suggestion)
	assert.Equal(t, "alice", err.Context["user"])
}

func TestWithContext_AddsKeyValue(t *testing.T) {
	err := NewError(ErrContainerNotFound, "container missing").
		WithContext("container_id", "abc123").
		WithContext("runtime", "docker")

	assert.Equal(t, "abc123", err.Context["container_id"])
	assert.Equal(t, "docker", err.Context["runtime"])
}

func TestWithContext_SupportsAnyValue(t *testing.T) {
	err := NewError(ErrPluginLoadFailed, "plugin failed").
		WithContext("attempts", 3).
		WithContext("enabled", true).
		WithContext("data", []string{"a", "b"})

	assert.Equal(t, 3, err.Context["attempts"])
	assert.Equal(t, true, err.Context["enabled"])
	assert.Equal(t, []string{"a", "b"}, err.Context["data"])
}

func TestIsDokryptError_WithDokryptError(t *testing.T) {
	err := NewError(ErrNetworkCreateFailed, "network failed")
	assert.True(t, IsDokryptError(err))
}

func TestIsDokryptError_WithWrappedDokryptError(t *testing.T) {
	de := NewError(ErrContainerPullFailed, "pull failed")
	wrapped := fmt.Errorf("outer: %w", de)
	assert.True(t, IsDokryptError(wrapped))
}

func TestIsDokryptError_WithPlainError(t *testing.T) {
	err := errors.New("plain error")
	assert.False(t, IsDokryptError(err))
}

func TestIsDokryptError_WithNil(t *testing.T) {
	assert.False(t, IsDokryptError(nil))
}

func TestAsDokryptError_WithDokryptError(t *testing.T) {
	original := NewError(ErrDependencyCycle, "cycle detected")
	de, ok := AsDokryptError(original)

	require.True(t, ok)
	require.NotNil(t, de)
	assert.Equal(t, ErrDependencyCycle, de.Code)
}

func TestAsDokryptError_WithWrappedDokryptError(t *testing.T) {
	original := NewError(ErrPortConflict, "port conflict")
	wrapped := fmt.Errorf("wrapper: %w", original)

	de, ok := AsDokryptError(wrapped)
	require.True(t, ok)
	require.NotNil(t, de)
	assert.Equal(t, ErrPortConflict, de.Code)
	assert.Equal(t, "port conflict", de.Message)
}

func TestAsDokryptError_WithPlainError(t *testing.T) {
	err := errors.New("not a dokrypt error")
	de, ok := AsDokryptError(err)

	assert.False(t, ok)
	assert.Nil(t, de)
}

func TestAsDokryptError_WithNil(t *testing.T) {
	de, ok := AsDokryptError(nil)
	assert.False(t, ok)
	assert.Nil(t, de)
}

func TestErrorCode_WithDokryptError(t *testing.T) {
	err := NewError(ErrAuthTokenExpired, "token expired")
	assert.Equal(t, ErrAuthTokenExpired, ErrorCode(err))
}

func TestErrorCode_WithWrappedDokryptError(t *testing.T) {
	de := NewError(ErrAuthUnauthorized, "unauthorized")
	wrapped := fmt.Errorf("http layer: %w", de)
	assert.Equal(t, ErrAuthUnauthorized, ErrorCode(wrapped))
}

func TestErrorCode_WithPlainError(t *testing.T) {
	err := errors.New("plain")
	assert.Equal(t, "", ErrorCode(err))
}

func TestErrorCode_WithNil(t *testing.T) {
	assert.Equal(t, "", ErrorCode(nil))
}

func TestErrorCodeConstants(t *testing.T) {
	assert.Contains(t, ErrConfigParseFailed, "CONFIG")
	assert.Contains(t, ErrChainStartFailed, "CHAIN")
	assert.Contains(t, ErrServiceStartFailed, "SERVICE")
	assert.Contains(t, ErrContainerCreateFailed, "CONTAINER")
	assert.Contains(t, ErrSnapshotSaveFailed, "SNAPSHOT")
	assert.Contains(t, ErrAuthFailed, "AUTH")
	assert.Contains(t, ErrAPIRequestFailed, "API")
	assert.Contains(t, ErrPluginLoadFailed, "PLUGIN")
	assert.Contains(t, ErrNetworkCreateFailed, "NETWORK")
}

func TestDokryptError_Unwrap_DeepChain(t *testing.T) {
	root := errors.New("root")
	mid := Wrap(root, ErrContainerExecFailed, "exec failed")
	top := Wrap(mid, ErrServiceStartFailed, "service failed")

	assert.True(t, errors.Is(top, root))
	assert.True(t, errors.Is(top, mid))
}

func TestDokryptError_AsInChain(t *testing.T) {
	inner := NewError(ErrChainForkFailed, "fork failed")
	outer := fmt.Errorf("level2: %w", fmt.Errorf("level1: %w", inner))

	var de *DokryptError
	require.True(t, errors.As(outer, &de))
	assert.Equal(t, ErrChainForkFailed, de.Code)
}

func TestWithContext_NilContextMapIsInitialized(t *testing.T) {
	err := &DokryptError{
		Code:    ErrServiceNotFound,
		Message: "service not found",
		Context: nil, // explicitly nil
	}

	err.WithContext("container", "nginx")

	require.NotNil(t, err.Context)
	assert.Equal(t, "nginx", err.Context["container"])
}

func TestWrap_DokryptErrorCauseUnwrapsCorrectly(t *testing.T) {
	inner := NewError(ErrChainRPCFailed, "rpc down")
	outer := Wrap(inner, ErrServiceHealthFailed, "health check failed")

	var innerDE *DokryptError
	require.True(t, errors.As(outer.Unwrap(), &innerDE))
	assert.Equal(t, ErrChainRPCFailed, innerDE.Code)
}

func TestNewError_EmptyCodeAndMessage(t *testing.T) {
	err := NewError("", "")
	assert.Equal(t, "[] ", err.Error())
}

func TestErrorCode_WrapNilError(t *testing.T) {
	err := Wrap(nil, ErrPluginNotFound, "plugin missing")
	assert.Equal(t, ErrPluginNotFound, ErrorCode(err))
}

func TestWrap_WithSuggestionAndContext(t *testing.T) {
	cause := errors.New("disk full")
	err := Wrap(cause, ErrSnapshotSaveFailed, "save failed").
		WithSuggestion("free disk space").
		WithContext("path", "/data/snapshot")

	assert.Equal(t, "free disk space", err.Suggestion)
	assert.Equal(t, "/data/snapshot", err.Context["path"])
	assert.True(t, errors.Is(err, cause))
}
