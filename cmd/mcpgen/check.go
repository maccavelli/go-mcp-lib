package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/qualify"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

// checkTree renders the protocol package into a scratch directory and
// compares bytes and inventory against the destination without changing
// it. It requires local prepared inputs; the network is never touched.
func checkTree(repo string) (*checkReport, error) {
	inputs, err := profile.Load(repo, profile.Revision)
	if err != nil {
		return nil, err
	}
	// First generation and migrations take the same complete candidate path.
	// The candidate owns all writes; no destination is a qualification sandbox.
	scratch, err := os.MkdirTemp("", "mcpgen-check-*")
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	sdkDir, err := pinnedSDKDir(repo)
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	fresh, err := renderProtocolInputs(repo, sdkDir, scratch, inputs)
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(scratch, "generated-manifest.json"))
	if err != nil {
		return nil, err
	}
	fresh["generated-manifest.json"] = manifestBytes
	candidate, err := qualify.Assemble(repo, fresh)
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	defer func() { _ = candidate.Close() }()
	cache, err := qualify.BuildCache()
	if err != nil {
		return nil, err
	}
	compilation, err := candidate.Compile(qualify.OSRunner(), cache)
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	report := &checkReport{Repo: repo, Schema: profile.SchemaSHA256, Qualification: compilation}
	if !qualify.AllPassed(compilation) {
		return report, nil
	}
	dest := filepath.Join(repo, "protocol")
	if _, err := os.Stat(dest); err != nil {
		if os.IsNotExist(err) {
			// First generation: the destination does not exist yet. An
			// empty report with no inventory is the honest state; every
			// emission file will be a new publication.
			return report, nil
		}
		return nil, fmt.Errorf("check: %w", err)
	}
	destManifest, err := loadJSONFile[emitManifest](filepath.Join(dest, "generated-manifest.json"))
	if err != nil {
		report.DriftFiles = append(report.DriftFiles, "generated-manifest.json: missing or unreadable")
		return report, nil
	}
	report.Manifest = true
	report.Inventory = destManifest.Files
	// Byte-compare every manifest-owned file; find unowned collisions.
	// A manifest-owned file that is absent from the destination is an
	// interrupted publication, not drift: it is reported separately so
	// generate can complete it through the journal rather than refuse.
	freshNames := make([]string, 0, len(fresh))
	for name := range fresh {
		freshNames = append(freshNames, name)
	}
	sort.Strings(freshNames)
	for _, name := range freshNames {
		freshBytes := fresh[name]
		destPath := filepath.Join(dest, name)
		destBytes, err := os.ReadFile(destPath)
		if err != nil {
			if os.IsNotExist(err) {
				report.IncompleteFiles = append(report.IncompleteFiles, name)
				continue
			}
			return nil, fmt.Errorf("check: %w", err)
		}
		if !bytes.Equal(freshBytes, destBytes) {
			report.DriftFiles = append(report.DriftFiles, name+": bytes differ from deterministic emission")
		}
	}
	// Handwritten source is part of the qualified candidate, not output drift.
	// Assemble already rejects collisions with newly owned output filenames.
	sort.Strings(report.DriftFiles)
	sort.Strings(report.ExtraFiles)
	return report, nil
}

// renderProtocol renders the deterministic emission into dir and returns
// the file map.
func renderProtocol(repo, sdkDir, dir string) (map[string][]byte, error) {
	return renderProtocolInputs(repo, sdkDir, dir, nil)
}

func renderProtocolInputs(repo, sdkDir, dir string, inputs *profile.Inputs) (map[string][]byte, error) {
	schemaPath := filepath.Join(repo, "schema", "inputs", "2026-07-28", "schema.json")
	manifest, err := schemaToProtocol(schemaPath, sdkDir, "2026-07-28", dir, inputs)
	if err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(manifest.Files))
	for _, name := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		files[name] = data
	}
	return files, nil
}

// pinnedSDKDir locates the pinned SDK source in the local module cache.
func pinnedSDKDir(repo string) (string, error) {
	return sdkmap.LocateSDK(repo)
}

// computeSDKMapping recomputes the audited disposition table for inspect.
func computeSDKMapping(repo string) (*sdkmap.Result, error) {
	inputs, err := profile.Load(repo, profile.Revision)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, inputs.Schema.Definitions)
	for name := range inputs.Mapping.Dispositions {
		names = append(names, name)
	}
	sort.Strings(names)
	sdkDir, err := pinnedSDKDir(repo)
	if err != nil {
		return nil, err
	}
	actual, err := sdkmap.BuildInModule(profile.Revision, sdkDir, "v1.8.0", repo, names)
	if err != nil {
		return nil, err
	}
	return inputs.ApplySDK(actual)
}

// diffManifests compares two emitted manifests.
func diffManifests(fromPath, toPath string) (*diffChanges, error) {
	from, err := loadJSONFile[emitManifest](fromPath)
	if err != nil {
		return nil, fmt.Errorf("diff: %w", err)
	}
	to, err := loadJSONFile[emitManifest](toPath)
	if err != nil {
		return nil, fmt.Errorf("diff: %w", err)
	}
	changes := &diffChanges{From: from.Revision, To: to.Revision}
	fromSet := make(map[string]bool, len(from.Files))
	for _, name := range from.Files {
		fromSet[name] = true
	}
	toSet := make(map[string]bool, len(to.Files))
	for _, name := range to.Files {
		toSet[name] = true
		if !fromSet[name] {
			changes.Added = append(changes.Added, name)
		}
	}
	for _, name := range from.Files {
		if !toSet[name] {
			changes.Removed = append(changes.Removed, name)
		}
	}
	if from.Revision != to.Revision {
		changes.Changed = append(changes.Changed, "revision: "+from.Revision+" -> "+to.Revision)
	}
	sort.Strings(changes.Added)
	sort.Strings(changes.Removed)
	sort.Strings(changes.Changed)
	return changes, nil
}

// emitManifest is the on-disk generated manifest.
type emitManifest struct {
	Revision string   `json:"revision"`
	Files    []string `json:"files"`
}
