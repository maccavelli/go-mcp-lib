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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// ExcludeStale removes files in dst that the manifest owns but the new
// emission no longer contains, so stale outputs never enter the candidate.
func ExcludeStale(dst string, manifestFiles []string) error {
	data, err := os.ReadFile(filepath.Join(dst, "generated-manifest.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var old Manifest
	if err := json.Unmarshal(data, &old); err != nil {
		return fmt.Errorf("qualify: stale manifest unreadable: %w", err)
	}
	current := make(map[string]bool, len(manifestFiles))
	for _, name := range manifestFiles {
		current[name] = true
	}
	for _, name := range old.Files {
		if current[name] {
			continue
		}
		path := filepath.Join(dst, name)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("qualify: removing stale %s: %w", name, err)
		}
	}
	return nil
}

// Manifest is the ownership record written alongside generated outputs.
type Manifest struct {
	Revision string   `json:"revision"`
	Files    []string `json:"files"`
}
