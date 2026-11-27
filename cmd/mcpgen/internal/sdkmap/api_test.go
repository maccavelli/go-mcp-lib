package sdkmap

import (
	"strings"
	"testing"
)

func TestRepairA09CompilerAPI(t *testing.T) {
	pkg, _ := fixturePackage(t, "example.invalid/api", `package api
type Box[T any] struct { Value T }
func (b Box[T]) Get() T { return b.Value }
func (b *Box[T]) Set(value T) { b.Value = value }
type StringBox = Box[string]
type Outer struct { StringBox }
`)
	api := typeAPI(pkg.Scope().Lookup("Outer").Type())
	if api.Identity != "example.invalid/api.Outer" || !strings.Contains(api.Underlying, "StringBox") || len(api.ValueMethods) != 1 || len(api.PointerMethods) != 2 {
		t.Errorf("FAIL compiler API method sets: %+v", api)
	}
	if !strings.Contains(strings.Join(api.ValueMethods, "\n"), "Get func() string") || !strings.Contains(strings.Join(api.PointerMethods, "\n"), "Set func(value string)") {
		t.Errorf("FAIL compiler API generic/promoted signature: %+v", api)
	}
}
