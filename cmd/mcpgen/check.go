package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

// checkTree renders the protocol package into a scratch directory and
// compares bytes and inventory against the destination without changing
// it. It requires local prepared inputs; the network is never touched.
func checkTree(repo string) (*checkReport, error) {
	schemaPath := filepath.Join(repo, "schema", "inputs", "2026-07-28", "schema.json")
	dest := filepath.Join(repo, "protocol")
	if _, err := os.Stat(schemaPath); err != nil {
		return nil, fmt.Errorf("check: pinned schema missing at %s", repoRelative(repo, schemaPath))
	}
	if _, err := os.Stat(dest); err != nil {
		if os.IsNotExist(err) {
			// First generation: the destination does not exist yet. An
			// empty report with no inventory is the honest state; every
			// emission file will be a new publication.
			data, err := os.ReadFile(schemaPath)
			if err != nil {
				return nil, fmt.Errorf("check: %w", err)
			}
			sum := sha256.Sum256(data)
			return &checkReport{
				Repo:      repo,
				Schema:    hex.EncodeToString(sum[:]),
				Manifest:  false,
				Inventory: nil,
			}, nil
		}
		return nil, fmt.Errorf("check: %w", err)
	}
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	sum := sha256.Sum256(data)
	report := &checkReport{
		Repo:      repo,
		Schema:    hex.EncodeToString(sum[:]),
		Manifest:  false,
		Inventory: nil,
	}
	destManifest, err := loadJSONFile[emitManifest](filepath.Join(dest, "generated-manifest.json"))
	if err != nil {
		report.DriftFiles = append(report.DriftFiles, "generated-manifest.json: missing or unreadable")
		return report, nil
	}
	report.Manifest = true
	report.Inventory = destManifest.Files
	// Render fresh into scratch.
	scratch, err := os.MkdirTemp("", "mcpgen-check-*")
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	sdkDir, err := pinnedSDKDir()
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	fresh, err := renderProtocol(repo, sdkDir, scratch)
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
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
		freshSum := sha256.Sum256(freshBytes)
		destSum := sha256.Sum256(destBytes)
		if freshSum != destSum {
			report.DriftFiles = append(report.DriftFiles, name+": bytes differ from deterministic emission")
		}
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	owned := make(map[string]bool, len(freshNames)+1)
	for _, name := range freshNames {
		owned[name] = true
	}
	owned["generated-manifest.json"] = true
	for _, entry := range entries {
		if !entry.IsDir() && !owned[entry.Name()] && filepath.Ext(entry.Name()) == ".go" {
			report.ExtraFiles = append(report.ExtraFiles, entry.Name()+": unowned .go file in protocol/")
		}
	}
	sort.Strings(report.DriftFiles)
	sort.Strings(report.ExtraFiles)
	return report, nil
}

// renderProtocol renders the deterministic emission into dir and returns
// the file map.
func renderProtocol(repo, sdkDir, dir string) (map[string][]byte, error) {
	schemaPath := filepath.Join(repo, "schema", "inputs", "2026-07-28", "schema.json")
	manifest, err := SchemaToProtocol(schemaPath, sdkDir, "2026-07-28", dir)
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
func pinnedSDKDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "go", "pkg", "mod", "github.com", "modelcontextprotocol", "go-sdk@v1.8.0", "mcp")
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("pinned SDK source not present in the local module cache")
	}
	return dir, nil
}

// computeDispositions recomputes the disposition table for inspect.
func computeDispositions(repo string) (map[string]string, error) {
	schemaPath := filepath.Join(repo, "schema", "inputs", "2026-07-28", "schema.json")
	doc, err := loadSchema(schemaPath)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(doc.Defs))
	for name := range doc.Defs {
		names = append(names, name)
	}
	sort.Strings(names)
	sdkDir, err := pinnedSDKDir()
	if err != nil {
		return nil, err
	}
	result, err := sdkmap.Build("2026-07-28", sdkDir, "v1.8.0", names)
	if err != nil {
		return nil, err
	}
	return result.Dispositions, nil
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
