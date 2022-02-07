package public_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/matryer/is"
	"gitlab.com/mnm/bud/internal/fscache"
	"gitlab.com/mnm/bud/internal/test"
	"gitlab.com/mnm/bud/pkg/modcache"
)

func TestEmpty(t *testing.T) {
	is := is.New(t)
	generator := test.Generator(t)
	fsCache := fscache.New()
	app, err := generator.Generate(fsCache)
	is.NoErr(err)
	is.Equal(false, app.Exists("bud/public/public.go"))
}

// Pulled from: https://github.com/mathiasbynens/small
// Built with: xxd -i small.ico
var favicon = []byte{
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00,
	0x18, 0x00, 0x30, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00,
}

func TestFavicon(t *testing.T) {
	is := is.New(t)
	generator := test.Generator(t)
	generator.Files["public/favicon.ico"] = favicon
	fsCache := fscache.New()
	app, err := generator.Generate(fsCache)
	is.NoErr(err)
	is.Equal(true, app.Exists("bud/public/public.go")) // bud/public/public.go should exist
	is.Equal(true, app.Exists("bud/main.go"))          // bud/main.go should exist
	server, err := app.Start()
	is.NoErr(err)
	defer server.Close()
	res, err := server.Get("/favicon.ico")
	is.NoErr(err)
	defer res.Body.Close()
	is.Equal(200, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.True(bytes.Equal(favicon, body))
}

func TestNested(t *testing.T) {
	is := is.New(t)
	generator := test.Generator(t)
	css := []byte(`* { box-sizing: border-box; }`)
	generator.Files["public/normalize/normalize.css"] = css
	fsCache := fscache.New()
	app, err := generator.Generate(fsCache)
	is.NoErr(err)
	is.Equal(true, app.Exists("bud/public/public.go"))
	is.Equal(true, app.Exists("bud/main.go"))
	server, err := app.Start()
	is.NoErr(err)
	defer server.Close()
	res, err := server.Get("/normalize/normalize.css")
	is.NoErr(err)
	defer res.Body.Close()
	is.Equal(200, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.True(bytes.Equal(css, body))
}

func TestPlugin(t *testing.T) {
	is := is.New(t)
	generator := test.Generator(t)
	preflight := `/* tailwind */`
	generator.Modules = map[string]modcache.Files{
		"gitlab.com/mnm/bud-tailwind@v0.0.1": modcache.Files{
			"public/tailwind/preflight.css": preflight,
		},
	}
	fsCache := fscache.New()
	app, err := generator.Generate(fsCache)
	is.NoErr(err)
	is.Equal(true, app.Exists("bud/public/public.go"))
	is.Equal(true, app.Exists("bud/main.go"))
	server, err := app.Start()
	is.NoErr(err)
	defer server.Close()
	res, err := server.Get("/tailwind/preflight.css")
	is.NoErr(err)
	defer res.Body.Close()
	is.Equal(200, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.Equal(preflight, string(body))
}

func TestAppPluginOverlap(t *testing.T) {
	t.SkipNow()
}

func TestPluginPluginOverlap(t *testing.T) {
	t.SkipNow()
}
