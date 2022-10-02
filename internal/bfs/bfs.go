package bfs

import (
	"errors"
	"io/fs"

	"github.com/livebud/bud/framework"
	"github.com/livebud/bud/framework/app"
	"github.com/livebud/bud/framework/controller"
	"github.com/livebud/bud/framework/generator"
	"github.com/livebud/bud/framework/public"
	"github.com/livebud/bud/framework/transform/transformrt"
	"github.com/livebud/bud/framework/view"
	"github.com/livebud/bud/framework/view/dom"
	"github.com/livebud/bud/framework/view/ssr"
	"github.com/livebud/bud/framework/web"
	"github.com/livebud/bud/package/budfs"
	"github.com/livebud/bud/package/di"
	"github.com/livebud/bud/package/gomod"
	v8 "github.com/livebud/bud/package/js/v8"
	"github.com/livebud/bud/package/log"
	"github.com/livebud/bud/package/parser"
	"github.com/livebud/bud/package/svelte"
)

func Load(flag *framework.Flag, log log.Interface, module *gomod.Module) (*FS, error) {
	fsys := budfs.New(module, log)
	parser := parser.New(fsys, module)
	injector := di.New(fsys, log, module, parser)
	vm, err := v8.Load()
	if err != nil {
		return nil, err
	}
	svelteCompiler, err := svelte.Load(vm)
	if err != nil {
		return nil, err
	}
	transforms, err := transformrt.Load(svelte.NewTransformable(svelteCompiler))
	if err != nil {
		return nil, err
	}
	fsys.FileGenerator("bud/internal/app/main.go", app.New(injector, module, flag))
	fsys.FileGenerator("bud/internal/app/web/web.go", web.New(module, parser))
	fsys.FileGenerator("bud/internal/app/controller/controller.go", controller.New(injector, module, parser))
	fsys.FileGenerator("bud/internal/app/view/view.go", view.New(module, transforms, flag))
	fsys.FileGenerator("bud/internal/app/public/public.go", public.New(flag, module))
	fsys.FileGenerator("bud/view/_ssr.js", ssr.New(module, transforms.SSR))
	fsys.FileServer("bud/view", dom.New(module, transforms.DOM))
	fsys.FileServer("bud/node_modules", dom.NodeModules(module))
	fsys.DirGenerator("bud/command/generate", generator.New(fsys, flag, injector, log, module, parser))
	return &FS{fsys, module}, nil
}

type FS struct {
	fsys   *budfs.FileSystem
	module *gomod.Module
}

func (f *FS) Open(name string) (fs.File, error) {
	return f.fsys.Open(name)
}

func (f *FS) Sync(to string) error {
	if err := f.fsys.Sync(f.module, "bud/command/generate"); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	if err := f.fsys.Sync(f.module, to); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (f *FS) Change(paths ...string) {
	f.fsys.Change(paths...)
}

func (f *FS) Close() error {
	return f.fsys.Close()
}
