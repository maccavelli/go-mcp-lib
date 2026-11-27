package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/publish"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/qualify"
)

func cliCapability(repo, nonce string) error {
	root, err := filepath.EvalSymlinks(repo)
	if err != nil {
		return err
	}
	contained := false
	for _, temporary := range []string{os.TempDir(), os.Getenv("GOTMPDIR")} {
		if temporary == "" {
			continue
		}
		base, err := filepath.EvalSymlinks(temporary)
		if err != nil {
			continue
		}
		relative, err := filepath.Rel(base, root)
		if err == nil && relative != "." && filepath.IsLocal(relative) {
			contained = true
		}
	}
	if !contained {
		return fmt.Errorf("CLI fixture outside temporary roots")
	}
	body, err := os.ReadFile(filepath.Join(root, ".cli-fixture-capability"))
	decoded, decodeErr := hex.DecodeString(nonce)
	if err != nil || decodeErr != nil || len(decoded) != 16 || string(body) != nonce {
		return fmt.Errorf("CLI fixture capability missing or different")
	}
	_, err = profile.Load(root, profile.Revision)
	return err
}

func cliPublicationChild() int {
	repo, nonce, mode := os.Getenv("GO_MCP_CLI_REPO"), os.Getenv("GO_MCP_CLI_NONCE"), os.Getenv("GO_MCP_CLI_OPERATION")
	if err := cliCapability(repo, nonce); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 99
	}
	args := []string{"generate", "--repo", repo, "--format", "json"}
	switch mode {
	case "bootstrap", "staged", "after-replace", "cleanup":
		args = append(args, "--bootstrap-legacy")
	case "generate":
	case "check", "inspect":
		args = []string{mode, "--repo", repo, "--format", "json"}
	case "complete", "restore", "journal":
		action := mode
		if mode == "journal" {
			action = "inspect"
		}
		args = []string{"recover", "--repo", repo, "--action", action, "--format", "json"}
	default:
		return 99
	}
	out := newPrinter("json")
	out.publicationHook = func(point, name string) error {
		if point == mode && (mode != "after-replace" || name == "aliases.go") {
			if mode == "cleanup" {
				body, err := os.ReadFile(filepath.Join(repo, "protocol", ownership.ManifestName))
				if err != nil {
					return err
				}
				if ownership.Digest(body) == ownership.LegacyManifestSHA256 {
					return nil
				}
			}
			if _, err := fmt.Fprintln(os.Stdout, "CLI_CHECKPOINT "+point); err != nil {
				return err
			}
			select {}
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	return runCLIPrinter(ctx, args, out)
}

func TestCLIPublicationChild(t *testing.T) {
	if os.Getenv("GO_MCP_CLI_OPERATION") != "" {
		os.Exit(cliPublicationChild())
	}
}

func cliScratch(t *testing.T) (string, string) {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := profile.Load(repo, profile.Revision)
	if err != nil {
		t.Fatal(err)
	}
	sdk, err := pinnedSDKDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	scratch := t.TempDir()
	fresh, err := renderProtocolInputs(repo, sdk, scratch, inputs)
	if err != nil {
		t.Fatal(err)
	}
	fresh[ownership.ManifestName], err = os.ReadFile(filepath.Join(scratch, ownership.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := qualify.Assemble(repo, fresh)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := candidate.Close(); err != nil {
			t.Error(err)
		}
	})
	root := candidate.Directory()
	// Remove only generated files created in this owned scratch snapshot.
	for name := range fresh {
		if err := os.Remove(filepath.Join(root, "protocol", name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"aliases.go", "constants.go", "owned.go", ownership.ManifestName} {
		fixture := name
		if strings.HasSuffix(name, ".go") {
			fixture += ".txt"
		}
		body, err := os.ReadFile(filepath.Join("internal/qualify/testdata/legacy", fixture))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "protocol", name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitdir, err := gitRead(context.Background(), repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		t.Fatal(err)
	}
	// Give the fixture its own HEAD and index. Only immutable Git objects are
	// borrowed read-only; tests never mutate the source repository's index.
	if _, err := gitRead(context.Background(), root, "init", "--quiet"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git/HEAD"), []byte(ownership.LegacyCommit+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git/objects/info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git/objects/info/alternates"), []byte(filepath.ToSlash(filepath.Join(strings.TrimSpace(string(gitdir)), "objects"))+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := gitRead(context.Background(), root, "read-tree", ownership.LegacyCommit); err != nil {
		t.Fatal(err)
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	nonce := hex.EncodeToString(random[:])
	if err := os.WriteFile(filepath.Join(root, ".cli-fixture-capability"), []byte(nonce), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "protocol/sentinel.txt"), []byte("unrelated"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, nonce
}

func cliChild(t *testing.T, repo, nonce, mode string) (*exec.Cmd, context.CancelFunc, *bytes.Buffer) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 170*time.Second)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIPublicationChild$", "-test.timeout=160s")
	cmd.Env = append(os.Environ(), "GO_MCP_CLI_REPO="+repo, "GO_MCP_CLI_NONCE="+nonce, "GO_MCP_CLI_OPERATION="+mode)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	return cmd, cancel, stderr
}

func runFixtureCLI(t *testing.T, repo, nonce, mode string, want int) result {
	t.Helper()
	cmd, cancel, stderr := cliChild(t, repo, nonce, mode)
	defer cancel()
	stdout, err := cmd.Output()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatal(err)
		}
		code = exit.ExitCode()
	}
	if code != want {
		t.Fatalf("FAIL actual CLI %s: got=%d want=%d stdout=%s stderr=%s", mode, code, want, stdout, stderr.Bytes())
	}
	var envelope result
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	if err := decoder.Decode(&envelope); err != nil || envelope.Version != 1 {
		t.Fatalf("FAIL actual CLI result %s: %v %s", mode, err, stdout)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Errorf("FAIL actual CLI multiple documents: %v", err)
	}
	return envelope
}

func protocolSnapshot(t *testing.T, repo string) map[string]string {
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

func TestRepairA13Recovery(t *testing.T) {
	repo, nonce := cliScratch(t)
	before := protocolSnapshot(t, repo)
	if err := verifyLegacyGit(context.Background(), repo); err != nil {
		t.Fatalf("FAIL verified Git fixture: %v", err)
	}
	if err := verifyLegacyGitReader(context.Background(), repo, func(context.Context, string, ...string) ([]byte, error) { return []byte("forged Git object"), nil }); err == nil {
		t.Error("FAIL bootstrap accepted wrong frozen Git bytes")
	}
	bad := filepath.Join(repo, ".fixture-bad-blob")
	if err := os.WriteFile(bad, []byte("staged different bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	object, err := gitRead(context.Background(), repo, "hash-object", "-w", bad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gitRead(context.Background(), repo, "update-index", "--cacheinfo", "100644", strings.TrimSpace(string(object)), "protocol/owned.go"); err != nil {
		t.Fatal(err)
	}
	if err := verifyLegacyGit(context.Background(), repo); err == nil {
		t.Error("FAIL bootstrap accepted staged owned change")
	}
	if _, err := gitRead(context.Background(), repo, "read-tree", ownership.LegacyCommit); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(bad); err != nil {
		t.Fatal(err)
	}
	runFixtureCLI(t, repo, nonce, "generate", 1)
	if !reflect.DeepEqual(before, protocolSnapshot(t, repo)) {
		t.Fatal("FAIL implicit bootstrap changed output")
	}
	runFixtureCLI(t, repo, nonce, "bootstrap", 0)
	after := protocolSnapshot(t, repo)
	if reflect.DeepEqual(before, after) || after["sentinel.txt"] != "unrelated" || after["api.json"] == "" {
		t.Fatal("FAIL actual CLI bootstrap did not install full contract")
	}
	info, err := os.Stat(filepath.Join(repo, "protocol/owned.go"))
	if err != nil {
		t.Fatal(err)
	}
	if result := runFixtureCLI(t, repo, nonce, "generate", 0); result.Status != "unchanged" {
		t.Error("FAIL repeated CLI generation changed inventory")
	}
	repeated, err := os.Stat(filepath.Join(repo, "protocol/owned.go"))
	if err != nil || !info.ModTime().Equal(repeated.ModTime()) {
		t.Error("FAIL repeated CLI generation changed mtime")
	}
	runFixtureCLI(t, repo, nonce, "check", 0)
	inspection := runFixtureCLI(t, repo, nonce, "inspect", 0)
	body, err := json.Marshal(inspection.Data)
	if err != nil || !bytes.Contains(body, []byte(`"coverage.json"`)) || !bytes.Contains(body, []byte(`"api.json"`)) {
		t.Error("FAIL actual CLI inspect lacks complete contract")
	}
	runFixtureCLI(t, repo, nonce, "restore", 0)
	if !reflect.DeepEqual(before, protocolSnapshot(t, repo)) {
		t.Fatal("FAIL actual CLI restore lost original bytes")
	}
	for _, test := range []struct{ point, action string }{{"staged", "restore"}, {"after-replace", "complete"}, {"cleanup", "restore"}} {
		t.Run(test.point+"-"+test.action, func(t *testing.T) {
			cmd, cancel, stderr := cliChild(t, repo, nonce, test.point)
			defer cancel()
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			line, readErr := bufio.NewReader(stdout).ReadString('\n')
			if readErr != nil || strings.TrimSpace(line) != "CLI_CHECKPOINT "+test.point {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				t.Fatalf("FAIL killed CLI checkpoint: %v %s %s", readErr, line, stderr.Bytes())
			}
			if err := cmd.Process.Kill(); err != nil {
				_ = cmd.Wait()
				t.Fatal(err)
			}
			_ = cmd.Wait()
			checkExit := 1
			if test.point == "cleanup" {
				checkExit = 0
			}
			runFixtureCLI(t, repo, nonce, "check", checkExit)
			runFixtureCLI(t, repo, nonce, test.action, 1)
			lockPath := filepath.Join(repo, "protocol", publish.LockName)
			lock, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatal(err)
			}
			var owner struct {
				PID int `json:"pid"`
			}
			if json.Unmarshal(lock, &owner) != nil || owner.PID != cmd.Process.Pid {
				t.Fatal("FAIL killed CLI lock owner mismatch")
			}
			// The parent reaped this exact owner: explicit fixture operator
			// resolution, never an age-based production lock steal.
			if err := os.Remove(lockPath); err != nil {
				t.Fatal(err)
			}
			runFixtureCLI(t, repo, nonce, test.action, 0)
			want := before
			if test.action == "complete" {
				want = after
			}
			if !maps.Equal(want, protocolSnapshot(t, repo)) {
				t.Error("FAIL killed CLI recovery changed expected inventory")
			}
			runFixtureCLI(t, repo, nonce, "restore", 0)
			if !maps.Equal(before, protocolSnapshot(t, repo)) {
				t.Fatal("FAIL killed CLI reset lost originals")
			}
		})
	}
}

func TestRepairA13GitBounds(t *testing.T) {
	var output gitOutput
	// io.Copy must not discover an embedded Buffer.ReadFrom that bypasses Write.
	_, err := io.Copy(&output, strings.NewReader(strings.Repeat("x", (8<<20)+1)))
	if err == nil || output.body.Len() > 8<<20 {
		t.Error("FAIL bootstrap Git output bound bypassed")
	}
}
