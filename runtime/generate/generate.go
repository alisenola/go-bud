package generate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gitlab.com/mnm/bud/pkg/hot"
	"gitlab.com/mnm/bud/pkg/watcher"

	"golang.org/x/sync/errgroup"

	"gitlab.com/mnm/bud/pkg/log/console"
	"gitlab.com/mnm/bud/pkg/socket"

	"gitlab.com/mnm/bud/internal/fscache"
	"gitlab.com/mnm/bud/internal/gobin"
	"gitlab.com/mnm/bud/package/commander"
	"gitlab.com/mnm/bud/pkg/gen"
	"gitlab.com/mnm/bud/pkg/generator"
)

type Command struct {
	Dir        string
	Port       int
	Embed      bool
	Hot        bool
	Minify     bool
	Generators map[string]gen.Generator
	Args       []string
}

func (c *Command) Parse(ctx context.Context) error {
	cli := commander.New("bud")
	cli.Run(c.Run)
	return cli.Parse(ctx, []string{})
}

func (c *Command) Run(ctx context.Context) error {
	// TODO: Use the passed in generators (c.Generators)
	// TODO: Enable file caching
	fsCache := fscache.New()
	generator, err := generator.Load(c.Dir, generator.WithFSCache(fsCache))
	if err != nil {
		return err
	}
	// Load the socket up, this should come from LISTENER_FDS
	listener, err := socket.Load(":3000")
	if err != nil {
		return err
	}
	// Start listening
	hotListener, err := socket.Listen(":35729")
	if err != nil {
		fmt.Println("load error", err)
		return err
	}
	hotServer := hot.New()
	// Setup the command runner
	runner := &Runner{
		Generator: generator,
		Listener:  listener,
		Args:      c.Args,
		Dir:       c.Dir,
	}
	eg, ctx := errgroup.WithContext(ctx)
	// Start running, but don't exit if it fails
	if err := runner.Start(ctx); err != nil {
		console.Error("error starting app > %s", err)
	}
	// Watch for file changes
	eg.Go(func() error {
		return watcher.Watch(ctx, c.Dir, func(path string) error {
			fsCache.Update(path)
			// Hot reload non-Go files
			if filepath.Ext(path) != ".go" {
				hotServer.Reload("*")
				return nil
			}
			fmt.Println("restarting", path)
			// Restart the process for Go files
			if err := runner.Restart(ctx); err != nil {
				console.Error("error restarting app > %s", err)
			}
			// Then hot reload
			hotServer.Reload("*")
			return nil
		})
	})

	// Start the hot reload server
	eg.Go(func() error {
		if err := hotServer.Serve(hotListener); err != nil {
			fmt.Println("serve error", err)
			return err
		}
		return nil
	})

	return eg.Wait()
}

type Runner struct {
	Generator *generator.Generator
	Listener  net.Listener
	Args      []string
	Dir       string

	// Existing command
	cmd *exec.Cmd
}

func (r *Runner) Start(ctx context.Context) error {
	if err := r.Generator.Generate(ctx); err != nil {
		return err
	}
	mainPath := filepath.Join(r.Dir, "bud", "main.go")
	// Check to see if we generated a main.go
	if _, err := os.Stat(mainPath); err != nil {
		return err
	}
	// Build into bud/main
	binPath := filepath.Join(r.Dir, "bud", "main")
	if err := gobin.Build(ctx, r.Dir, mainPath, binPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		// TODO: improve the welcome server
		return http.Serve(r.Listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Welcome Server!\n"))
		}))
	}
	files, env, err := socket.Files(r.Listener)
	if err != nil {
		return err
	}
	// Run the app
	cmd := exec.CommandContext(ctx, binPath, r.Args...)
	cmd.Env = append(os.Environ(), string(env))
	cmd.ExtraFiles = files
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = r.Dir
	if err := cmd.Start(); err != nil {
		return err
	}
	r.cmd = cmd
	return nil
}

func (r *Runner) Stop() error {
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	p := r.cmd.Process
	if err := p.Signal(os.Interrupt); err != nil {
		p.Kill()
	}
	if err := r.cmd.Wait(); err != nil {
		// TODO: figure out if there's a cleaner way to shutdown
		if isExitStatus(err) || isWaitAlreadyCalled(err) {
			r.cmd = nil
			return nil
		}
		return err
	}
	r.cmd = nil
	return nil
}

func (r *Runner) Restart(ctx context.Context) error {
	if err := r.Stop(); err != nil {
		return err
	}
	return r.Start(ctx)
}

func isExitStatus(err error) bool {
	return strings.Contains(err.Error(), "exit status ")
}

func isWaitAlreadyCalled(err error) bool {
	return strings.Contains(err.Error(), "Wait was already called")
}
