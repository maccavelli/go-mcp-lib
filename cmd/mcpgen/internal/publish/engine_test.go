package publish

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
)

var errInjected = errors.New("injected filesystem failure")

func generation(t *testing.T, values map[string]string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	manifest := ownership.Manifest{Version: 2, Revision: "fixture", SHA256: map[string]string{}}
	for name, value := range values {
		files[name] = []byte(value)
		manifest.SHA256[name] = ownership.Digest([]byte(value))
	}
	manifest.Files = ownership.Sorted(files)
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	files[ownership.ManifestName] = body
	return files
}

func fixture(t *testing.T) (string, Options) {
	t.Helper()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, "protocol"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := generation(t, map[string]string{"a.go": "old a", "stale.go": "old stale", "z.go": "old z"})
	for name, body := range before {
		if err := os.WriteFile(filepath.Join(repo, "protocol", name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "protocol", "sentinel.txt"), []byte("unrelated"), 0o644); err != nil {
		t.Fatal(err)
	}
	previous, err := ownership.Candidate(before)
	if err != nil {
		t.Fatal(err)
	}
	return repo, Options{Previous: previous, Files: generation(t, map[string]string{"a.go": "new a", "b.go": "new b", "z.go": "new z"})}
}

func publicBytes(t *testing.T, repo string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repo, "protocol"))
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]string{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(repo, "protocol", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		result[entry.Name()] = string(body)
	}
	return result
}

func stop(point, name string) Hook {
	return func(p, n string) error {
		if p == point && (name == "" || n == name) {
			return errInjected
		}
		return nil
	}
}

func TestRepairA13Publication(t *testing.T) {
	repo, options := fixture(t)
	before := publicBytes(t, repo)
	options.Hook = func(point, name string) error {
		if point == "rename" && name == ownership.ManifestName {
			body, err := os.ReadFile(filepath.Join(repo, "protocol", "z.go"))
			if err != nil || string(body) != "new z" {
				t.Error("FAIL manifest promoted before data")
			}
			if _, err := os.Stat(filepath.Join(repo, "protocol", "stale.go")); !os.IsNotExist(err) {
				t.Error("FAIL manifest promoted before deletion")
			}
		}
		return nil
	}
	receipt, err := Publish(context.Background(), repo, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Changed) != 5 || receipt.Changed[len(receipt.Changed)-1] != ownership.ManifestName {
		t.Errorf("FAIL manifest-last receipt: %+v", receipt)
	}
	if publicBytes(t, repo)["sentinel.txt"] != "unrelated" {
		t.Fatal("FAIL unrelated sentinel changed")
	}
	info, err := os.Stat(filepath.Join(repo, "protocol", "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	options.Previous, err = ownership.Candidate(options.Files)
	if err != nil {
		t.Fatal(err)
	}
	options.Hook = nil
	if receipt, err := Publish(context.Background(), repo, options); err != nil || receipt.State != "unchanged" {
		t.Fatalf("FAIL deterministic regeneration: %+v %v", receipt, err)
	}
	after, err := os.Stat(filepath.Join(repo, "protocol", "a.go"))
	if err != nil || !info.ModTime().Equal(after.ModTime()) {
		t.Error("FAIL unchanged mtime rewritten")
	}
	for i := 0; i < 2; i++ {
		if _, err := Recover(context.Background(), repo, "restore", nil); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, publicBytes(t, repo)) {
			t.Error("FAIL original bytes/manifest not restored")
		}
	}
	if _, err := Recover(context.Background(), repo, "complete", nil); err != nil {
		t.Fatal(err)
	}
	options.Files = generation(t, map[string]string{"a.go": "third", "b.go": "new b", "z.go": "new z"})
	if _, err := Publish(context.Background(), repo, options); err != nil {
		t.Fatal(err)
	}
	stages, err := os.ReadDir(filepath.Join(repo, "protocol", stateName))
	if err != nil || len(stages) != 1 {
		t.Errorf("FAIL prior stage cleanup: %d %v", len(stages), err)
	}
}

func TestRepairA13Bootstrap(t *testing.T) {
	t.Run("first-generation", func(t *testing.T) {
		repo := t.TempDir()
		options := Options{Files: generation(t, map[string]string{"a.go": "first"})}
		if _, err := Publish(context.Background(), repo, options); err != nil {
			t.Fatal(err)
		}
		if _, err := Recover(context.Background(), repo, "restore", nil); err != nil {
			t.Fatal(err)
		}
		if len(publicBytes(t, repo)) != 0 {
			t.Error("FAIL first-generation restore retained new outputs")
		}
	})
	t.Run("verified-legacy", func(t *testing.T) {
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, "protocol"), 0o755); err != nil {
			t.Fatal(err)
		}
		previous := ownership.LegacyHashes()
		previous[ownership.ManifestName] = ownership.LegacyManifestSHA256
		for name := range previous {
			fixture := name
			if strings.HasSuffix(name, ".go") {
				fixture += ".txt"
			}
			body, err := os.ReadFile(filepath.Join("../qualify/testdata/legacy", fixture))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repo, "protocol", name), body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		options := Options{Previous: previous, Files: generation(t, map[string]string{"a.go": "modern"})}
		before := publicBytes(t, repo)
		if _, err := Publish(context.Background(), repo, options); err == nil {
			t.Fatal("FAIL implicit hashless bootstrap accepted")
		}
		if !reflect.DeepEqual(before, publicBytes(t, repo)) {
			t.Error("FAIL refused bootstrap changed source")
		}
		options.Bootstrap = func(context.Context) error { return errInjected }
		if _, err := Publish(context.Background(), repo, options); !errors.Is(err, errInjected) {
			t.Fatalf("FAIL bootstrap proof bypass: %v", err)
		}
		options.Bootstrap = func(context.Context) error {
			if _, err := os.Stat(filepath.Join(repo, "protocol", LockName)); err != nil {
				return fmt.Errorf("proof did not run under the lock: %w", err)
			}
			return nil
		}
		if _, err := Publish(context.Background(), repo, options); err != nil {
			t.Fatal(err)
		}
		if _, err := Recover(context.Background(), repo, "restore", nil); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, publicBytes(t, repo)) {
			t.Error("FAIL bootstrap restore lost legacy bytes")
		}
	})
}

func TestRepairA12Containment(t *testing.T) {
	for _, name := range []string{"unowned", "case-fold", "symlink-file", "symlink-directory", "raced-directory", "intervening-edit", "stale-lock", "canceled", "prior-drift"} {
		t.Run(name, func(t *testing.T) {
			repo, options := fixture(t)
			before := publicBytes(t, repo)
			ctx := context.Background()
			var swapErr error
			checkpoint := false
			switch name {
			case "unowned":
				if err := os.WriteFile(filepath.Join(repo, "protocol", "b.go"), []byte("unowned"), 0o644); err != nil {
					t.Fatal(err)
				}
				before = publicBytes(t, repo)
			case "case-fold":
				options.Files = generation(t, map[string]string{"A.go": "new a"})
			case "symlink-file":
				if err := os.Symlink("a.go", filepath.Join(repo, "protocol", "b.go")); err != nil {
					t.Fatal(err)
				}
				before = publicBytes(t, repo)
			case "symlink-directory":
				if err := os.Symlink(t.TempDir(), filepath.Join(repo, "protocol", stateName)); err != nil {
					t.Fatal(err)
				}
			case "raced-directory":
				options.Hook = func(point, name string) error {
					if point == "rename" && name == "a.go" {
						checkpoint = true
						if err := os.Rename(filepath.Join(repo, "protocol"), filepath.Join(repo, "moved")); err != nil {
							swapErr = err
							return err
						}
						if err := os.Mkdir(filepath.Join(repo, "protocol"), 0o755); err != nil {
							return err
						}
						return os.WriteFile(filepath.Join(repo, "protocol", "sentinel.txt"), []byte("replacement"), 0o644)
					}
					return nil
				}
			case "intervening-edit":
				options.Revalidate = func() error { return errInjected }
			case "stale-lock":
				if err := os.WriteFile(filepath.Join(repo, "protocol", LockName), []byte("unknown owner; age is not authority"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "prior-drift":
				if err := os.WriteFile(filepath.Join(repo, "protocol", "a.go"), []byte("manual"), 0o644); err != nil {
					t.Fatal(err)
				}
				before = publicBytes(t, repo)
			}
			_, publicationErr := Publish(ctx, repo, options)
			if publicationErr == nil {
				t.Errorf("FAIL containment/conflict guard: %s", name)
			}
			if name == "raced-directory" {
				if !checkpoint {
					t.Fatal("FAIL directory race checkpoint not reached")
				}
				if swapErr != nil {
					t.Logf("native directory swap refusal: %v", swapErr)
					denied := errors.Is(swapErr, fs.ErrPermission) || errors.Is(swapErr, syscall.Errno(32))
					if runtime.GOOS != "windows" || !denied || !errors.Is(publicationErr, swapErr) {
						t.Errorf("FAIL unexpected directory swap error: %v / %v", swapErr, publicationErr)
					}
					if !reflect.DeepEqual(before, publicBytes(t, repo)) {
						t.Error("FAIL refused native directory swap changed original bytes")
					}
					if _, err := os.Lstat(filepath.Join(repo, "moved")); !errors.Is(err, os.ErrNotExist) {
						t.Error("FAIL refused native directory swap created moved destination")
					}
					return
				}
				body, err := os.ReadFile(filepath.Join(repo, "protocol", "sentinel.txt"))
				if err != nil || string(body) != "replacement" {
					t.Error("FAIL raced directory sentinel touched")
				}
				body, err = os.ReadFile(filepath.Join(repo, "moved", "a.go"))
				if err != nil || string(body) != "old a" {
					t.Error("FAIL raced destination mutated")
				}
			} else if !reflect.DeepEqual(before, publicBytes(t, repo)) {
				t.Error("FAIL refused publication changed public bytes")
			}
		})
	}
}

func TestRepairA12OpenRace(t *testing.T) {
	repo, options := fixture(t)
	before := publicBytes(t, repo)
	replacement := maps.Clone(before)
	replacement["sentinel.txt"] = "replacement"
	attempted := false
	options.Hook = func(point, _ string) error {
		if point != "before-root-open" {
			return nil
		}
		attempted = true
		if err := os.Rename(filepath.Join(repo, "protocol"), filepath.Join(repo, "moved")); err != nil {
			return err
		}
		if err := os.Mkdir(filepath.Join(repo, "protocol"), 0o755); err != nil {
			return err
		}
		for name, body := range replacement {
			if err := os.WriteFile(filepath.Join(repo, "protocol", name), []byte(body), 0o644); err != nil {
				return err
			}
		}
		return nil
	}
	_, err := Publish(context.Background(), repo, options)
	if !attempted || err == nil || !strings.Contains(err.Error(), "directory identity changed") {
		t.Errorf("FAIL open directory race guard: attempted=%t error=%v", attempted, err)
	}
	if !reflect.DeepEqual(replacement, publicBytes(t, repo)) {
		t.Error("FAIL open directory race changed replacement bytes")
	}
	for name, want := range before {
		body, err := os.ReadFile(filepath.Join(repo, "moved", name))
		if err != nil || string(body) != want {
			t.Errorf("FAIL open directory race changed moved original: %s %v", name, err)
		}
	}
}

func TestRepairA13Faults(t *testing.T) {
	for _, fault := range []struct {
		point, name string
		err         error
	}{
		{"write", filePending, syscall.ENOSPC}, {"sync", filePending, errInjected}, {"close", filePending, errInjected},
		{"rename", "a.go", errInjected}, {"delete", "stale.go", fs.ErrPermission},
		{"rename", ownership.ManifestName, errInjected}, {"cleanup", "", errInjected},
		{"directory-sync", ".", errInjected}, {"journal", JournalName, errInjected},
	} {
		t.Run(fault.point+"-"+fault.name, func(t *testing.T) {
			repo, options := fixture(t)
			before := publicBytes(t, repo)
			options.Hook = func(point, name string) error {
				if point == fault.point && (fault.name == "" || name == fault.name) {
					return fault.err
				}
				return nil
			}
			if _, err := Publish(context.Background(), repo, options); err == nil {
				t.Error("FAIL filesystem fault accepted")
			}
			journal, err := Inspect(repo)
			if err != nil {
				t.Fatal(err)
			}
			if journal != nil {
				if _, err := Recover(context.Background(), repo, "restore", nil); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(before, publicBytes(t, repo)) {
				t.Error("FAIL fault recovery lost original bytes")
			}
		})
	}
}

func TestRepairA13ManualAndCorrupt(t *testing.T) {
	for _, kind := range []string{"new-manual", "old-manual", "corrupt-stage", "corrupt-journal", "traversing-journal", "unowned-journal-entry"} {
		for _, action := range []string{"complete", "restore"} {
			t.Run(kind+"-"+action, func(t *testing.T) {
				repo, options := fixture(t)
				options.Hook = stop("after-replace", "b.go")
				if _, err := Publish(context.Background(), repo, options); !errors.Is(err, errInjected) {
					t.Fatal(err)
				}
				journal, err := Inspect(repo)
				if err != nil || journal == nil {
					t.Fatalf("FAIL inspect: %v", err)
				}
				switch kind {
				case "new-manual", "old-manual":
					name := "b.go"
					if kind == "old-manual" {
						name = "a.go"
					}
					if err := os.WriteFile(filepath.Join(repo, "protocol", name), []byte("manual edit"), 0o644); err != nil {
						t.Fatal(err)
					}
				case "corrupt-stage":
					if err := os.WriteFile(filepath.Join(repo, "protocol", stagePath(journal.Token, journal.Files[0].After)), []byte("corrupt"), 0o600); err != nil {
						t.Fatal(err)
					}
				case "corrupt-journal":
					if err := os.WriteFile(filepath.Join(repo, "protocol", JournalName), []byte(`{broken`), 0o600); err != nil {
						t.Fatal(err)
					}
				case "traversing-journal", "unowned-journal-entry":
					if kind == "traversing-journal" {
						journal.Files[0].Name = "../sentinel.txt"
					} else {
						journal.Files[0].Name = "sentinel.txt"
					}
					body, err := json.Marshal(journal)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(repo, "protocol", JournalName), body, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				before := publicBytes(t, repo)
				if _, err := Recover(context.Background(), repo, action, nil); err == nil {
					t.Errorf("FAIL manual/corrupt recovery accepted: %s", kind)
				}
				if !reflect.DeepEqual(before, publicBytes(t, repo)) {
					t.Error("FAIL recovery overwrote manual/unrelated bytes")
				}
			})
		}
	}
}

func TestRepairA13KilledProcess(t *testing.T) {
	for _, point := range []string{"staged", "after-replace", "cleanup"} {
		for _, action := range []string{"complete", "restore"} {
			t.Run(point+"-"+action, func(t *testing.T) {
				repo, options := fixture(t)
				before := publicBytes(t, repo)
				nonce, err := token()
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(repo, ".publication-child-token"), []byte(nonce), 0o600); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPublicationChild$", "-test.timeout=12s")
				child.Env = append(os.Environ(), "MCP_PUBLICATION_CHILD="+repo, "MCP_PUBLICATION_STOP="+point, "MCP_PUBLICATION_NONCE="+nonce)
				var diagnostic strings.Builder
				child.Stderr = &diagnostic
				stdout, err := child.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				if err := child.Start(); err != nil {
					t.Fatal(err)
				}
				reader := bufio.NewReader(stdout)
				ready, readErr := reader.ReadString('\n')
				buffered, err := reader.Peek(reader.Buffered())
				if err != nil {
					t.Fatal(err)
				}
				if ready == "publication-checkpoint\n" {
					if _, err := Recover(context.Background(), repo, action, nil); err == nil {
						t.Error("FAIL live generator lock stolen")
					}
				}
				killErr := child.Process.Kill()
				waitErr := child.Wait()
				if readErr != nil || ready != "publication-checkpoint\n" || killErr != nil || waitErr == nil {
					t.Fatalf("FAIL actual child interruption: %q %v %v %v\n%s%s", ready, readErr, killErr, waitErr, buffered, diagnostic.String())
				}
				if _, err := Recover(context.Background(), repo, action, nil); err == nil {
					t.Fatal("FAIL killed owner lock stolen")
				}
				// The fixture operator owns this child and has reaped it. Production
				// never guesses liveness or removes an unknown/stale lock by age.
				body, err := os.ReadFile(filepath.Join(repo, "protocol", LockName))
				if err != nil {
					t.Fatal(err)
				}
				var owner struct {
					PID int `json:"pid"`
				}
				if err := json.Unmarshal(body, &owner); err != nil || owner.PID != child.Process.Pid {
					t.Fatal("FAIL child lock identity")
				}
				if err := os.Remove(filepath.Join(repo, "protocol", LockName)); err != nil {
					t.Fatal(err)
				}
				if _, err := Recover(context.Background(), repo, action, nil); err != nil {
					t.Fatal(err)
				}
				if action == "restore" {
					if !reflect.DeepEqual(before, publicBytes(t, repo)) {
						t.Error("FAIL killed-child restore lost originals")
					}
				} else {
					for name, body := range options.Files {
						if publicBytes(t, repo)[name] != string(body) {
							t.Errorf("FAIL killed-child completion: %s", name)
						}
					}
				}
			})
		}
	}
}

func TestPublicationChild(t *testing.T) {
	repo := os.Getenv("MCP_PUBLICATION_CHILD")
	if repo == "" {
		return
	}
	canonical, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	// Go 1.27 testing.TempDir uses GOTMPDIR when configured, then os.TempDir.
	// Match that exact root while retaining the nonce and fixed-byte guards.
	temporaryRoot := os.Getenv("GOTMPDIR")
	if temporaryRoot == "" {
		temporaryRoot = os.TempDir()
	}
	base, err := filepath.EvalSymlinks(temporaryRoot)
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(base, canonical)
	if err != nil || !filepath.IsLocal(relative) || relative == "." {
		t.Fatal("FAIL child fixture is not a private temporary tree")
	}
	nonce, err := os.ReadFile(filepath.Join(repo, ".publication-child-token"))
	if err != nil || !validToken(string(nonce)) || string(nonce) != os.Getenv("MCP_PUBLICATION_NONCE") {
		t.Fatal("FAIL child fixture capability")
	}
	files := generation(t, map[string]string{"a.go": "new a", "b.go": "new b", "z.go": "new z"})
	root, err := os.OpenRoot(filepath.Join(repo, "protocol"))
	if err != nil {
		t.Fatal(err)
	}
	previous, _, err := ownership.Existing(root, "")
	if closeErr := root.Close(); err != nil || closeErr != nil {
		t.Fatal(errors.Join(err, closeErr))
	}
	expected, err := ownership.Candidate(generation(t, map[string]string{"a.go": "old a", "stale.go": "old stale", "z.go": "old z"}))
	if err != nil || !maps.Equal(previous, expected) {
		t.Fatal("FAIL child fixture does not hold independently fixed test bytes")
	}
	_, err = Publish(context.Background(), repo, Options{Previous: previous, Files: files, Hook: func(point, name string) error {
		if point == os.Getenv("MCP_PUBLICATION_STOP") && (point != "after-replace" || name == "a.go") {
			fmt.Println("publication-checkpoint")
			select {}
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Fatal("FAIL child did not reach checkpoint")
}
