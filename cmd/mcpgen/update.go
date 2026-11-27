package main

import "github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/acquire"

func runUpdate(repo, source, revision, sum, candidate string, out *printer) error {
	result, err := acquire.Candidate(repo, candidate, acquire.Options{URL: source, Commit: revision, Revision: revision, SHA256: sum, Context: out.ctx})
	if err != nil {
		return err
	}
	return out.emit("update", "candidate-ready", map[string]any{"sha256": result.SHA256, "bytes": result.Bytes, "promoted": false, "commit": revision})
}
