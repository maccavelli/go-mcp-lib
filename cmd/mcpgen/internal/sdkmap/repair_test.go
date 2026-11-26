package sdkmap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const aliasFixture = `package mcp
type Server struct{}
// Deprecated: legacy type.
type Legacy string
// Clean mentions Deprecated: as text rather than a deprecation paragraph.
type Clean struct { Name string }
type Recursive struct { Next *Recursive }
type Box[T any] struct { Value T }
type SafeBox = Box[Clean]
type RuntimeBox = Box[*Server]
type LegacyAlias = Legacy
type PointerLeak struct{}
func (*PointerLeak) Expose() *Server { return nil }
type PromotedLeak struct { PointerLeak }
type Sealed interface { sealed() }
type InterfaceLeak struct { Value Sealed }
type FieldLeak struct {
    // Deprecated: legacy member.
    Old string
}
type EmbeddedLeak struct {
    // Deprecated: embedded legacy member.
    Clean
}
type Generic[A, B any] struct { Left A; Right B }
// Deprecated: legacy method.
func (*Generic[A, B]) Old() {}
type GenericMethod = Generic[string, int]
type MethodLeak struct{}
// Deprecated: legacy method.
func (*MethodLeak) Old() {}
type PromotedDeprecated struct { MethodLeak }
type FunctionLeak func() *Server
type Containers struct { Items map[string][]*Server }
`

func fixturePackage(t *testing.T, path, source string) (*types.Package, map[string]bool) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("FAIL parse type-audit fixture: %v", err)
	}
	configuration := types.Config{}
	pkg, err := configuration.Check(path, fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("FAIL type-check audit fixture: %v", err)
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "fixture.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	comments, err := deprecationComments(packageMetadata{Dir: directory, GoFiles: []string{"fixture.go"}})
	if err != nil {
		t.Fatalf("FAIL source comment join: %v", err)
	}
	return pkg, comments
}

func TestRepairA09AliasClosure(t *testing.T) {
	pkg, comments := fixturePackage(t, sdkPackage, aliasFixture)
	for _, name := range []string{"Legacy", "FieldLeak.Old", "EmbeddedLeak.Clean", "Generic.Old", "MethodLeak.Old"} {
		if !comments[name] {
			t.Errorf("FAIL deprecation source association %s", name)
		}
	}
	if comments["Clean"] {
		t.Error("FAIL ordinary documentation text became a deprecation marker")
	}
	cases := map[string]string{
		"Clean": "", "Recursive": "", "SafeBox": "",
		"Legacy": "deprecated", "LegacyAlias": "deprecated",
		"RuntimeBox": "SDK runtime type Server", "PointerLeak": "SDK runtime type Server",
		"PromotedLeak": "SDK runtime type Server", "InterfaceLeak": "sealed SDK interface",
		"FieldLeak": "deprecated field FieldLeak.Old", "EmbeddedLeak": "deprecated field EmbeddedLeak.Clean",
		"GenericMethod": "deprecated method Generic.Old", "PromotedDeprecated": "deprecated method MethodLeak.Old",
		"FunctionLeak": "SDK runtime type Server", "Containers": "SDK runtime type Server",
		"Box": "uninstantiated generic",
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			got := aliasViolation(pkg.Scope().Lookup(name).Type(), comments)
			if want == "" && got != "" || want != "" && !strings.Contains(got, want) {
				t.Errorf("FAIL alias closure %s: got %q; want %q", name, got, want)
			}
		})
	}
	t.Run("unrelated-package-name", func(t *testing.T) {
		other, _ := fixturePackage(t, "example.invalid/other", `package other; type FieldLeak struct { Old string }`)
		if reason := aliasViolation(other.Scope().Lookup("FieldLeak").Type(), comments); reason != "" {
			t.Errorf("FAIL deprecation attached to a different package: %s", reason)
		}
	})
}

func TestRepairA09PinnedInventory(t *testing.T) {
	result, err := Build("2026-07-28", sdkDir(t), "v1.8.0", []string{"Meta", "CreateMessageResult", "ListRootsResult"})
	if err != nil {
		t.Fatalf("FAIL pinned inventory: %v", err)
	}
	for _, name := range result.Deprecations {
		if name == "Meta" {
			t.Error("FAIL modern metadata mislabeled as deprecated")
		}
	}
	for name, want := range map[string]string{
		"Meta": "owned", "CreateMessageResult": "owned-codec", "ListRootsResult": "owned-codec",
	} {
		if got := result.Dispositions[name]; got != want {
			t.Errorf("FAIL ownership policy %s: got %q want %q", name, got, want)
		}
	}
	if result.SDKSum == "" || len(result.SourceSHA256) != 64 {
		t.Error("FAIL pinned inventory lacks verified SDK context identity")
	}
}

func TestRepairA09TargetExports(t *testing.T) {
	// Exercise the production export-data loader on independently handwritten
	// target files. The fixture is not a replacement for the pinned SDK; the
	// production module/version/checksum validation is still mandatory.
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module " + sdkModule + "\n\ngo 1.27.0\n",
		"go.sum": "",
		"mcp/base.go": `package mcp
type Server struct{}
type Clean struct { Name string }
`,
		"mcp/platform_darwin.go": `package mcp
type PlatformLeak struct{}
type ChangingKind struct{}
type DarwinOnly struct{}
type DeprecatedOnWindows string
`,
		"mcp/platform_linux.go": `package mcp
type PlatformLeak struct{}
type ChangingKind struct{}
type DeprecatedOnWindows string
`,
		"mcp/platform_windows.go": `package mcp
type PlatformLeak struct{}
func (*PlatformLeak) Expose() *Server { return nil }
var ChangingKind int
// Deprecated: Windows-only legacy binding.
type DeprecatedOnWindows string
`,
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	result := auditResult{reasons: make(map[string]string), available: make(map[string]bool), deprecated: make(map[string]bool)}
	for _, target := range []string{"darwin/arm64", "linux/amd64", "windows/amd64"} {
		pkg, comments, err := s.loadSDKTarget(target)
		if err != nil {
			t.Errorf("FAIL target export loader %s: %v", target, err)
			continue
		}
		includeTarget(&result, target, pkg, comments)
	}
	if !result.available["Clean"] || result.reasons["Clean"] != "" {
		t.Errorf("FAIL common safe type rejected: %s", result.reasons["Clean"])
	}
	for _, name := range []string{"DarwinOnly", "ChangingKind"} {
		if result.available[name] {
			t.Errorf("FAIL target-dependent non-type accepted for alias: %s", name)
		}
	}
	for _, name := range []string{"PlatformLeak", "DeprecatedOnWindows"} {
		if !strings.HasPrefix(result.reasons[name], "windows/amd64:") {
			t.Errorf("FAIL target-dependent forbidden alias accepted: %s: %s", name, result.reasons[name])
		}
	}
	if result.deprecated["DeprecatedOnWindows"] {
		t.Error("FAIL target-specific marker reported as unanimous source fact")
	}
	for name, want := range files {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(body) != want {
			t.Errorf("FAIL target export loader changed fixture input %s: %v", name, err)
		}
	}
}
