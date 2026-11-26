package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/qualify"
)

// loadJSONFile decodes a JSON file into T.
func loadJSONFile[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		return zero, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return zero, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return value, nil
}

// checkReport is what `check` verified.
type checkReport struct {
	Repo            string            `json:"repo"`
	Schema          string            `json:"schema-sha256"`
	Manifest        bool              `json:"manifest-present"`
	Inventory       []string          `json:"inventory"`
	DriftFiles      []string          `json:"drift,omitempty"`
	ExtraFiles      []string          `json:"unowned,omitempty"`
	IncompleteFiles []string          `json:"interrupted,omitempty"`
	Qualification   []*qualify.Report `json:"qualification"`
}

func (r *checkReport) ok() bool {
	return qualify.AllPassed(r.Qualification) && r.Manifest && len(r.DriftFiles) == 0 && len(r.ExtraFiles) == 0 && len(r.IncompleteFiles) == 0
}

// interrupted reports a tree with missing manifest-owned files but no
// conflicting edits: an interrupted publication, completable by design.
func (r *checkReport) interrupted() bool {
	return qualify.AllPassed(r.Qualification) && len(r.IncompleteFiles) > 0 && len(r.DriftFiles) == 0 && len(r.ExtraFiles) == 0
}

// firstGeneration reports a destination with no prior emission: every
// file is new and publication is the only sensible act.
func (r *checkReport) firstGeneration() bool {
	return qualify.AllPassed(r.Qualification) && !r.Manifest && len(r.Inventory) == 0 && len(r.DriftFiles) == 0 && len(r.ExtraFiles) == 0
}

func (r *checkReport) data() map[string]any {
	return map[string]any{
		"repo":          repoRelative(r.Repo, r.Repo),
		"schema-sha256": r.Schema,
		"manifest":      r.Manifest,
		"inventory":     len(r.Inventory),
		"drift":         r.DriftFiles,
		"unowned":       r.ExtraFiles,
		"interrupted":   r.IncompleteFiles,
		"qualification": r.Qualification,
	}
}

// diffChanges is what `diff` found between two manifests.
type diffChanges struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Changed []string `json:"changed"`
}

func (c *diffChanges) data() map[string]any {
	return map[string]any{
		"from":    c.From,
		"to":      c.To,
		"added":   c.Added,
		"removed": c.Removed,
		"changed": c.Changed,
	}
}

// pubReport is what `generate` published.
type pubReport struct {
	Published []string `json:"published"`
	Removed   []string `json:"removed"`
	Journal   string   `json:"journal"`
}

func (p *pubReport) data() map[string]any {
	return map[string]any{
		"published": len(p.Published),
		"removed":   len(p.Removed),
		"journal":   p.Journal,
	}
}
