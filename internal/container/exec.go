package container

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
)

type ExecManager struct {
	runtime Runtime
}

func NewExecManager(rt Runtime) *ExecManager {
	return &ExecManager{runtime: rt}
}

func (m *ExecManager) Run(ctx context.Context, containerID string, cmd []string, opts ExecOptions) (*ExecResult, error) {
	slog.Debug("exec in container", "container", containerID, "cmd", cmd)
	return m.runtime.ExecInContainer(ctx, containerID, cmd, opts)
}

func (m *ExecManager) RunScript(ctx context.Context, containerID string, script string, opts ExecOptions) (*ExecResult, error) {
	slog.Debug("exec script in container", "container", containerID, "script_len", len(script))

	writeCmd := []string{"sh", "-c", fmt.Sprintf("cat > /tmp/dokrypt_script.sh << 'DOKRYPT_EOF'\n%s\nDOKRYPT_EOF\nchmod +x /tmp/dokrypt_script.sh", script)}
	if _, err := m.runtime.ExecInContainer(ctx, containerID, writeCmd, ExecOptions{}); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	return m.runtime.ExecInContainer(ctx, containerID, []string{"sh", "/tmp/dokrypt_script.sh"}, opts)
}

func (m *ExecManager) RunInteractive(ctx context.Context, containerID string, cmd []string, stdin *bytes.Buffer, opts ExecOptions) (*ExecResult, error) {
	if stdin != nil {
		opts.Stdin = stdin
		opts.Interactive = true
	}
	return m.runtime.ExecInContainer(ctx, containerID, cmd, opts)
}
