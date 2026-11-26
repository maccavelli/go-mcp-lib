package qualify

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func candidateWrite(t *testing.T, root, name string, body []byte) {
	t.Helper()
	destination := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func proposedFixture(t *testing.T) map[string][]byte {
	t.Helper()
	body := []byte("package protocol\nconst Generated = 1\n")
	manifest := outputManifest{Version: 2, Revision: "fixture", Files: []string{"owned.go"}, SHA256: map[string]string{"owned.go": contentSHA256(body)}}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{"owned.go": body, "generated-manifest.json": encoded}
}

func candidateFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{
		"go.mod":               "module example.invalid/candidate\ngo 1.27\n",
		"handwritten.go":       "package fixture\nimport p \"example.invalid/candidate/protocol\"\nvar Value = p.Generated\n",
		"internal_test.go":     "package fixture\nimport \"testing\"\nfunc TestInternal(t *testing.T) { if Value != 1 { t.Fatal(Value) } }\n",
		"external_test.go":     "package fixture_test\nimport (\"testing\"; f \"example.invalid/candidate\")\nfunc TestExternal(t *testing.T) { if f.Value != 1 { t.Fatal(f.Value) } }\n",
		"testdata/fixture.txt": "embedded context\n",
		"cmd/mcpgen/go.mod":    "module example.invalid/generator\ngo 1.27\n",
		"cmd/mcpgen/main.go":   "package generator\nconst Value = 1\n",
	} {
		candidateWrite(t, root, name, []byte(body))
	}
	return root
}

func TestRepairA11CompleteSnapshot(t *testing.T) {
	root := candidateFixture(t)
	candidateWrite(t, root, "protocol/doc.go", []byte("// Package protocol documents the handwritten boundary.\npackage protocol\n"))
	candidate, err := Assemble(root, proposedFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := candidate.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, name := range []string{"handwritten.go", "internal_test.go", "external_test.go", "testdata/fixture.txt", "cmd/mcpgen/main.go", "protocol/doc.go", "protocol/owned.go", "protocol/generated-manifest.json"} {
		if body, err := os.ReadFile(filepath.Join(candidate.Directory(), filepath.FromSlash(name))); err != nil || len(body) == 0 {
			t.Errorf("FAIL complete candidate omitted %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "protocol/owned.go")); !os.IsNotExist(err) {
		t.Error("FAIL candidate published into source")
	}
	calls := 0
	runner := func(env []string, dir string, args ...string) (int, string) {
		calls++
		if !strings.HasPrefix(dir, candidate.Directory()) {
			t.Error("FAIL compiler received live source")
		}
		return 0, ""
	}
	reports, err := candidate.Compile(runner, t.TempDir())
	if err != nil || !AllPassed(reports) || calls != 12 {
		t.Errorf("FAIL complete module/target matrix: calls=%d reports=%+v error=%v", calls, reports, err)
	}
	candidateWrite(t, root, "handwritten.go", []byte("changed by owner\n"))
	if err := candidate.VerifySource(); err == nil {
		t.Error("FAIL intervening source edit accepted")
	}
}

func TestRepairA11OwnershipAndContainment(t *testing.T) {
	for _, name := range []string{"unowned", "case-fold", "traversal", "bad-hash", "escaping-link", "forged-manifest", "tampered-legacy"} {
		t.Run(name, func(t *testing.T) {
			root := candidateFixture(t)
			proposed := proposedFixture(t)
			switch name {
			case "unowned":
				candidateWrite(t, root, "protocol/owned.go", []byte("handwritten"))
			case "case-fold":
				candidateWrite(t, root, "protocol/OWNED.go", []byte("handwritten"))
			case "traversal":
				var manifest outputManifest
				if err := json.Unmarshal(proposed["generated-manifest.json"], &manifest); err != nil {
					t.Fatal(err)
				}
				manifest.Files = []string{"../escape.go"}
				manifest.SHA256 = map[string]string{"../escape.go": contentSHA256(proposed["owned.go"])}
				proposed["../escape.go"] = proposed["owned.go"]
				delete(proposed, "owned.go")
				body, err := json.Marshal(manifest)
				if err != nil {
					t.Fatal(err)
				}
				proposed["generated-manifest.json"] = body
			case "bad-hash":
				proposed["owned.go"] = []byte("tampered proposed bytes")
			case "escaping-link":
				outside := filepath.Join(t.TempDir(), "outside.go")
				if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(root, "escape.go")); err != nil {
					t.Fatalf("FAIL required symlink fixture: %v", err)
				}
			case "forged-manifest":
				candidateWrite(t, root, "protocol/generated-manifest.json", []byte(`{"files":["doc.go"]}`))
			case "tampered-legacy":
				copyLegacyFixture(t, root)
				candidateWrite(t, root, "protocol/owned.go", []byte("manual edit"))
			}
			if candidate, err := Assemble(root, proposed); err == nil {
				_ = candidate.Close()
				t.Errorf("FAIL candidate ownership/containment guard accepted %s", name)
			}
			if body, err := os.ReadFile(filepath.Join(root, "handwritten.go")); err != nil || !strings.Contains(string(body), "p.Generated") {
				t.Errorf("FAIL candidate failure changed source: %s %v", body, err)
			}
		})
	}
	t.Run("verified-historical-migration", func(t *testing.T) {
		root := candidateFixture(t)
		copyLegacyFixture(t, root)
		candidate, err := Assemble(root, proposedFixture(t))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = candidate.Close() }()
		for _, name := range []string{"aliases.go", "constants.go"} {
			if _, err := os.Stat(filepath.Join(candidate.Directory(), "protocol", name)); !os.IsNotExist(err) {
				t.Errorf("FAIL verified stale file retained in scratch: %s", name)
			}
			if _, err := os.Stat(filepath.Join(root, "protocol", name)); err != nil {
				t.Errorf("FAIL source migration file removed: %s", name)
			}
		}
	})
	t.Run("exact-current-generation", func(t *testing.T) {
		root := candidateFixture(t)
		proposed := proposedFixture(t)
		for name, body := range proposed {
			candidateWrite(t, root, "protocol/"+name, body)
		}
		candidate, err := Assemble(root, proposed)
		if err != nil {
			t.Fatal(err)
		}
		if err := candidate.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func copyLegacyFixture(t *testing.T, root string) {
	t.Helper()
	for _, name := range []string{"aliases.go", "constants.go", "owned.go", "generated-manifest.json"} {
		fixture := name
		if strings.HasSuffix(name, ".go") {
			fixture += ".txt"
		}
		body, err := os.ReadFile(filepath.Join("testdata/legacy", fixture))
		if err != nil {
			t.Fatal(err)
		}
		candidateWrite(t, root, "protocol/"+name, body)
	}
}

func TestRepairA11RealPlatformPlant(t *testing.T) {
	root := candidateFixture(t)
	candidateWrite(t, root, "fault_windows.go", []byte("package fixture\nvar PlatformFault = missingPlatformSymbol\n"))
	candidate, err := Assemble(root, proposedFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = candidate.Close() }()
	cache, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	reports, err := candidate.Compile(OSRunner(), strings.TrimSpace(string(cache)))
	if err != nil {
		t.Fatal(err)
	}
	if AllPassed(reports) {
		t.Error("FAIL complete candidate accepted real Windows compile plant")
	}
	for _, report := range reports {
		for _, target := range report.Targets {
			shouldFail := report.Candidate == "." && target.Target == "windows/amd64"
			if shouldFail {
				if target.Package || target.Tests || !strings.Contains(target.Detail, "missingPlatformSymbol") {
					t.Errorf("FAIL platform plant was not attributed: %+v", target)
				}
			} else if !target.Package || !target.Tests {
				t.Errorf("FAIL valid platform/module control: %s %+v", report.Candidate, target)
			}
		}
	}
	if err := candidate.VerifySource(); err != nil {
		t.Errorf("FAIL platform rejection changed source: %v", err)
	}
}

func TestRepairA11ReportAccounting(t *testing.T) {
	makeReports := func() []*Report {
		var result []*Report
		for _, module := range []string{".", "cmd/mcpgen"} {
			report := &Report{Candidate: module, Passed: true}
			for _, target := range Targets {
				report.Targets = append(report.Targets, Result{Target: target, Package: true, Tests: true})
			}
			result = append(result, report)
		}
		return result
	}
	if !AllPassed(makeReports()) {
		t.Fatal("FAIL complete qualification control rejected")
	}
	for _, name := range []string{"empty", "missing-module", "false-summary", "missing-target", "duplicate-target", "package-failed", "tests-failed"} {
		t.Run(name, func(t *testing.T) {
			reports := makeReports()
			switch name {
			case "empty":
				reports = nil
			case "missing-module":
				reports = reports[:1]
			case "false-summary":
				reports[0].Passed = false
			case "missing-target":
				reports[0].Targets = reports[0].Targets[:2]
			case "duplicate-target":
				reports[0].Targets[1].Target = reports[0].Targets[0].Target
			case "package-failed":
				reports[0].Targets[0].Package = false
			case "tests-failed":
				reports[0].Targets[0].Tests = false
			}
			if AllPassed(reports) {
				t.Errorf("FAIL incomplete candidate qualification accepted: %s", name)
			}
		})
	}
}
