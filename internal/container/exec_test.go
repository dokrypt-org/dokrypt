package container

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecManager(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewExecManager(mock)
	require.NotNil(t, mgr)
	assert.Equal(t, mock, mgr.runtime)
}

func TestNewExecManager_NilRuntime(t *testing.T) {
	mgr := NewExecManager(nil)
	require.NotNil(t, mgr)
	assert.Nil(t, mgr.runtime)
}

func TestExecManager_Run_Success(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return &ExecResult{
			ExitCode: 0,
			Stdout:   "hello world",
		}, nil
	}

	mgr := NewExecManager(mock)
	result, err := mgr.Run(context.Background(), "container1", []string{"echo", "hello world"}, ExecOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "hello world", result.Stdout)
	assert.Equal(t, 1, mock.calls["ExecInContainer"])
}

func TestExecManager_Run_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return nil, fmt.Errorf("container not running")
	}

	mgr := NewExecManager(mock)
	result, err := mgr.Run(context.Background(), "container1", []string{"ls"}, ExecOptions{})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "container not running")
}

func TestExecManager_Run_NonZeroExitCode(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return &ExecResult{
			ExitCode: 127,
			Stderr:   "command not found",
		}, nil
	}

	mgr := NewExecManager(mock)
	result, err := mgr.Run(context.Background(), "container1", []string{"nonexistent"}, ExecOptions{})
	require.NoError(t, err)
	assert.Equal(t, 127, result.ExitCode)
	assert.Equal(t, "command not found", result.Stderr)
}

func TestExecManager_Run_WithExecOptions(t *testing.T) {
	mock := newMockRuntime()
	var capturedOpts ExecOptions
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		capturedOpts = opts
		return &ExecResult{ExitCode: 0}, nil
	}

	mgr := NewExecManager(mock)
	opts := ExecOptions{
		WorkingDir: "/app",
		Env:        []string{"FOO=bar"},
		TTY:        true,
	}

	_, err := mgr.Run(context.Background(), "c1", []string{"ls"}, opts)
	require.NoError(t, err)
	assert.Equal(t, "/app", capturedOpts.WorkingDir)
	assert.Equal(t, []string{"FOO=bar"}, capturedOpts.Env)
	assert.True(t, capturedOpts.TTY)
}

func TestExecManager_RunScript_Success(t *testing.T) {
	mock := newMockRuntime()
	callCount := 0
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		callCount++
		if callCount == 1 {
			assert.Equal(t, "sh", cmd[0])
			assert.Equal(t, "-c", cmd[1])
			assert.Contains(t, cmd[2], "cat > /tmp/dokrypt_script.sh")
			return &ExecResult{ExitCode: 0}, nil
		}
		assert.Equal(t, []string{"sh", "/tmp/dokrypt_script.sh"}, cmd)
		return &ExecResult{ExitCode: 0, Stdout: "script output"}, nil
	}

	mgr := NewExecManager(mock)
	result, err := mgr.RunScript(context.Background(), "container1", "echo hello\necho world", ExecOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "script output", result.Stdout)
	assert.Equal(t, 2, callCount)
}

func TestExecManager_RunScript_WriteFailure(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return nil, fmt.Errorf("write failed")
	}

	mgr := NewExecManager(mock)
	result, err := mgr.RunScript(context.Background(), "container1", "echo test", ExecOptions{})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to write script")
}

func TestExecManager_RunScript_ExecuteFailure(t *testing.T) {
	mock := newMockRuntime()
	callCount := 0
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		callCount++
		if callCount == 1 {
			return &ExecResult{ExitCode: 0}, nil
		}
		return nil, fmt.Errorf("execution failed")
	}

	mgr := NewExecManager(mock)
	result, err := mgr.RunScript(context.Background(), "container1", "echo test", ExecOptions{})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "execution failed")
}

func TestExecManager_RunScript_ScriptContentEmbedded(t *testing.T) {
	mock := newMockRuntime()
	var capturedWriteCmd []string
	callCount := 0
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		callCount++
		if callCount == 1 {
			capturedWriteCmd = cmd
		}
		return &ExecResult{ExitCode: 0}, nil
	}

	script := "#!/bin/bash\nset -e\necho 'hello'"
	mgr := NewExecManager(mock)
	_, err := mgr.RunScript(context.Background(), "c1", script, ExecOptions{})
	require.NoError(t, err)
	require.Len(t, capturedWriteCmd, 3)
	assert.Contains(t, capturedWriteCmd[2], script)
	assert.Contains(t, capturedWriteCmd[2], "DOKRYPT_EOF")
}

func TestExecManager_RunInteractive_WithStdin(t *testing.T) {
	mock := newMockRuntime()
	var capturedOpts ExecOptions
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		capturedOpts = opts
		return &ExecResult{ExitCode: 0, Stdout: "interactive output"}, nil
	}

	mgr := NewExecManager(mock)
	stdinBuf := bytes.NewBufferString("input data")
	result, err := mgr.RunInteractive(context.Background(), "c1", []string{"cat"}, stdinBuf, ExecOptions{})

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "interactive output", result.Stdout)
	assert.Equal(t, stdinBuf, capturedOpts.Stdin)
	assert.True(t, capturedOpts.Interactive)
}

func TestExecManager_RunInteractive_NilStdin(t *testing.T) {
	mock := newMockRuntime()
	var capturedOpts ExecOptions
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		capturedOpts = opts
		return &ExecResult{ExitCode: 0}, nil
	}

	mgr := NewExecManager(mock)
	_, err := mgr.RunInteractive(context.Background(), "c1", []string{"ls"}, nil, ExecOptions{
		WorkingDir: "/tmp",
	})

	require.NoError(t, err)
	assert.Nil(t, capturedOpts.Stdin)
	assert.False(t, capturedOpts.Interactive)
	assert.Equal(t, "/tmp", capturedOpts.WorkingDir)
}

func TestExecManager_RunInteractive_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return nil, errMock
	}

	mgr := NewExecManager(mock)
	result, err := mgr.RunInteractive(context.Background(), "c1", []string{"cmd"}, bytes.NewBuffer(nil), ExecOptions{})
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestExecManager_RunInteractive_PreservesExistingOpts(t *testing.T) {
	mock := newMockRuntime()
	var capturedOpts ExecOptions
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		capturedOpts = opts
		return &ExecResult{ExitCode: 0}, nil
	}

	mgr := NewExecManager(mock)
	stdinBuf := bytes.NewBufferString("data")
	opts := ExecOptions{
		TTY:        true,
		Env:        []string{"KEY=val"},
		WorkingDir: "/app",
	}

	_, err := mgr.RunInteractive(context.Background(), "c1", []string{"sh"}, stdinBuf, opts)
	require.NoError(t, err)
	assert.True(t, capturedOpts.TTY)
	assert.Equal(t, []string{"KEY=val"}, capturedOpts.Env)
	assert.Equal(t, "/app", capturedOpts.WorkingDir)
	assert.True(t, capturedOpts.Interactive) // Set by RunInteractive
	assert.Equal(t, stdinBuf, capturedOpts.Stdin)
}
