// Package acquire downloads the pinned MCP schema and verifies its bytes
// against the frozen hash before anything is written. A mismatched fetch
// is discarded; the network is only touched when the target file is absent.
package acquire

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Options configures an acquire run.
type Options struct {
	// Inputs is the schema inputs directory to populate, for example
	// schema/inputs/2026-07-28/.
	InputsDir string
	// Revision is the protocol revision being acquired, for example
	// 2026-07-28.
	Revision string
	// Commit is the pinned upstream commit of the schema source.
	Commit string
	// SHA256 is the frozen expected hash of the schema bytes.
	SHA256 string
	// URL overrides the derived upstream source; used by tests.
	URL string
	// Client overrides the HTTP client; nil uses a 30-second default.
	Client *http.Client
}

// Result reports what acquire verified and wrote.
type Result struct {
	Bytes    int64  `json:"bytes"`
	SHA256   string `json:"sha256"`
	Existing bool   `json:"existing"`
}

const rawURLFormat = "https://raw.githubusercontent.com/modelcontextprotocol/modelcontextprotocol/%s/schema/%s/schema.json"

// Run acquires the pinned schema into InputsDir. When the target file
// already exists, its hash is verified and the network is never touched.
// A fresh fetch is verified against SHA256 before it replaces the target;
// a mismatched fetch is discarded and reported, never written.
func Run(opts Options) (*Result, error) {
	if opts.InputsDir == "" || opts.Revision == "" || opts.Commit == "" || opts.SHA256 == "" {
		return nil, fmt.Errorf("acquire: inputs dir, revision, commit, and sha256 are required")
	}
	target := filepath.Join(opts.InputsDir, "schema.json")
	if data, err := os.ReadFile(target); err == nil {
		got := hash(data)
		if got != opts.SHA256 {
			return nil, fmt.Errorf("acquire: existing %s hash %s does not match the pin %s", target, got, opts.SHA256)
		}
		return &Result{Bytes: int64(len(data)), SHA256: got, Existing: true}, nil
	}
	url := opts.URL
	if url == "" {
		url = fmt.Sprintf(rawURLFormat, opts.Commit, opts.Revision)
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("acquire: fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("acquire: fetch %s: status %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("acquire: fetch %s: %w", url, err)
	}
	if got := hash(data); got != opts.SHA256 {
		return nil, fmt.Errorf("acquire: fetched bytes hash %s do not match the pin %s; nothing was written", got, opts.SHA256)
	}
	if err := os.MkdirAll(opts.InputsDir, 0o755); err != nil {
		return nil, fmt.Errorf("acquire: %w", err)
	}
	temp, err := os.CreateTemp(opts.InputsDir, ".acquire-*")
	if err != nil {
		return nil, fmt.Errorf("acquire: %w", err)
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return nil, fmt.Errorf("acquire: %w", err)
	}
	if err := temp.Close(); err != nil {
		return nil, fmt.Errorf("acquire: %w", err)
	}
	if err := os.Rename(tempName, target); err != nil {
		return nil, fmt.Errorf("acquire: %w", err)
	}
	return &Result{Bytes: int64(len(data)), SHA256: opts.SHA256, Existing: false}, nil
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
