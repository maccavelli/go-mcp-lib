package qualify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRepairA11BoundedRunner(t *testing.T) {
	const modeKey = "MCP_QUALIFICATION_TEST_MODE"
	if mode := os.Getenv(modeKey); mode != "" {
		marker := os.Getenv("MCP_QUALIFICATION_TEST_MARKER")
		switch mode {
		case "spawn":
			child := exec.Command(os.Args[0], "-test.run=^TestRepairA11BoundedRunner$")
			child.Env = commandEnvironment([]string{modeKey + "=heartbeat"})
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(marker+".pid", []byte(strconv.Itoa(child.Process.Pid)), 0o600); err != nil {
				t.Fatal(err)
			}
			time.Sleep(5 * time.Minute)
		case "heartbeat":
			for i := 0; i < 15000; i++ {
				if err := os.WriteFile(marker, []byte(strconv.FormatInt(time.Now().UnixNano(), 10)), 0o600); err != nil {
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
		case "spam":
			for i := 0; i < 100000; i++ {
				if _, err := fmt.Fprint(os.Stdout, strings.Repeat("x", 8192)); err != nil {
					return
				}
			}
		default:
			t.Fatalf("FAIL unknown helper mode %s", mode)
		}
		return
	}
	marker := filepath.Join(t.TempDir(), "heartbeat")
	code, output := runCommand(os.Args[0], []string{modeKey + "=spawn", "MCP_QUALIFICATION_TEST_MARKER=" + marker}, "", []string{"-test.run=^TestRepairA11BoundedRunner$"}, 2*time.Second, 4096)
	if code != 124 || !strings.Contains(output, "qualification command timed out") {
		t.Fatalf("FAIL compiler timeout not enforced: %d %s", code, output)
	}
	body, err := os.ReadFile(marker + ".pid")
	if err != nil {
		t.Fatalf("FAIL cancellation fixture did not start child: %v", err)
	}
	pid, err := strconv.Atoi(string(body))
	if err != nil {
		t.Fatal(err)
	}
	// The fallback protects the test environment when a deliberately broken
	// cancellation implementation is being exercised in a scratch copy.
	defer func() {
		if child, err := os.FindProcess(pid); err == nil {
			_ = child.Kill()
			_ = child.Release()
		}
	}()
	time.Sleep(100 * time.Millisecond)
	before, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	after, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("FAIL compiler timeout left a child running")
	}
	code, output = runCommand(os.Args[0], []string{modeKey + "=spam"}, "", []string{"-test.run=^TestRepairA11BoundedRunner$"}, 5*time.Second, 4096)
	if code != 125 || !strings.Contains(output, "qualification command output limit exceeded") {
		t.Errorf("FAIL compiler output limit not enforced: %d %s", code, output)
	}
}
