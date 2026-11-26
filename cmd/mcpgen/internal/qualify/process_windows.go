package qualify

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func cancelProcessTree(process *os.Process) error {
	if process == nil {
		return os.ErrProcessDone
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(process.Pid), "/T", "/F").CombinedOutput()
	if err != nil {
		return fmt.Errorf("qualify: command-tree cancellation failed: %w: %s", err, tail(string(output), 256))
	}
	return nil
}
