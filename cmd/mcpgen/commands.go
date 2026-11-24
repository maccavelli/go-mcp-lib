package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

// cmdInspect implements `inspect`: provenance, dispositions, mappings,
// unsupported features.
func cmdInspect(args []string, out *printer) error {
	fs := flagSet("inspect")
	repo := fs.String("repo", ".", "repository root")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	out.format = *format
	manifestPath := filepath.Join(*repo, "schema", "inputs", "2026-07-28", "manifest.json")
	manifest, err := readInputManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("inspect: %w", err)
	}
	dispositionPath := filepath.Join(*repo, "generated", "dispositions.json")
	dispositions, err := readDispositions(dispositionPath)
	if err != nil {
		// Fall back to computing dispositions if the artifact is absent.
		dispositions, err = computeDispositions(*repo)
		if err != nil {
			return fmt.Errorf("inspect: %w", err)
		}
	}
	counts := dispositionCounts(dispositions)
	data := map[string]any{
		"revision":           manifest.Revision,
		"source-commit":      manifest.Source.Commit,
		"schema-sha256":      manifest.SHA256,
		"definitions":        manifest.Definitions,
		"definitions-counts": fmt.Sprintf("alias=%d owned=%d owned-codec=%d", counts["alias"], counts["owned"], counts["owned-codec"]),
		"deprecated-surface": len(sdkmap.Deprecations),
		"unsupported":        "none; unsupported assertions and vocabularies fail at load",
		"mapping-revision":   manifest.Revision,
	}
	return out.emit("inspect", "ok", data)
}

func dispositionCounts(dispositions map[string]string) map[string]int {
	counts := map[string]int{}
	for _, d := range dispositions {
		counts[d]++
	}
	return counts
}

// cmdCheck implements `check`: render and qualify in scratch, compare
// bytes and inventory without changing destinations.
func cmdCheck(args []string, out *printer) error {
	fs := flagSet("check")
	repo := fs.String("repo", ".", "repository root")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	out.format = *format
	report, err := checkTree(*repo)
	if err != nil {
		return err
	}
	if err := out.emit("check", statusOf(report), report.data()); err != nil {
		return err
	}
	if !report.ok() {
		// Drift, unowned collisions, and interrupted publications are all
		// failures for check: the operator decides whether to complete or
		// restore.
		return fmt.Errorf("check failed: drift=%v unowned=%v interrupted=%v",
			report.DriftFiles, report.ExtraFiles, report.IncompleteFiles)
	}
	return nil
}

// cmdGenerate implements `generate`: all checks, then deterministic
// publication.
func cmdGenerate(args []string, out *printer) error {
	fs := flagSet("generate")
	repo := fs.String("repo", ".", "repository root")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	out.format = *format
	report, err := checkTree(*repo)
	if err != nil {
		return err
	}
	if !report.ok() {
		if report.interrupted() || report.firstGeneration() {
			// An interrupted publication with no conflicting edits, or a
			// destination with no prior emission, is completed through
			// the journal by design, not refused.
			pub, err := publish(*repo, report)
			if err != nil {
				return err
			}
			return out.emit("generate", "completed-interrupted", pub.data())
		}
		// Drift is a failure: report and exit 1, never publish over it.
		if err := out.emit("generate", "drift", report.data()); err != nil {
			return err
		}
		return fmt.Errorf("refusing to publish over drift: %v", report.DriftFiles)
	}
	pub, err := publish(*repo, report)
	if err != nil {
		return err
	}
	return out.emit("generate", "ok", pub.data())
}

// cmdDiff implements `diff --from M --to M`.
func cmdDiff(args []string, out *printer) error {
	fs := flagSet("diff")
	from := fs.String("from", "", "source manifest")
	to := fs.String("to", "", "target manifest")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if *from == "" || *to == "" {
		return usageError("diff", "--from and --to are required")
	}
	out.format = *format
	changes, err := diffManifests(*from, *to)
	if err != nil {
		return err
	}
	status := "ok"
	if len(changes.Changed) > 0 || len(changes.Removed) > 0 || len(changes.Added) > 0 {
		status = "changed"
	}
	return out.emit("diff", status, changes.data())
}

// cmdUpdate implements bounded network acquisition into a candidate.
func cmdUpdate(args []string, out *printer) error {
	fs := flagSet("update")
	source := fs.String("source", "", "schema source URL")
	revision := fs.String("revision", "", "upstream commit")
	sum := fs.String("sha256", "", "expected digest")
	candidate := fs.String("candidate", "", "candidate directory")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if *source == "" || *revision == "" || *sum == "" || *candidate == "" {
		return usageError("update", "--source, --revision, --sha256, and --candidate are all required")
	}
	out.format = *format
	return runUpdate(*source, *revision, *sum, *candidate, out)
}

// cmdRecover implements journal-based recovery.
func cmdRecover(args []string, out *printer) error {
	fs := flagSet("recover")
	action := fs.String("action", "", "inspect|complete|restore")
	repo := fs.String("repo", ".", "repository root")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if *action == "" {
		return usageError("recover", "--action inspect|complete|restore is required")
	}
	out.format = *format
	return runRecover(*action, *repo, out)
}

// cmdDiagnostic reports generator and environment diagnostics.
func cmdDiagnostic(args []string, out *printer) error {
	fs := flagSet("diagnostic")
	repo := fs.String("repo", ".", "repository root")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	out.format = *format
	data := map[string]any{
		"repo":            repoRelative(*repo, *repo),
		"schema-loader":   "strict, exact-number tokens",
		"resolution":      "local-only",
		"lint-exclusions": "none",
	}
	return out.emit("diagnostic", "ok", data)
}

func readInputManifest(path string) (*inputManifest, error) {
	m, err := loadJSONFile[inputManifest](path)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func readDispositions(path string) (map[string]string, error) {
	return loadJSONFile[map[string]string](path)
}

func statusOf(r *checkReport) string {
	if r.ok() {
		return "ok"
	}
	return "drift"
}

func repoRelative(repo, path string) string {
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(repo, path)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return path
}
