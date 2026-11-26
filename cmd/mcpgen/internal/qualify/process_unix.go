//go:build unix

package qualify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Keep children in the enclosing gate's process group. Its cancellation must
// still own this command if the generator process itself is terminated.
func cancelProcessTree(process *os.Process) error {
	if process == nil {
		return os.ErrProcessDone
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	body, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return errors.Join(fmt.Errorf("qualify: process-tree inspection failed: %w", err), process.Kill())
	}
	children := map[int][]int{}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 {
			return errors.Join(fmt.Errorf("qualify: malformed process-tree metadata"), process.Kill())
		}
		pid, pidErr := strconv.Atoi(fields[0])
		parent, parentErr := strconv.Atoi(fields[1])
		if pidErr != nil || parentErr != nil {
			return errors.Join(fmt.Errorf("qualify: invalid process-tree metadata"), process.Kill())
		}
		children[parent] = append(children[parent], pid)
	}
	var terminate func(int) error
	terminate = func(pid int) error {
		var failures []error
		for _, child := range children[pid] {
			if err := terminate(child); err != nil {
				failures = append(failures, err)
			}
		}
		owned, err := os.FindProcess(pid)
		if err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := owned.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			failures = append(failures, err)
		}
		if err := owned.Release(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			failures = append(failures, err)
		}
		return errors.Join(failures...)
	}
	return terminate(process.Pid)
}
