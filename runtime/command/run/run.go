package run

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/mnm/bud/package/watcher"

	"gitlab.com/mnm/bud/package/hot"

	"golang.org/x/sync/errgroup"

	"gitlab.com/mnm/bud/package/socket"
	"gitlab.com/mnm/bud/runtime/bud"
)

type Command struct {
	Flag    *bud.Flag
	Project *bud.Project
	Port    string
}

func (c *Command) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	// Initialize the hot server
	var hotServer *hot.Server
	if c.Flag.Hot {
		hotServer = hot.New()
		// Start the hot reload server
		eg.Go(func() error { return c.startHot(ctx, hotServer) })
	}
	// Start the web server
	eg.Go(func() error { return c.startApp(ctx, hotServer) })
	return eg.Wait()
}

func (c *Command) startApp(ctx context.Context, hotServer *hot.Server) error {
	app, err := c.Project.Compile(ctx, c.Flag)
	if err != nil {
		return err
	}
	listener, err := socket.Load(c.Port)
	if err != nil {
		return err
	}
	process, err := app.Start(ctx, listener)
	if err != nil {
		return err
	}
	defer process.Close()
	// Start watching
	if err := watcher.Watch(ctx, ".", func(path string) error {
		switch filepath.Ext(path) {
		// Re-compile the app and restart the Go server
		case ".go":
			fmt.Println("restarting", path)
			// Trigger a reload if there's a hot reload server configured
			if hotServer != nil {
				hotServer.Reload("*")
			}
			now := time.Now()
			if err := process.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "error closing process", err)
				return nil
			}
			app, err := c.Project.Compile(ctx, c.Flag)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error compiling", err.Error())
				return nil
			}
			process, err = app.Start(ctx, listener)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error starting", err.Error())
				return nil
			}
			fmt.Println("restarted", time.Since(now))
			return nil
		// Hot reload the page
		default:
			fmt.Println("reloading", path)
			// Trigger a reload if there's a hot reload server configured
			if hotServer != nil {
				hotServer.Reload("*")
			}
			return nil
		}
	}); err != nil {
		return err
	}
	return process.Wait()
}

func (c *Command) startHot(ctx context.Context, hotServer *hot.Server) error {
	return hotServer.ListenAndServe(ctx, ":35729")
}
