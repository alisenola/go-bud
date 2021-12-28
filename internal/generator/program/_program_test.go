package program_test

import (
	"testing"

	"github.com/matryer/is"
	"gitlab.com/mnm/bud/gen"
	"gitlab.com/mnm/bud/internal/di"
	"gitlab.com/mnm/bud/internal/generator/program"
	"gitlab.com/mnm/bud/internal/tester"
)

const goMod = `
module app.com

require (
	gitlab.com/mnm/bud v0.0.0
)
`

func TestNoCommand(t *testing.T) {
	is := is.New(t)
	tr := tester.New(t)
	tr.Files(map[string]string{
		"go.mod": goMod,
	})
	genFS := tr.GenFS()
	genFS.Add(map[string]gen.Generator{
		"bud/program/program.go": gen.FileGenerator(&program.Generator{
			Module:   tr.Module(),
			Injector: tr.Injector(di.Map{}),
		}),
	})
	is.NoErr(tr.Sync())
	is.Equal(false, tr.Exists("bud/program/program.go"))
}
