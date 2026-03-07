package plugin

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/spf13/cobra"
)

type containerPlugin struct {
	info *Info
}

func (p *containerPlugin) Name() string        { return p.info.Manifest.Name }
func (p *containerPlugin) Version() string     { return p.info.Manifest.Version }
func (p *containerPlugin) Description() string { return p.info.Manifest.Description }
func (p *containerPlugin) Author() string      { return p.info.Manifest.Author }

func (p *containerPlugin) image() string {
	return p.info.Manifest.Container.Image
}

func (p *containerPlugin) runHook(ctx context.Context, hook string, env Environment) error {
	img := p.image()
	if img == "" {
		slog.Debug("no container image for plugin, skipping hook", "plugin", p.Name(), "hook", hook)
		return nil
	}

	args := []string{"run", "--rm"}

	args = append(args, "-e", "DOKRYPT_PROJECT="+env.ProjectName())
	args = append(args, "-e", "DOKRYPT_HOOK="+hook)

	for k, v := range p.info.Manifest.Container.Environment {
		args = append(args, "-e", k+"="+v)
	}

	args = append(args, img, hook)

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	slog.Info("running plugin hook", "plugin", p.Name(), "hook", hook, "image", img)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("plugin %s hook %q failed: %w\nstderr: %s", p.Name(), hook, err, stderr.String())
	}
	return nil
}

func (p *containerPlugin) OnInit(ctx context.Context, env Environment) error {
	return p.runHook(ctx, "init", env)
}

func (p *containerPlugin) OnUp(ctx context.Context, env Environment) error {
	return p.runHook(ctx, "up", env)
}

func (p *containerPlugin) OnDown(ctx context.Context, env Environment) error {
	return p.runHook(ctx, "down", env)
}

func (p *containerPlugin) Commands() []*cobra.Command {
	var cmds []*cobra.Command
	img := p.image()
	for _, def := range p.info.Manifest.Commands {
		cmdName := def.Name
		desc := def.Description
		cmds = append(cmds, &cobra.Command{
			Use:   cmdName,
			Short: desc,
			RunE: func(cmd *cobra.Command, args []string) error {
				if img == "" {
					slog.Warn("no container image for plugin, cannot run command", "plugin", p.Name(), "command", cmdName)
					return fmt.Errorf("plugin %s has no container image configured", p.Name())
				}

				dockerArgs := []string{"run", "--rm"}

				for k, v := range p.info.Manifest.Container.Environment {
					dockerArgs = append(dockerArgs, "-e", k+"="+v)
				}

				dockerArgs = append(dockerArgs, img, cmdName)
				dockerArgs = append(dockerArgs, args...)

				run := exec.CommandContext(cmd.Context(), "docker", dockerArgs...)
				run.Stdout = cmd.OutOrStdout()
				run.Stderr = cmd.ErrOrStderr()
				run.Stdin = cmd.InOrStdin()

				return run.Run()
			},
		})
	}
	return cmds
}

func (p *containerPlugin) Health(ctx context.Context) error {
	img := p.image()
	if img == "" {
		return nil // no container configured, nothing to health-check
	}

	cmd := exec.CommandContext(ctx, "docker", "inspect", "--type=image", img)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("plugin %s: container image %q not available: %w", p.Name(), img, err)
	}
	return nil
}
