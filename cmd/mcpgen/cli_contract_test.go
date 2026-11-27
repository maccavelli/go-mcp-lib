package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Every child constructs its own fixed non-publishing case. An environment
// variable cannot supply CLI arguments or redirect a writer into a live tree.
func cliFixtureCase(mode string) int {
	repo, err := os.MkdirTemp("", "mcp-cli-case-")
	if err != nil {
		return 99
	}
	defer func() { _ = os.RemoveAll(repo) }()
	name, kind, ok := strings.Cut(mode, ":")
	if !ok {
		return 99
	}
	if _, ok := commands[name]; !ok {
		return 99
	}
	args := []string{name}
	switch kind {
	case "help":
		args = append(args, "--help")
	case "unknown":
		args = append(args, "--unknown-option")
	case "positional":
		args = append(args, "stray")
	case "format", "json":
		switch name {
		case "inspect", "check", "generate":
			args = append(args, "--repo", repo, "--manifest", "../escape.json")
		case "diff":
			args = append(args, "--from", filepath.Join(repo, "missing.json"), "--to", filepath.Join(repo, "missing2.json"))
		case "update":
			args = append(args, "--repo", repo, "--source", "https://example.invalid/main/schema.json", "--revision", "main", "--sha256", strings.Repeat("0", 64), "--candidate", filepath.Join(repo, "candidate"))
		case "recover":
			if err := os.Mkdir(filepath.Join(repo, "protocol"), 0o755); err != nil {
				return 99
			}
			if err := os.WriteFile(filepath.Join(repo, "protocol/.mcpgen-journal.json"), []byte(`{broken`), 0o644); err != nil {
				return 99
			}
			args = append(args, "--repo", repo, "--action", "inspect")
		case "diagnostic":
			args = append(args, "--repo", repo)
		}
		format := "invalid"
		if kind == "json" {
			format = "json"
		}
		args = append(args, "--format", format)
	default:
		return 99
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return runCLI(ctx, args)
}

func TestCLIContractChild(t *testing.T) {
	if mode := os.Getenv("GO_MCP_CLI_CASE"); mode != "" {
		os.Exit(cliFixtureCase(mode))
	}
}

func TestRepairA14CLI(t *testing.T) {
	for _, name := range commandNames() {
		for _, kind := range []string{"help", "unknown", "positional", "format", "json"} {
			t.Run(name+"-"+kind, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIContractChild$")
				cmd.Env = append(os.Environ(), "GO_MCP_CLI_CASE="+name+":"+kind)
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				err := cmd.Run()
				code := 0
				if err != nil {
					var exit *exec.ExitError
					if errors.As(err, &exit) {
						code = exit.ExitCode()
					} else {
						t.Fatal(err)
					}
				}
				want := 2
				if kind == "help" || kind == "json" && name == "diagnostic" {
					want = 0
				} else if kind == "json" {
					want = 1
				}
				if code != want {
					t.Errorf("FAIL CLI exit %s/%s: got=%d want=%d stdout=%s stderr=%s", name, kind, code, want, stdout.Bytes(), stderr.Bytes())
				}
				if kind == "json" {
					decoder := json.NewDecoder(&stdout)
					var value result
					if err := decoder.Decode(&value); err != nil || value.Version != 1 || value.Command != name {
						t.Errorf("FAIL CLI JSON envelope: %v %+v", err, value)
					}
					var extra any
					if err := decoder.Decode(&extra); err != io.EOF {
						t.Errorf("FAIL multiple/malformed CLI result documents: %v", err)
					}
					if want == 1 && stderr.Len() == 0 {
						t.Error("FAIL CLI failure lacks separate diagnostic")
					}
				} else if kind != "help" && stderr.Len() == 0 {
					t.Error("FAIL CLI usage lacks diagnostic")
				}
				if ctx.Err() != nil {
					t.Fatal(fmt.Errorf("FAIL CLI child timed out: %w", ctx.Err()))
				}
			})
		}
	}
}
