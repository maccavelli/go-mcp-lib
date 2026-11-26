package qualify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// commandLog bounds both diagnostic retention and total command output. Keeping
// the tail avoids retaining megabytes through a small Report.Detail substring.
type commandLog struct {
	mu           sync.Mutex
	data         []byte
	total, limit int
	exceeded     bool
	cancel       context.CancelFunc
}

func (log *commandLog) Write(body []byte) (int, error) {
	log.mu.Lock()
	defer log.mu.Unlock()
	written := len(body)
	log.total += len(body)
	const retained = 64 << 10
	if len(body) > retained {
		body = body[len(body)-retained:]
	}
	if excess := len(log.data) + len(body) - retained; excess > 0 {
		log.data = append(log.data[:0], log.data[excess:]...)
	}
	log.data = append(log.data, body...)
	if log.total > log.limit {
		log.exceeded = true
		log.cancel()
		return 0, fmt.Errorf("qualification command output limit exceeded")
	}
	return written, nil
}

func commandEnvironment(overrides []string) []string {
	values := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if runtime.GOOS == "windows" {
			key = strings.ToUpper(key)
		}
		if ok && key != "GOTOOLDIR" && key != "GOOS" && key != "GOARCH" {
			values[key] = value
		}
	}
	for _, entry := range overrides {
		key, value, ok := strings.Cut(entry, "=")
		if runtime.GOOS == "windows" {
			key = strings.ToUpper(key)
		}
		if ok {
			values[key] = value
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}

func runCommand(executable string, env []string, directory string, args []string, timeout time.Duration, limit int) (int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	log := &commandLog{limit: limit, cancel: cancel}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = directory
	cmd.Env = commandEnvironment(env)
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error { return cancelProcessTree(cmd.Process) }
	err := cmd.Run()
	log.mu.Lock()
	defer log.mu.Unlock()
	output := string(log.data)
	if err != nil {
		output += "\n" + err.Error()
	}
	if log.exceeded {
		return 125, output + "\nqualification command output limit exceeded"
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return 124, output + "\nqualification command timed out"
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode(), output
		}
		return 1, output
	}
	return 0, output
}

// BuildCache resolves the prepared compiler cache under the same offline policy.
func BuildCache() (string, error) {
	env := []string{}
	for _, entry := range GoEnv("") {
		if !strings.HasPrefix(entry, "GOCACHE=") {
			env = append(env, entry)
		}
	}
	code, output := runCommand("go", env, "", []string{"env", "GOCACHE"}, 10*time.Second, 4096)
	if code != 0 {
		return "", fmt.Errorf("qualify: compiler cache metadata failed: %s", tail(output, 256))
	}
	return strings.TrimSpace(output), nil
}
