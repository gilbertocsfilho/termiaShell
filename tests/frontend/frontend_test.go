package frontend_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
)

func script(t *testing.T, vm *goja.Runtime, name string) {
	t.Helper()
	b, e := os.ReadFile(filepath.Join("..", "..", name))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = vm.RunScript(name, string(b)); e != nil {
		t.Fatal(e)
	}
}
func harness(t *testing.T) (*goja.Runtime, func()) {
	vm := goja.New()
	type testCase struct {
		name     string
		callback goja.Value
	}
	cases := []testCase{}
	vm.Set("test", func(name string, callback goja.Value) {
		cases = append(cases, testCase{name, callback})
	})
	run := func() {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				fn, ok := goja.AssertFunction(c.callback)
				if !ok {
					t.Fatal("callback inválido")
				}
				result, e := fn(goja.Undefined())
				if e != nil {
					t.Fatal(e)
				}
				if p, ok := result.Export().(*goja.Promise); ok {
					if p.State() != goja.PromiseStateFulfilled {
						t.Fatalf("Teste assíncrono não concluído: %v", p.Result())
					}
				}
			})
		}
	}
	vm.Set("atob", func(encoded string) (string, error) {
		b, e := base64.StdEncoding.DecodeString(encoded)
		return string(b), e
	})
	return vm, run
}
func TestThemeBehavior(t *testing.T) {
	vm, run := harness(t)
	script(t, vm, "frontend/themes.js")
	script(t, vm, "tests/frontend/themes.test.js")
	run()
}
func TestApplicationThemeAndSecurity(t *testing.T) {
	vm, run := harness(t)
	script(t, vm, "frontend/themes.js")
	script(t, vm, "frontend/exit.js")
	script(t, vm, "tests/frontend/dom-fixture.js")
	script(t, vm, "frontend/app.js")
	script(t, vm, "tests/frontend/app.test.js")
	run()
}
func TestExitDialog(t *testing.T) {
	vm, run := harness(t)
	script(t, vm, "frontend/exit.js")
	script(t, vm, "tests/frontend/exit.test.js")
	run()
}
