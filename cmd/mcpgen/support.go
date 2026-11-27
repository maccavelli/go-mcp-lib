package main

import (
	"encoding/json"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/qualify"
)

// checkReport is what `check` verified.
type checkReport struct {
	files           map[string][]byte
	previous        map[string]string
	revalidate      func() error
	Legacy          bool              `json:"legacy-bootstrap-required"`
	Repo            string            `json:"repo"`
	Schema          string            `json:"schema-sha256"`
	Manifest        bool              `json:"manifest-present"`
	Inventory       []string          `json:"inventory"`
	DriftFiles      []string          `json:"drift,omitempty"`
	ExtraFiles      []string          `json:"unowned,omitempty"`
	IncompleteFiles []string          `json:"interrupted,omitempty"`
	Qualification   []*qualify.Report `json:"qualification"`
	Migration       json.RawMessage   `json:"migration,omitempty"`
}

func (r *checkReport) ok() bool {
	return qualify.AllPassed(r.Qualification) && r.Manifest && len(r.DriftFiles) == 0 && len(r.ExtraFiles) == 0 && len(r.IncompleteFiles) == 0
}

func (r *checkReport) data() map[string]any {
	return map[string]any{
		"repo":                      repoRelative(r.Repo, r.Repo),
		"schema-sha256":             r.Schema,
		"manifest":                  r.Manifest,
		"legacy-bootstrap-required": r.Legacy,
		"inventory":                 len(r.Inventory),
		"drift":                     r.DriftFiles,
		"unowned":                   r.ExtraFiles,
		"interrupted":               r.IncompleteFiles,
		"qualification":             r.Qualification,
		"migration":                 r.Migration,
	}
}
