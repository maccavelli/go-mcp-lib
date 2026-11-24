package main

import (
	"fmt"
	"path/filepath"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/acquire"
)

// acquireResult mirrors the internal acquire result for command output.
type acquireResult = acquire.Result

// runAcquire fetches the pinned bytes into dest with verification.
func runAcquire(source, revision, sum, dest string) (acquireResult, error) {
	dir := filepath.Dir(dest)
	result, err := acquire.Run(acquire.Options{
		InputsDir: dir,
		Revision:  revision,
		Commit:    extractCommit(source),
		SHA256:    sum,
		URL:       source,
	})
	if err != nil {
		return acquireResult{}, err
	}
	return *result, nil
}

// extractCommit pulls the commit out of a raw.githubusercontent URL when
// present; the URL flag governs the actual fetch either way.
func extractCommit(source string) string {
	const marker = "modelcontextprotocol/"
	at := indexOf(source, marker)
	if at < 0 {
		return "unknown"
	}
	rest := source[at+len(marker):]
	slash := indexOf(rest, "/")
	if slash < 0 {
		return "unknown"
	}
	return rest[:slash]
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

var _ = fmt.Sprint
