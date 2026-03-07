package plugin

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

type binaryPlugin struct {
	info *Info
}

func (p *binaryPlugin) Name() string        { return p.info.Manifest.Name }
func (p *binaryPlugin) Version() string     { return p.info.Manifest.Version }
func (p *binaryPlugin) Description() string { return p.info.Manifest.Description }
func (p *binaryPlugin) Author() string      { return p.info.Manifest.Author }

func (p *binaryPlugin) binaryPath() string {
	return filepath.Join(p.info.Path, p.info.Manifest.Name)
}

func (p *binaryPlugin) runHook(ctx context.Context, hook string, env Environment) error {
	bin := p.binaryPath()

	cmd := exec.CommandContext(ctx, bin, hook)
	cmd.Env = append(cmd.Environ(),
		"DOKRYPT_PROJECT="+env.ProjectName(),
		"DOKRYPT_HOOK="+hook,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	slog.Info("running binary plugin hook", "plugin", p.Name(), "hook", hook, "bin", bin)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("plugin %s hook %q failed: %w\nstderr: %s", p.Name(), hook, err, stderr.String())
	}
	return nil
}

func (p *binaryPlugin) OnInit(ctx context.Context, env Environment) error {
	return p.runHook(ctx, "init", env)
}

func (p *binaryPlugin) OnUp(ctx context.Context, env Environment) error {
	return p.runHook(ctx, "up", env)
}

func (p *binaryPlugin) OnDown(ctx context.Context, env Environment) error {
	return p.runHook(ctx, "down", env)
}

func (p *binaryPlugin) Commands() []*cobra.Command {
	var cmds []*cobra.Command
	bin := p.binaryPath()
	for _, def := range p.info.Manifest.Commands {
		cmdName := def.Name
		desc := def.Description
		cmds = append(cmds, &cobra.Command{
			Use:   cmdName,
			Short: desc,
			RunE: func(cmd *cobra.Command, args []string) error {
				runArgs := append([]string{cmdName}, args...)
				run := exec.CommandContext(cmd.Context(), bin, runArgs...)
				run.Stdout = cmd.OutOrStdout()
				run.Stderr = cmd.ErrOrStderr()
				run.Stdin = cmd.InOrStdin()
				return run.Run()
			},
		})
	}
	return cmds
}

func (p *binaryPlugin) Health(ctx context.Context) error {
	bin := p.binaryPath()
	cmd := exec.CommandContext(ctx, bin, "health")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("plugin %s health check failed: %w", p.Name(), err)
	}
	return nil
}
