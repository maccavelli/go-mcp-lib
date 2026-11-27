package sdkmap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const sdkModule = "github.com/modelcontextprotocol/go-sdk"
const sdkPackage = sdkModule + "/mcp"

type moduleMetadata struct {
	Path, Version, Dir, Sum string
	Replace                 *moduleMetadata
}

type packageMetadata struct {
	ImportPath, Dir, Export string
	GoFiles                 []string
	Module                  *moduleMetadata
	Error                   *struct{ Err string }
	DepsErrors              []struct{ Err string }
}

type metadataSnapshot struct {
	root, directory string
	mod, sum        []byte
}

func snapshot(root string) (*metadataSnapshot, error) {
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "mcp-sdk-metadata-*")
	if err != nil {
		return nil, err
	}
	s := &metadataSnapshot{root, directory, mod, sum}
	for name, body := range map[string][]byte{"go.mod": mod, "go.sum": sum} {
		if err := os.WriteFile(filepath.Join(directory, name), body, 0o600); err != nil {
			s.close()
			return nil, err
		}
	}
	return s, nil
}

func (s *metadataSnapshot) close() { _ = os.RemoveAll(s.directory) }

func metadataEnvironment(target string) []string {
	overrides := map[string]string{"CGO_ENABLED": "0", "GOWORK": "off", "GOTOOLCHAIN": "local", "GOFLAGS": "-mod=readonly", "GOENV": "off", "GOEXPERIMENT": "", "GOPROXY": "off", "GOSUMDB": "off"}
	if target != "" {
		parts := strings.Split(target, "/")
		overrides["GOOS"], overrides["GOARCH"] = parts[0], parts[1]
	}
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, overridden := overrides[key]; !overridden && key != "GOOS" && key != "GOARCH" && key != "GOTOOLDIR" {
			env = append(env, entry)
		}
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

type metadataBuffer struct{ bytes.Buffer }

func (b *metadataBuffer) Write(data []byte) (int, error) {
	if b.Len()+len(data) > 32<<20 {
		return 0, fmt.Errorf("SDK metadata output exceeds its buffer bound")
	}
	return b.Buffer.Write(data)
}

func (s *metadataSnapshot) command(target string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", args...)
	command.Dir, command.Env = s.root, metadataEnvironment(target)
	if len(args) >= 2 && args[0] == "mod" && args[1] == "verify" {
		command.Dir = s.directory
	}
	var output, diagnostic metadataBuffer
	command.Stdout, command.Stderr = &output, &diagnostic
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("sdkmap: offline metadata command failed: %w: %.256s", err, diagnostic.String())
	}
	for name, expected := range map[string][]byte{"go.mod": s.mod, "go.sum": s.sum} {
		actual, err := os.ReadFile(filepath.Join(s.directory, name))
		if err != nil || !bytes.Equal(actual, expected) {
			return nil, fmt.Errorf("sdkmap: metadata command changed isolated module input")
		}
	}
	return output.Bytes(), nil
}

func (s *metadataSnapshot) module() (*moduleMetadata, error) {
	data, err := s.command("", "list", "-m", "-json", "-modfile="+filepath.Join(s.directory, "go.mod"), sdkModule)
	if err != nil {
		return nil, err
	}
	var module moduleMetadata
	if err := json.Unmarshal(data, &module); err != nil {
		return nil, err
	}
	if module.Path != sdkModule || module.Dir == "" || module.Version != "v1.8.0" || module.Replace != nil || module.Sum == "" {
		return nil, fmt.Errorf("sdkmap: missing, replaced or mismatched SDK pin")
	}
	return &module, nil
}

// RuntimeModule finds the explicitly prepared runtime repository containing start.
func RuntimeModule(start string) (string, error) {
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "tools", "versions.json")); err == nil {
			if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
				return directory, nil
			}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("sdkmap: prepared runtime module context not found")
		}
		directory = parent
	}
}

// LocateSDK uses pinned module metadata, independent of a home-directory layout.
func LocateSDK(root string) (string, error) {
	s, err := snapshot(root)
	if err != nil {
		return "", err
	}
	defer s.close()
	module, err := s.module()
	if err != nil {
		return "", err
	}
	return filepath.Join(module.Dir, "mcp"), nil
}

type auditResult struct {
	reasons    map[string]string
	digest     string
	sum        string
	available  map[string]bool
	deprecated map[string]bool
	api        map[string]map[string]TypeAPI
}

var audits = struct {
	sync.Mutex
	values map[string]auditResult
}{values: make(map[string]auditResult)}

func auditSDK(root, sdkDir string) (auditResult, error) {
	s, err := snapshot(root)
	if err != nil {
		return auditResult{}, err
	}
	defer s.close()
	module, err := s.module()
	if err != nil {
		return auditResult{}, err
	}
	want, err := filepath.EvalSymlinks(filepath.Join(module.Dir, "mcp"))
	if err != nil {
		return auditResult{}, err
	}
	actual, err := filepath.EvalSymlinks(sdkDir)
	if err != nil || actual != want {
		return auditResult{}, fmt.Errorf("sdkmap: SDK source directory does not match verified module metadata")
	}
	hash := sha256.New()
	hash.Write(s.mod)
	hash.Write(s.sum)
	entries, err := os.ReadDir(want)
	if err != nil {
		return auditResult{}, err
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".go") {
			body, err := os.ReadFile(filepath.Join(want, entry.Name()))
			if err != nil {
				return auditResult{}, err
			}
			hash.Write([]byte(entry.Name() + "\x00"))
			hash.Write(body)
		}
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	compiler, err := s.command("", "env", "GOVERSION")
	if err != nil {
		return auditResult{}, err
	}
	key := root + "\x00" + want + "\x00" + digest + "\x00" + string(compiler)
	audits.Lock()
	defer audits.Unlock()
	if result, found := audits.values[key]; found {
		return result, nil
	}
	if _, err := s.command("", "mod", "verify"); err != nil {
		return auditResult{}, err
	}
	result := auditResult{reasons: make(map[string]string), digest: digest, sum: module.Sum,
		available: make(map[string]bool), deprecated: make(map[string]bool)}
	for _, target := range []string{"darwin/arm64", "linux/amd64", "windows/amd64"} {
		pkg, deprecated, err := s.loadSDKTarget(target)
		if err != nil {
			return auditResult{}, err
		}
		includeTarget(&result, target, pkg, deprecated)
	}
	// At most one prepared SDK context is retained; changed inputs invalidate it.
	clear(audits.values)
	audits.values[key] = result
	return result, nil
}

func (s *metadataSnapshot) loadSDKTarget(target string) (*types.Package, map[string]bool, error) {
	data, err := s.command(target, "list", "-deps", "-export", "-json", "-modfile="+filepath.Join(s.directory, "go.mod"), sdkPackage)
	if err != nil {
		return nil, nil, err
	}
	packages := make(map[string]packageMetadata)
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg packageMetadata
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, err
		}
		if pkg.Error != nil || len(pkg.DepsErrors) != 0 || pkg.Module != nil && pkg.Module.Replace != nil {
			return nil, nil, fmt.Errorf("sdkmap: incomplete or replaced package metadata")
		}
		packages[pkg.ImportPath] = pkg
	}
	fset := token.NewFileSet()
	loader := importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		pkg, found := packages[path]
		if !found || pkg.Export == "" {
			return nil, fmt.Errorf("sdkmap: explicit export data missing for %s", path)
		}
		return os.Open(pkg.Export)
	})
	pkg, err := loader.Import(sdkPackage)
	if err != nil {
		return nil, nil, fmt.Errorf("sdkmap: compiler export data import: %w", err)
	}
	deprecated, err := deprecationComments(packages[sdkPackage])
	if err != nil {
		return nil, nil, err
	}
	return pkg, deprecated, nil
}

func includeTarget(result *auditResult, target string, pkg *types.Package, deprecated map[string]bool) {
	if result.api == nil {
		result.api = make(map[string]map[string]TypeAPI)
	}
	if target == "darwin/arm64" {
		for name, marked := range deprecated {
			if marked {
				result.deprecated[name] = true
			}
		}
	} else {
		for name := range result.deprecated {
			if !deprecated[name] {
				delete(result.deprecated, name)
			}
		}
	}
	if target != "darwin/arm64" {
		for name := range result.available {
			if object, ok := pkg.Scope().Lookup(name).(*types.TypeName); !ok || !object.Exported() {
				delete(result.available, name)
			}
		}
	}
	for _, name := range pkg.Scope().Names() {
		object, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok || !object.Exported() {
			continue
		}
		if target == "darwin/arm64" {
			result.available[name] = true
		}
		if result.api[name] == nil {
			result.api[name] = make(map[string]TypeAPI)
		}
		result.api[name][target] = typeAPI(object.Type())
		if reason := aliasViolation(object.Type(), deprecated); reason != "" && result.reasons[name] == "" {
			result.reasons[name] = target + ": " + reason
		}
	}
}

func deprecationComments(pkg packageMetadata) (map[string]bool, error) {
	deprecated := make(map[string]bool)
	for _, name := range pkg.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(pkg.Dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range declaration.Specs {
					if spec, ok := specification.(*ast.TypeSpec); ok {
						deprecated[spec.Name.Name] = deprecatedComment(declaration.Doc, spec.Doc)
						if structure, ok := spec.Type.(*ast.StructType); ok {
							for _, field := range structure.Fields.List {
								if len(field.Names) == 0 {
									deprecated[spec.Name.Name+"."+declarationName(field.Type)] = deprecatedComment(field.Doc, field.Comment)
								}
								for _, name := range field.Names {
									deprecated[spec.Name.Name+"."+name.Name] = deprecatedComment(field.Doc, field.Comment)
								}
							}
						}
					}
				}
			case *ast.FuncDecl:
				if declaration.Recv != nil && len(declaration.Recv.List) == 1 {
					if name := declarationName(declaration.Recv.List[0].Type); name != "" {
						deprecated[name+"."+declaration.Name.Name] = deprecatedComment(declaration.Doc)
					}
				}
			}
		}
	}
	return deprecated, nil
}

func deprecatedComment(groups ...*ast.CommentGroup) bool {
	for _, group := range groups {
		for paragraph := range strings.SplitSeq(group.Text(), "\n\n") {
			if strings.HasPrefix(paragraph, "Deprecated:") {
				return true
			}
		}
	}
	return false
}

func declarationName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.StarExpr:
		return declarationName(expression.X)
	case *ast.IndexExpr:
		return declarationName(expression.X)
	case *ast.IndexListExpr:
		return declarationName(expression.X)
	case *ast.SelectorExpr:
		return expression.Sel.Name
	}
	return ""
}

func aliasViolation(root types.Type, deprecated map[string]bool) string {
	seen := make(map[types.Type]bool)
	var walk func(types.Type) string
	walk = func(value types.Type) string {
		if value == nil || seen[value] {
			return ""
		}
		seen[value] = true
		switch value := value.(type) {
		case *types.Alias:
			if value.TypeParams() != nil && value.TypeParams().Len() != value.TypeArgs().Len() {
				return "uninstantiated generic alias " + value.Obj().Name()
			}
			if object := value.Obj(); object.Pkg() != nil && strings.HasPrefix(object.Pkg().Path(), sdkModule) {
				if object.Pkg().Path() != sdkPackage || !object.Exported() {
					return "unreviewed SDK alias " + object.Name()
				}
				if deprecated[object.Name()] {
					return "deprecated " + object.Name()
				}
			}
			return walk(types.Unalias(value))
		case *types.Named:
			object := value.Obj()
			isSDK := object.Pkg() != nil && object.Pkg().Path() == sdkPackage
			if object.Pkg() != nil && strings.HasPrefix(object.Pkg().Path(), sdkModule) {
				if object.Pkg().Path() != sdkPackage || !object.Exported() || deprecated[object.Name()] {
					return "unreviewed or deprecated SDK type " + object.Name()
				}
				switch object.Name() {
				case "Client", "Server", "ClientSession", "ServerSession", "Connection", "Transport", "MethodHandler", "Request", "Session":
					return "SDK runtime type " + object.Name()
				}
			}
			if value.TypeParams() != nil && value.TypeParams().Len() != value.TypeArgs().Len() {
				return "uninstantiated generic " + object.Name()
			}
			for i := 0; i < value.TypeArgs().Len(); i++ {
				if reason := walk(value.TypeArgs().At(i)); reason != "" {
					return object.Name() + " type argument: " + reason
				}
			}
			for _, receiver := range []types.Type{value, types.NewPointer(value)} {
				methods := types.NewMethodSet(receiver)
				for i := 0; i < methods.Len(); i++ {
					method := methods.At(i).Obj()
					if isSDK && deprecated[object.Name()+"."+method.Name()] {
						return "deprecated method " + object.Name() + "." + method.Name()
					}
					if reason := walk(method.Type()); reason != "" {
						return object.Name() + "." + method.Name() + ": " + reason
					}
				}
			}
			if structure, ok := value.Underlying().(*types.Struct); ok {
				for i := 0; i < structure.NumFields(); i++ {
					field := structure.Field(i)
					if isSDK && deprecated[object.Name()+"."+field.Name()] {
						return "deprecated field " + object.Name() + "." + field.Name()
					}
				}
			}
			return walk(value.Underlying())
		case *types.Pointer:
			return walk(value.Elem())
		case *types.Array:
			return walk(value.Elem())
		case *types.Slice:
			return walk(value.Elem())
		case *types.Map:
			if reason := walk(value.Key()); reason != "" {
				return reason
			}
			return walk(value.Elem())
		case *types.Chan:
			return walk(value.Elem())
		case *types.Struct:
			for i := 0; i < value.NumFields(); i++ {
				if reason := walk(value.Field(i).Type()); reason != "" {
					return value.Field(i).Name() + ": " + reason
				}
			}
		case *types.Signature:
			for _, parameters := range []*types.TypeParamList{value.TypeParams(), value.RecvTypeParams()} {
				for i := 0; i < parameters.Len(); i++ {
					if reason := walk(parameters.At(i)); reason != "" {
						return reason
					}
				}
			}
			if value.Recv() != nil {
				if reason := walk(value.Recv().Type()); reason != "" {
					return reason
				}
			}
			if reason := walk(value.Params()); reason != "" {
				return reason
			}
			return walk(value.Results())
		case *types.Tuple:
			for i := 0; i < value.Len(); i++ {
				if reason := walk(value.At(i).Type()); reason != "" {
					return reason
				}
			}
		case *types.Interface:
			value.Complete()
			for i := 0; i < value.NumMethods(); i++ {
				method := value.Method(i)
				if !method.Exported() && method.Pkg() != nil && strings.HasPrefix(method.Pkg().Path(), sdkModule) {
					return "sealed SDK interface"
				}
				if reason := walk(method.Type()); reason != "" {
					return reason
				}
			}
			for i := 0; i < value.NumEmbeddeds(); i++ {
				if reason := walk(value.EmbeddedType(i)); reason != "" {
					return reason
				}
			}
		case *types.TypeParam:
			return walk(value.Constraint())
		case *types.Union:
			for i := 0; i < value.Len(); i++ {
				if reason := walk(value.Term(i).Type()); reason != "" {
					return reason
				}
			}
		}
		return ""
	}
	return walk(root)
}

func sortedReasons(reasons map[string]string) []string {
	var names []string
	for name := range reasons {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
