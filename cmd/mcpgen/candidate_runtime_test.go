package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/qualify"
)

// TestRepairA11ActualProduct exercises the real runtime in the complete proposed
// tree. It neither publishes generated files nor executes foreign binaries.
func TestRepairA11ActualProduct(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := profile.Load(repo, profile.Revision)
	if err != nil {
		t.Fatal(err)
	}
	sdkDir, err := pinnedSDKDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	scratch := t.TempDir()
	fresh, err := renderProtocolInputs(repo, sdkDir, scratch, inputs)
	if err != nil {
		t.Fatal(err)
	}
	fresh["generated-manifest.json"], err = os.ReadFile(filepath.Join(scratch, "generated-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := qualify.Assemble(repo, fresh)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = candidate.Close() }()
	cache, err := qualify.BuildCache()
	if err != nil {
		t.Fatal(err)
	}
	runner := qualify.OSRunner()
	reports, err := candidate.Compile(runner, cache)
	if err != nil {
		t.Fatalf("FAIL actual product compilation: %v", err)
	}
	if !qualify.AllPassed(reports) {
		for _, report := range reports {
			for _, target := range report.Targets {
				if !target.Package || !target.Tests {
					t.Errorf("FAIL actual product compilation: module=%s target=%s package=%t tests=%t\n%s", report.Candidate, target.Target, target.Package, target.Tests, target.Detail)
				}
			}
		}
		t.FailNow()
	}
	env := append(qualify.GoEnv(cache), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	linter := os.Getenv("GO_MCP_LINTER")
	if linter == "" {
		t.Fatal("FAIL actual product requires the gate-selected pinned linter")
	}
	// Reuse the runtime gate: one owner each for formatting, complete test
	// accounting, standalone vet and pinned lint. Only scratch is inspected.
	code, output := qualify.CommandRunner("python3")(env, candidate.Directory(), "-B", "scripts/go-precheck.py", "--module", "runtime")
	if code != 0 {
		t.Errorf("FAIL actual product runtime pre-add: exit %d\n%s", code, output)
	} else {
		t.Logf("actual product runtime pre-add: %s", output)
	}
	// Deliberately discard an owned validation error only in scratch. Changing
	// JSON packages is insufficient: both decoders invoke the owned hooks.
	source := filepath.Join(candidate.Directory(), "internal", "sdkbridge", "codec.go")
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	anchor := []byte("if err := json.Unmarshal(body, &result); err != nil {")
	if bytes.Count(body, anchor) != 1 {
		t.Fatal("FAIL owned decoder plant anchor changed")
	}
	original := bytes.Clone(body)
	body = bytes.Replace(body, anchor, []byte("if err := json.Unmarshal(body, &result); err != nil && len(body) == 0 {"), 1)
	if err := os.WriteFile(source, body, 0o644); err != nil {
		t.Fatal(err)
	}
	code, output = runner(env, candidate.Directory(), "test", "-vet=off", "-count=1", "-run", "^TestOwnedResultBoundary$", ".")
	if code == 0 || !bytes.Contains([]byte(output), []byte("FAIL invalid owned result accepted or unclassified")) {
		t.Errorf("FAIL owned decoder bypass was not detected: exit %d\n%s", code, output)
	} else {
		t.Log("owned decoder bypass caught: FAIL invalid owned result accepted or unclassified")
	}
	if err := os.WriteFile(source, original, 0o644); err != nil {
		t.Fatal(err)
	}
	server := filepath.Join(candidate.Directory(), "server.go")
	body, err = os.ReadFile(server)
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, []byte("\nfunc (s *Server) UpstreamForFixture() *sdk.Server { return s.sdk }\n")...)
	if err := os.WriteFile(server, body, 0o644); err != nil {
		t.Fatal(err)
	}
	code, output = runner(env, candidate.Directory(), "test", "-vet=off", "-count=1", "-run", "^TestOwnedPublicAPI$", ".")
	if code == 0 || !bytes.Contains([]byte(output), []byte("FAIL public SDK reachability")) {
		t.Errorf("FAIL SDK runtime accessor plant was not detected: exit %d\n%s", code, output)
	} else {
		t.Log("SDK runtime accessor caught: FAIL public SDK reachability")
	}
	// Each type guard has its own compilable bypass and unchanged fixture.
	bridge := filepath.Join(candidate.Directory(), "middleware_bridge.go")
	clean, err := os.ReadFile(bridge)
	if err != nil {
		t.Fatal(err)
	}
	for _, plant := range []struct{ name, anchor, diagnostic string }{
		{"parameter-type", "if forwarded != method || request == nil || reflect.TypeOf(request.Parameters()) != parameterType {", "FAIL middleware parameter-type guard"},
		{"wrong-result-type", "if reflect.TypeOf(source) != reflect.TypeOf(expected) {", "FAIL middleware wrong-result-type guard"},
		{"complete-result", "source = projectSDKCompleteResult(source)", "FAIL SDK complete-result projection"},
	} {
		anchor := []byte(plant.anchor)
		if !bytes.Contains(clean, anchor) {
			t.Fatalf("FAIL middleware %s plant anchor changed", plant.name)
		}
		// The parameter-type plant retains the type computation so unused-name
		// compilation errors cannot masquerade as observed guard failures.
		replacement := []byte("if false {")
		if plant.name == "parameter-type" {
			replacement = []byte("_ = parameterType\nif false {")
		}
		test := "^TestOwnedMiddlewareBoundary$/^" + plant.name + "$"
		if plant.name == "complete-result" {
			replacement = []byte("// deliberately omit the SDK finalization projection")
			test = "^TestSDKResultProjection$"
		}
		body := bytes.Replace(clean, anchor, replacement, 1)
		if err := os.WriteFile(bridge, body, 0o644); err != nil {
			t.Fatal(err)
		}
		code, output := runner(env, candidate.Directory(), "test", "-vet=off", "-count=1", "-run", test, ".")
		if code == 0 || !bytes.Contains([]byte(output), []byte(plant.diagnostic)) {
			t.Errorf("FAIL middleware %s bypass was not detected: exit %d\n%s", plant.name, code, output)
		} else {
			t.Logf("middleware %s bypass caught: %s", plant.name, plant.diagnostic)
		}
	}
	if err := candidate.VerifySource(); err != nil {
		t.Fatal(err)
	}
}
