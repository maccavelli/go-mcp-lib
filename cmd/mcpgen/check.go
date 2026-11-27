package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/publish"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/qualify"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

// checkTree renders the protocol package into a scratch directory and
// compares bytes and inventory against the destination without changing
// it. It requires local prepared inputs; the network is never touched.
func checkTree(repo string) (*checkReport, error) {
	return checkTreeNamed(repo, profile.DefaultFile)
}

func checkTreeNamed(repo, manifest string) (*checkReport, error) {
	return checkTreeContext(context.Background(), repo, manifest)
}

func checkTreeContext(ctx context.Context, repo, manifest string) (*checkReport, error) {
	inputs, err := profile.LoadNamed(repo, profile.Revision, manifest)
	if err != nil {
		return nil, err
	}
	journal, err := publish.Inspect(repo)
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	if journal != nil && journal.State != "complete" && journal.State != "restored" {
		return nil, fmt.Errorf("check: interrupted publication requires explicit recovery")
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
	compilation, err := candidate.Compile(qualify.OSRunnerContext(ctx), cache)
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	report := &checkReport{Repo: repo, Schema: profile.SchemaSHA256, Qualification: compilation,
		files: fresh, previous: maps.Clone(candidate.Previous), revalidate: candidate.VerifySource}
	if !qualify.AllPassed(compilation) {
		return report, nil
	}
	report.Migration, err = candidate.MigrationJSON(profile.Revision, profile.SchemaSHA256, profile.ProfileSHA256, compilation)
	if err != nil {
		return nil, err
	}
	report.Manifest = len(candidate.Previous) > 0
	report.Legacy = candidate.Previous[ownership.ManifestName] == ownership.LegacyManifestSHA256
	report.Inventory = ownership.Sorted(candidate.Proposed)
	for _, name := range report.Inventory {
		if candidate.Previous[name] != candidate.Proposed[name] {
			report.DriftFiles = append(report.DriftFiles, name)
		}
	}
	for _, name := range ownership.Sorted(candidate.Previous) {
		if candidate.Proposed[name] == "" {
			report.DriftFiles = append(report.DriftFiles, name+": stale owned output")
		}
	}
	sort.Strings(report.DriftFiles)
	return report, nil
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
