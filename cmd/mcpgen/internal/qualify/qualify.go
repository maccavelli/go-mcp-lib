// Package qualify compiles a complete scratch candidate tree offline and
// refuses to publish anything that does not build for every target
// platform.
//
// Qualification runs Go with GOWORK=off, GOTOOLCHAIN=local, GOPROXY=off in
// readonly module mode. Package sources and external tests compile for
// darwin/arm64, linux/amd64, and windows/amd64; foreign test binaries are
// never executed, only compiled. Stale owned outputs are excluded from the
// candidate by manifest ownership before anything runs.
package qualify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
)

// Targets are the three supported platforms; compile-only, never executed
// cross-platform.
var Targets = []string{
	"darwin/arm64", "linux/amd64", "windows/amd64",
}

// Result reports per-target compile outcomes.
type Result struct {
	Target  string `json:"target"`
	Package bool   `json:"package"`
	Tests   bool   `json:"tests"`
	Detail  string `json:"detail,omitempty"`
}

// Report is a qualification run.
type Report struct {
	Candidate string   `json:"candidate"`
	Targets   []Result `json:"targets"`
	Passed    bool     `json:"passed"`
}

// GoEnv is the offline environment qualification runs under.
func GoEnv(cache string) []string {
	return []string{
		"GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off",
		"CGO_ENABLED=0", "GOENV=off", "GOEXPERIMENT=",
		"GOCACHE=" + cache, "GOFLAGS=-mod=readonly",
	}
}

// Runner abstracts command execution for tests.
type Runner func(env []string, dir string, args ...string) (int, string)

// OSRunner runs real go commands through argument arrays.
func OSRunner() Runner {
	return CommandRunner("go")
}

// OSRunnerContext bounds all compiler children by the CLI invocation context.
func OSRunnerContext(ctx context.Context) Runner {
	return func(env []string, dir string, args ...string) (int, string) {
		return runCommandContext(ctx, "go", env, dir, args, 10*time.Minute, 32<<20)
	}
}

// CommandRunner applies the qualification timeout, output and process bounds to
// a prepared executable, including the gate-selected native analyzer.
func CommandRunner(executable string) Runner {
	return func(env []string, dir string, args ...string) (int, string) {
		return runCommand(executable, env, dir, args, 10*time.Minute, 32<<20)
	}
}

// CompileTree compiles package sources and test files for every target.
// moduleDir is the candidate root containing go.mod.
func CompileTree(runner Runner, cache, moduleDir string) (*Report, error) {
	results := make([]Result, 0, len(Targets))
	passed := true
	for _, target := range Targets {
		pkg := compileTarget(runner, cache, moduleDir, target, false)
		tests := compileTarget(runner, cache, moduleDir, target, true)
		entry := Result{
			Target: target, Package: pkg.Package, Tests: tests.Tests,
			Detail: firstDetail(pkg, tests),
		}
		if !entry.Package || !entry.Tests {
			passed = false
		}
		results = append(results, entry)
	}
	return &Report{Candidate: moduleDir, Targets: results, Passed: passed}, nil
}

func compileTarget(runner Runner, cache, moduleDir, target string, tests bool) Result {
	args := []string{"build"}
	if tests {
		args = []string{"test", "-vet=off", "-c", "-o", os.DevNull}
	}
	args = append(args, "./...")
	env := append(GoEnv(cache), "GOOS="+splitOS(target), "GOARCH="+splitArch(target))
	code, out := runner(env, moduleDir, args...)
	if code != 0 {
		return Result{Detail: tail(out, 400)}
	}
	return Result{Package: true, Tests: true}
}

func splitOS(target string) string { return target[:len(target)-len("/"+splitArch(target))] }
func splitArch(target string) string {
	for i := len(target) - 1; i >= 0; i-- {
		if target[i] == '/' {
			return target[i+1:]
		}
	}
	return target
}

func firstDetail(a, b Result) string {
	if a.Detail != "" {
		return a.Detail
	}
	return b.Detail
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.Clone(s[len(s)-n:])
}

// ExcludeStale removes only verified hashed stale output in an owned scratch
// directory. Production publication always uses the recoverable publisher.
func ExcludeStale(dst string, manifestFiles []string) (failure error) {
	root, err := os.OpenRoot(dst)
	if err != nil {
		return err
	}
	defer func() { failure = errors.Join(failure, root.Close()) }()
	previous, legacy, err := ownership.Existing(root, "")
	if err != nil {
		return err
	}
	if legacy {
		return fmt.Errorf("qualify: legacy deletion requires verified publication")
	}
	current := map[string]bool{ownership.ManifestName: true}
	for _, name := range manifestFiles {
		if err := ownership.Name(name); err != nil {
			return err
		}
		current[name] = true
	}
	for _, name := range ownership.Sorted(previous) {
		if current[name] {
			continue
		}
		body, err := ownership.Read(root, name)
		if err != nil || ownership.Digest(body) != previous[name] {
			return fmt.Errorf("qualify: stale output changed: %s", name)
		}
		if err := root.Remove(name); err != nil {
			return err
		}
	}
	return nil
}
