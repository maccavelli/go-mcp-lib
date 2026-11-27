package acquire

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairA14Acquisition(t *testing.T) {
	for _, name := range []string{"immutable", "url-commit", "http", "redirect", "oversized", "hash", "canceled", "overlap", "existing", "alias"} {
		t.Run(name, func(t *testing.T) {
			want, opts := pin()
			opts.SHA256 = want
			body := "schema-body-for-pinning"
			if name == "oversized" {
				body = strings.Repeat("x", maxBytes+1)
				opts.SHA256 = hash([]byte(body[:maxBytes]))
			}
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if name == "redirect" {
					http.Redirect(w, r, r.URL.Path+"/redirected", http.StatusFound)
					return
				}
				_, _ = fmt.Fprint(w, body)
			}))
			defer server.Close()
			opts.Client = server.Client()
			opts.URL = server.URL + "/" + opts.Commit + "/schema.json"
			parent := t.TempDir()
			repo := filepath.Join(parent, "repo")
			candidate := filepath.Join(parent, "candidate")
			if err := os.Mkdir(repo, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repo, "sentinel"), []byte("live"), 0o644); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "immutable":
				opts.Commit = "main"
				opts.URL = server.URL + "/main/schema.json"
			case "url-commit":
				opts.URL = server.URL + "/main/schema.json"
			case "http":
				opts.URL = strings.Replace(opts.URL, "https:", "http:", 1)
			case "hash":
				opts.SHA256 = strings.Repeat("0", 64)
			case "canceled":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				opts.Context = ctx
			case "overlap":
				candidate = filepath.Join(repo, "schema")
			case "existing":
				if err := os.Mkdir(candidate, 0o755); err != nil {
					t.Fatal(err)
				}
			case "alias":
				if err := os.Symlink(repo, candidate); err != nil {
					t.Fatal(err)
				}
			}
			_, err := Candidate(repo, candidate, opts)
			if err == nil {
				t.Errorf("FAIL acquisition guard accepted %s", name)
			}
			if body, err := os.ReadFile(filepath.Join(repo, "sentinel")); err != nil || string(body) != "live" {
				t.Error("FAIL acquisition changed live sentinel")
			}
			if _, err := os.Lstat(filepath.Join(candidate, "schema.json")); !os.IsNotExist(err) {
				t.Error("FAIL rejected acquisition wrote candidate")
			}
		})
	}
	t.Run("valid", func(t *testing.T) {
		want, opts := pin()
		opts.SHA256 = want
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, "schema-body-for-pinning") }))
		defer server.Close()
		opts.Client = server.Client()
		opts.URL = server.URL + "/" + opts.Commit + "/schema.json"
		parent := t.TempDir()
		repo := filepath.Join(parent, "repo")
		if err := os.Mkdir(repo, 0o755); err != nil {
			t.Fatal(err)
		}
		candidate := filepath.Join(parent, "candidate")
		result, err := Candidate(repo, candidate, opts)
		if err != nil || result == nil || result.SHA256 != want {
			t.Fatalf("FAIL valid acquisition: %v", err)
		}
		body, err := os.ReadFile(filepath.Join(candidate, "manifest.json"))
		if err != nil || !strings.Contains(string(body), opts.Commit) || !strings.Contains(string(body), want) {
			t.Error("FAIL immutable provenance missing")
		}
	})
}
