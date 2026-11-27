package qualify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
)

func fakeRunner(failOS string) Runner {
	return func(env []string, dir string, args ...string) (int, string) {
		for _, e := range env {
			if failOS != "" && e == "GOOS="+failOS {
				return 1, "planted compile failure for " + failOS
			}
		}
		return 0, ""
	}
}

func TestAllTargetsPass(t *testing.T) {
	report, err := CompileTree(fakeRunner(""), t.TempDir(), "module")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed {
		t.Fatalf("FAIL report: %+v", report)
	}
	if len(report.Targets) != 3 {
		t.Fatalf("FAIL targets: %d", len(report.Targets))
	}
	for _, result := range report.Targets {
		if !result.Package || !result.Tests {
			t.Errorf("FAIL target %s: %+v", result.Target, result)
		}
	}
}

func TestPlatformPlantFailsReport(t *testing.T) {
	report, err := CompileTree(fakeRunner("windows"), t.TempDir(), "module")
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed {
		t.Fatal("FAIL windows compile plant reported pass")
	}
	for _, result := range report.Targets {
		if result.Target == "windows/amd64" && result.Detail == "" {
			t.Error("FAIL plant detail empty")
		}
	}
}

func TestSplitTarget(t *testing.T) {
	for _, target := range Targets {
		if splitOS(target)+"/"+splitArch(target) != target {
			t.Errorf("FAIL split %s -> %s/%s", target, splitOS(target), splitArch(target))
		}
	}
}

func TestExcludeStaleRemovesOrphan(t *testing.T) {
	dst := t.TempDir()
	oldManifest := ownership.Manifest{Version: 2, Revision: "r1", Files: []string{"aliases.go", "gone.go", "owned.go"}, SHA256: map[string]string{}}
	for _, name := range oldManifest.Files {
		oldManifest.SHA256[name] = ownership.Digest([]byte("x"))
	}
	data, err := json.Marshal(oldManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "generated-manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range oldManifest.Files {
		if err := os.WriteFile(filepath.Join(dst, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := ExcludeStale(dst, []string{"owned.go", "aliases.go"}); err != nil {
		t.Fatalf("FAIL exclude: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "gone.go")); !os.IsNotExist(err) {
		t.Fatal("FAIL stale file survived exclusion")
	}
	for _, kept := range []string{"owned.go", "aliases.go"} {
		if _, err := os.Stat(filepath.Join(dst, kept)); err != nil {
			t.Fatalf("FAIL current file removed: %v", err)
		}
	}
}

func TestExcludeStaleNoManifestIsNoop(t *testing.T) {
	if err := ExcludeStale(t.TempDir(), nil); err != nil {
		t.Fatalf("FAIL noop: %v", err)
	}
}

func TestOSRunnerRealBuild(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.invalid/rt\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	report, err := CompileTree(OSRunner(), cache, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed {
		for _, r := range report.Targets {
			t.Logf("%s: %+v", r.Target, r)
		}
		t.Fatal("FAIL real three-platform compile did not pass")
	}
}

func TestOSRunnerRealBrokenModuleFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.invalid/brk\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { undefinedCall() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := CompileTree(OSRunner(), t.TempDir(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed {
		t.Fatal("FAIL broken module reported pass")
	}
}
