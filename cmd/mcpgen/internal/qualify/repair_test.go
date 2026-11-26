package qualify

import (
	"slices"
	"testing"
)

func TestRepairA11CandidateQualification(t *testing.T) {
	calls := 0
	runner := func(env []string, dir string, args ...string) (int, string) {
		calls++
		for _, required := range []string{"CGO_ENABLED=0", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=readonly", "GOENV=off"} {
			if !slices.Contains(env, required) {
				t.Errorf("FAIL candidate environment missing %s", required)
			}
		}
		if args[0] != "build" && (args[0] != "test" || !slices.Contains(args, "-c")) {
			t.Error("FAIL foreign candidate binary could execute")
		}
		return 0, ""
	}
	report, err := CompileTree(runner, t.TempDir(), "candidate")
	if err != nil || !report.Passed || calls != 6 {
		t.Fatalf("FAIL incomplete candidate target/component matrix: %+v %v calls=%d", report, err, calls)
	}
}
