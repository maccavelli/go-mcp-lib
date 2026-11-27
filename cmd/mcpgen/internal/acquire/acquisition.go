// Package acquire verifies immutable, bounded HTTPS inputs before writing a
// candidate. Ordinary generation does not import or invoke its network path.
package acquire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
)

const maxBytes = 8 << 20
const rawURLFormat = "https://raw.githubusercontent.com/modelcontextprotocol/modelcontextprotocol/%s/schema/%s/schema.json"

// Options binds acquisition to an immutable source and an isolated destination.
type Options struct {
	InputsDir string
	Revision  string
	Commit    string
	SHA256    string
	URL       string
	Client    *http.Client
	Context   context.Context
}

// Result records the verified content identity and whether it already existed.
type Result struct {
	Bytes    int64  `json:"bytes"`
	SHA256   string `json:"sha256"`
	Existing bool   `json:"existing"`
}

func hash(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }

func source(opts Options) (string, error) {
	commit, err := hex.DecodeString(opts.Commit)
	if err != nil || len(commit) != 20 || opts.Commit != strings.ToLower(opts.Commit) || !ownership.Hash(opts.SHA256) || opts.Revision == "" {
		return "", fmt.Errorf("acquire: immutable 40-hex commit, revision and 64-hex sha256 required")
	}
	source := opts.URL
	if source == "" {
		source = fmt.Sprintf(rawURLFormat, opts.Commit, opts.Revision)
	}
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || strings.Contains(source, "%") || path.Clean(u.Path) != u.Path {
		return "", fmt.Errorf("acquire: clean immutable HTTPS URL required")
	}
	found := 0
	for _, part := range strings.Split(u.Path, "/") {
		if part == opts.Commit {
			found++
		}
	}
	if found != 1 {
		return "", fmt.Errorf("acquire: source URL must contain the exact commit as one path component")
	}
	return source, nil
}

func fetch(opts Options) ([]byte, error) {
	source, err := source(opts)
	if err != nil {
		return nil, err
	}
	parent := opts.Context
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, err
	}
	client := http.Client{Timeout: 30 * time.Second}
	if opts.Client != nil {
		client = *opts.Client
		if client.Timeout <= 0 || client.Timeout > 30*time.Second {
			client.Timeout = 30 * time.Second
		}
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return fmt.Errorf("acquire: redirects are not accepted") }
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("acquire: request failed: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Join(fmt.Errorf("acquire: status %d", response.StatusCode), response.Body.Close())
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err := errors.Join(readErr, response.Body.Close()); err != nil {
		return nil, err
	}
	if len(body) > maxBytes {
		return nil, fmt.Errorf("acquire: response exceeds 8 MiB")
	}
	if hash(body) != opts.SHA256 {
		return nil, fmt.Errorf("acquire: hash mismatch; nothing written")
	}
	return body, nil
}

func write(root *os.Root, name string, body []byte) (failure error) {
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() { failure = errors.Join(failure, file.Close()) }()
	n, err := file.Write(body)
	if err != nil {
		return err
	}
	if n != len(body) {
		return io.ErrShortWrite
	}
	return file.Sync()
}

// Run is the independently pinned prepared-input primitive. Existing verified
// bytes require no network; a fresh output uses rooted exclusive creation.
func Run(opts Options) (_ *Result, failure error) {
	if opts.InputsDir == "" {
		return nil, fmt.Errorf("acquire: inputs directory required")
	}
	if _, err := source(opts); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(opts.InputsDir)
	if err != nil {
		return nil, err
	}
	defer func() { failure = errors.Join(failure, root.Close()) }()
	body, err := ownership.Read(root, "schema.json")
	if err == nil {
		if hash(body) != opts.SHA256 {
			return nil, fmt.Errorf("acquire: existing hash mismatch")
		}
		return &Result{Bytes: int64(len(body)), SHA256: opts.SHA256, Existing: true}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	body, err = fetch(opts)
	if err != nil {
		return nil, err
	}
	if err := write(root, "schema.json", body); err != nil {
		return nil, err
	}
	return &Result{Bytes: int64(len(body)), SHA256: opts.SHA256}, nil
}

// Candidate fetches into a new disjoint directory after all argument and pin
// checks. A failure never changes repository inputs or existing candidate data.
func Candidate(repo, directory string, opts Options) (_ *Result, failure error) {
	if directory == "" || filepath.Clean(directory) != directory {
		return nil, fmt.Errorf("acquire: clean new candidate directory required")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	base, err := filepath.EvalSymlinks(repo)
	if err != nil {
		return nil, err
	}
	base, err = filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return nil, err
	}
	if parent != filepath.Dir(absolute) {
		temporary, err := filepath.EvalSymlinks(os.TempDir())
		if err != nil {
			return nil, err
		}
		relative, err := filepath.Rel(filepath.Clean(os.TempDir()), filepath.Dir(absolute))
		if err != nil || !filepath.IsLocal(relative) || parent != filepath.Join(temporary, relative) {
			return nil, fmt.Errorf("acquire: candidate ancestor is aliased")
		}
	}
	// Canonicalize OS temporary-directory aliases while refusing a selected
	// candidate that itself aliases an existing path. The opened parent bounds
	// every creation; neither repo nor candidate may contain the other.
	absolute = filepath.Join(parent, filepath.Base(absolute))
	inside := func(a, b string) bool {
		relative, err := filepath.Rel(a, b)
		return err == nil && filepath.IsLocal(relative)
	}
	if inside(base, absolute) || inside(absolute, base) {
		return nil, fmt.Errorf("acquire: candidate overlaps live repository")
	}
	if err := ownership.Name(filepath.Base(absolute)); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, err
	}
	defer func() { failure = errors.Join(failure, root.Close()) }()
	name := filepath.Base(absolute)
	if _, err := root.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("acquire: candidate already exists or is unreadable")
	}
	body, err := fetch(opts)
	if err != nil {
		return nil, err
	}
	if err := root.Mkdir(name, 0o755); err != nil {
		return nil, err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	child, err := root.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	defer func() { failure = errors.Join(failure, child.Close()) }()
	actual, err := child.Stat(".")
	if err != nil || !os.SameFile(info, actual) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("acquire: candidate identity changed")
	}
	if err := write(child, "schema.json", body); err != nil {
		return nil, err
	}
	source, err := source(opts)
	if err != nil {
		return nil, err
	}
	provenance, err := json.MarshalIndent(map[string]any{"version": 1, "commit": opts.Commit, "revision": opts.Revision, "source": source, "sha256": opts.SHA256, "bytes": len(body), "promoted": false}, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := write(child, "manifest.json", append(provenance, '\n')); err != nil {
		return nil, err
	}
	current, err := root.Lstat(name)
	if err != nil || !os.SameFile(info, current) {
		return nil, fmt.Errorf("acquire: candidate directory moved during acquisition")
	}
	return &Result{Bytes: int64(len(body)), SHA256: opts.SHA256}, nil
}
