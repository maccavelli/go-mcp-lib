package qualify

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

// The hashless output inventory at R1 commit 28a9284 is migration evidence only.
// These independent source pins, not its filenames, establish historical bytes.
const legacyManifestSHA256 = "6d658b8c2f55c9caced6feb1d974ff6c7145b10b1da7d8a95e184107fd0b1c2b"

var legacyOutputSHA256 = map[string]string{
	"aliases.go":   "51e46c107b583dcec119dcfd0435e7ef1fcf5a8c24f8d35480946d159e3a1f33",
	"constants.go": "d9681ac8ee6e586dd3e32c16fdb8bb53cf73c065e9e5bbaa6f0d467f1cbf3a62",
	"owned.go":     "0ec535d11557f5dd50ff6a4ef9772b6c54f05e06533ff2b86acc007c0948ab3e",
}

type outputManifest struct {
	Version  int               `json:"version"`
	Revision string            `json:"revision"`
	Files    []string          `json:"files"`
	SHA256   map[string]string `json:"sha256"`
}

// Candidate is an owned scratch snapshot. Closing it never removes source data.
type Candidate struct {
	directory   string
	source      string
	fingerprint map[string]string
	// Previous records verified old output bytes; Proposed records final bytes.
	Previous map[string]string
	Proposed map[string]string
}

// Close releases only the temporary directory created by Assemble.
func (c *Candidate) Close() error { return os.RemoveAll(c.directory) }

// Directory returns the snapshot's module root, never the live repository.
func (c *Candidate) Directory() string { return c.directory }

func contentSHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// Assemble copies the complete source context and replaces verified output only
// inside scratch. It accepts the independently pinned historical generation or
// an exact current generation, and refuses to infer ownership from a header.
func Assemble(source string, proposed map[string][]byte) (_ *Candidate, failure error) {
	identities, err := verifyProposed(proposed)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(source)
	if err != nil {
		return nil, fmt.Errorf("qualify: source root cannot be opened")
	}
	defer func() { _ = root.Close() }()
	previous, err := verifiedPrevious(root, proposed)
	if err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "mcpgen-candidate-")
	if err != nil {
		return nil, err
	}
	candidate := &Candidate{directory: directory, source: source, Previous: previous, Proposed: identities}
	defer func() {
		if failure != nil {
			_ = candidate.Close()
		}
	}()
	destination, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer func() { _ = destination.Close() }()
	candidate.fingerprint, err = snapshot(root, func(name string, entry fs.DirEntry, identity string) error {
		if strings.HasPrefix(name, "protocol/") {
			base := strings.TrimPrefix(name, "protocol/")
			if expected, owned := previous[base]; owned {
				if identity != expected {
					return fmt.Errorf("qualify: historical output changed during snapshot: %s", base)
				}
				return nil
			}
			for proposedName := range proposed {
				if strings.EqualFold(base, proposedName) {
					return fmt.Errorf("qualify: unowned output collision: %s", base)
				}
			}
		}
		if name == "protocol" && !entry.IsDir() {
			return fmt.Errorf("qualify: protocol must be a real directory")
		}
		if entry.IsDir() {
			return destination.MkdirAll(filepath.FromSlash(name), 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := root.Readlink(filepath.FromSlash(name))
			if err != nil {
				return err
			}
			return destination.Symlink(target, filepath.FromSlash(name))
		}
		input, err := root.Open(filepath.FromSlash(name))
		if err != nil {
			return err
		}
		defer func() { _ = input.Close() }()
		info, err := input.Stat()
		if err != nil {
			return err
		}
		output, err := destination.OpenFile(filepath.FromSlash(name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(io.MultiWriter(output, hash), input)
		closeErr := output.Close()
		if err := errors.Join(copyErr, closeErr); err != nil {
			return err
		}
		if hex.EncodeToString(hash.Sum(nil)) != identity {
			return fmt.Errorf("qualify: source changed while copying: %s", name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := destination.MkdirAll("protocol", 0o755); err != nil {
		return nil, err
	}
	for _, name := range sortedNames(proposed) {
		if err := destination.WriteFile(filepath.Join("protocol", name), proposed[name], 0o644); err != nil {
			return nil, err
		}
	}
	if err := candidate.VerifySource(); err != nil {
		return nil, err
	}
	return candidate, nil
}

func sortedNames[V any](values map[string]V) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func verifyProposed(files map[string][]byte) (map[string]string, error) {
	body, found := files["generated-manifest.json"]
	if !found {
		return nil, fmt.Errorf("qualify: proposed manifest missing")
	}
	if _, err := load.New().LoadBytes("generated-manifest.json", body); err != nil {
		return nil, fmt.Errorf("qualify: invalid proposed manifest")
	}
	var manifest outputManifest
	if err := json.Unmarshal(body, &manifest); err != nil || manifest.Version != 2 || manifest.Revision == "" || len(manifest.Files) != len(files)-1 || len(manifest.SHA256) != len(files)-1 {
		return nil, fmt.Errorf("qualify: proposed manifest inventory differs")
	}
	seen := map[string]bool{}
	identities := map[string]string{"generated-manifest.json": contentSHA256(body)}
	for _, name := range manifest.Files {
		if name == "." || path.Base(name) != name || strings.ContainsAny(name, "\\:") || name == "generated-manifest.json" || seen[strings.ToLower(name)] {
			return nil, fmt.Errorf("qualify: invalid or colliding proposed output name")
		}
		body, found := files[name]
		if !found || manifest.SHA256[name] != contentSHA256(body) {
			return nil, fmt.Errorf("qualify: proposed output identity differs: %s", name)
		}
		seen[strings.ToLower(name)] = true
		identities[name] = contentSHA256(body)
	}
	return identities, nil
}

func verifiedPrevious(root *os.Root, proposed map[string][]byte) (map[string]string, error) {
	body, err := readOwnership(root)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("qualify: prior ownership record cannot be read")
	}
	expected := map[string]string{}
	switch {
	case contentSHA256(body) == legacyManifestSHA256:
		for name, hash := range legacyOutputSHA256 {
			expected[name] = hash
		}
	case bytes.Equal(body, proposed["generated-manifest.json"]):
		for name, body := range proposed {
			expected[name] = contentSHA256(body)
		}
	default:
		return nil, fmt.Errorf("qualify: prior output ownership is not independently verified")
	}
	expected["generated-manifest.json"] = contentSHA256(body)
	for _, name := range sortedNames(expected) {
		info, err := root.Lstat(filepath.Join("protocol", name))
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("qualify: prior owned output is missing or not regular: %s", name)
		}
		file, err := root.Open(filepath.Join("protocol", name))
		if err != nil {
			return nil, fmt.Errorf("qualify: prior output cannot be opened: %s", name)
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if errors.Join(copyErr, closeErr) != nil || hex.EncodeToString(hash.Sum(nil)) != expected[name] {
			return nil, fmt.Errorf("qualify: prior output identity differs: %s", name)
		}
	}
	return expected, nil
}

func readOwnership(root *os.Root) ([]byte, error) {
	file, err := root.Open("protocol/generated-manifest.json")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	const limit = 2 << 20
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if len(body) > limit {
		return nil, fmt.Errorf("qualify: ownership record exceeds 2 MiB")
	}
	return body, nil
}

// snapshot streams regular-file identities and records contained relative links.
// Git internals and execution caches are not compiler inputs in this repository.
func snapshot(root *os.Root, visit func(string, fs.DirEntry, string) error) (map[string]string, error) {
	identities := map[string]string{}
	err := fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("qualify: source entry cannot be read: %s", name)
		}
		if name == "." {
			return nil
		}
		if name == ".git" || name == ".cache" {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		var identity string
		switch {
		case entry.IsDir():
			identity = "directory"
		case entry.Type()&os.ModeSymlink != 0:
			if _, err := root.Stat(filepath.FromSlash(name)); err != nil {
				return fmt.Errorf("qualify: escaping or dangling source link: %s", name)
			}
			target, err := root.Readlink(filepath.FromSlash(name))
			if err != nil {
				return err
			}
			identity = "link:" + target
		default:
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("qualify: source entry is not a regular file: %s", name)
			}
			file, err := root.Open(filepath.FromSlash(name))
			if err != nil {
				return err
			}
			hash := sha256.New()
			_, copyErr := io.Copy(hash, file)
			closeErr := file.Close()
			if err := errors.Join(copyErr, closeErr); err != nil {
				return err
			}
			identity = hex.EncodeToString(hash.Sum(nil))
		}
		identities[name] = identity
		if visit != nil {
			return visit(name, entry, identity)
		}
		return nil
	})
	return identities, err
}

// VerifySource detects changes to bytes, links and source inventory during a run.
func (c *Candidate) VerifySource() error {
	root, err := os.OpenRoot(c.source)
	if err != nil {
		return fmt.Errorf("qualify: source root disappeared")
	}
	defer func() { _ = root.Close() }()
	current, err := snapshot(root, nil)
	if err != nil {
		return err
	}
	if len(current) != len(c.fingerprint) {
		return fmt.Errorf("qualify: source inventory changed during qualification")
	}
	for _, name := range sortedNames(c.fingerprint) {
		if current[name] != c.fingerprint[name] {
			return fmt.Errorf("qualify: source changed during qualification: %s", name)
		}
	}
	return nil
}

// Compile includes both module boundaries. All foreign test binaries
// compile only; native test execution remains a separately attributed gate.
func (c *Candidate) Compile(runner Runner, cache string) ([]*Report, error) {
	reports := []*Report{}
	for _, relative := range []string{".", "cmd/mcpgen"} {
		directory := filepath.Join(c.directory, filepath.FromSlash(relative))
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err != nil {
			if relative == "." {
				return nil, fmt.Errorf("qualify: runtime module is missing")
			}
			continue
		}
		report, err := CompileTree(runner, cache, directory)
		if err != nil {
			return nil, err
		}
		report.Candidate = relative
		reports = append(reports, report)
	}
	if err := c.VerifySource(); err != nil {
		return nil, err
	}
	return reports, nil
}

// AllPassed accepts only complete target matrices from both product modules.
func AllPassed(reports []*Report) bool {
	modules := []string{}
	for _, report := range reports {
		if report == nil || !report.Passed || len(report.Targets) != len(Targets) {
			return false
		}
		for i, target := range report.Targets {
			if target.Target != Targets[i] || !target.Package || !target.Tests {
				return false
			}
		}
		modules = append(modules, report.Candidate)
	}
	return slices.Equal(modules, []string{".", "cmd/mcpgen"})
}
