package sdkmap

import "go/types"

// TypeAPI records the compiler-observed representation and exported method sets
// of an SDK declaration. It shares the existing per-target export-data audit.
type TypeAPI struct {
	Identity       string   `json:"identity"`
	Underlying     string   `json:"underlying"`
	ValueMethods   []string `json:"value-methods"`
	PointerMethods []string `json:"pointer-methods"`
}

func typeAPI(value types.Type) TypeAPI {
	qualifier := func(pkg *types.Package) string { return pkg.Path() }
	methods := func(receiver types.Type) []string {
		result := []string{}
		set := types.NewMethodSet(receiver)
		for i := 0; i < set.Len(); i++ {
			method := set.At(i).Obj()
			if method.Exported() {
				result = append(result, method.Name()+" "+types.TypeString(method.Type(), qualifier))
			}
		}
		return result
	}
	return TypeAPI{
		Identity:     types.TypeString(value, qualifier),
		Underlying:   types.TypeString(value.Underlying(), qualifier),
		ValueMethods: methods(value), PointerMethods: methods(types.NewPointer(value)),
	}
}
