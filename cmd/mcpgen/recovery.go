package main

import (
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/publish"
)

func runRecover(action, repo string, out *printer) error {
	switch action {
	case "inspect":
		journal, err := publish.Inspect(repo)
		if err != nil {
			return err
		}
		if journal == nil {
			return out.emit("recover", "no-journal", map[string]any{"dest": "protocol"})
		}
		return out.emit("recover", journal.State, journal)
	case "complete", "restore":
		receipt, err := publish.Recover(out.ctx, repo, action, nil)
		if err != nil {
			return err
		}
		return out.emit("recover", receipt.State, receipt)
	default:
		return usageError("recover", "action must be inspect, complete, or restore")
	}
}
