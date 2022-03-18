package bud

import (
	"context"
	"io"
	"net"
	"os/exec"

	"gitlab.com/mnm/bud/pkg/gomod"
	"gitlab.com/mnm/bud/pkg/socket"
)

type Project struct {
	Module *gomod.Module
	Flag   Flag
	Env    Env
	Stdout io.Writer
	Stderr io.Writer
}

func (p *Project) args(command ...string) []string {
	return append(command, p.Flag.List()...)
}

func (p *Project) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, p.Module.Directory("bud", "cli"), args...)
	cmd.Dir = p.Module.Directory()
	cmd.Env = p.Env.List()
	cmd.Stderr = p.Stderr
	cmd.Stdout = p.Stdout
	return cmd
}

func (p *Project) Executor(ctx context.Context, args ...string) *exec.Cmd {
	return p.command(ctx, p.args(args...)...)
}

// Execute a custom command
func (p *Project) Execute(ctx context.Context, args ...string) error {
	cmd := p.Executor(ctx, args...)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) Builder(ctx context.Context) *exec.Cmd {
	return p.command(ctx, p.args("build")...)
}

func (p *Project) Build(ctx context.Context) error {
	cmd := p.Builder(ctx)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (p *Project) Runner(ctx context.Context, listener net.Listener) (*exec.Cmd, error) {
	// Pass the socket through
	files, env, err := socket.Files(listener)
	if err != nil {
		return nil, err
	}
	cmd := p.command(ctx, p.args("run")...)
	cmd.Env = append(p.Env.List(), string(env))
	cmd.Stdout = p.Stdout
	cmd.Stderr = p.Stderr
	cmd.ExtraFiles = append(cmd.ExtraFiles, files...)
	return cmd, nil
}

func (p *Project) Run(ctx context.Context, listener net.Listener) error {
	cmd, err := p.Runner(ctx, listener)
	if err != nil {
		return err
	}
	return cmd.Run()
}
