package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/qualify"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

// cmdInspect implements `inspect`: provenance, dispositions, mappings,
// unsupported features.
func cmdInspect(args []string, out *printer) error {
	fs := flagSet("inspect")
	repo := fs.String("repo", ".", "repository root")
	manifest := fs.String("manifest", profile.DefaultFile, "verified generation profile, relative to repo")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return errHelp
		}
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if err := out.selectFormat(*format); err != nil {
		return err
	}
	inputs, err := profile.LoadNamed(*repo, profile.Revision, *manifest)
	if err != nil {
		return fmt.Errorf("inspect: %w", err)
	}
	sdkDir, err := pinnedSDKDir(*repo)
	if err != nil {
		return err
	}
	scratch, err := os.MkdirTemp("", "mcpgen-inspect-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	var mapping *sdkmap.Result
	_, err = schemaToProtocolAudit(filepath.Join(*repo, filepath.FromSlash(profile.SchemaFile)), sdkDir, profile.Revision, scratch, inputs, func(actual *sdkmap.Result) { mapping = actual })
	if err != nil {
		return err
	}
	counts := dispositionCounts(mapping.Dispositions)
	data := map[string]any{
		"revision":           profile.Revision,
		"source-commit":      profile.SourceCommit,
		"schema-sha256":      profile.SchemaSHA256,
		"definitions":        len(mapping.Dispositions),
		"definitions-counts": fmt.Sprintf("alias=%d owned=%d owned-codec=%d", counts["alias"], counts["owned"], counts["owned-codec"]),
		"deprecated-surface": len(mapping.Deprecations),
		"qualification":      "inspection only; use check for complete candidate compilation",
		"mapping-revision":   profile.Revision,
		"sdk-audit":          mapping,
		"profile-sha256":     profile.ProfileSHA256,
		"generation-profile": *manifest,
	}
	for _, name := range []string{"api.json", "coverage.json", "bindings.json"} {
		body, err := os.ReadFile(filepath.Join(scratch, name))
		if err != nil {
			return err
		}
		data[name] = json.RawMessage(body)
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
	manifest := fs.String("manifest", profile.DefaultFile, "verified generation profile, relative to repo")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return errHelp
		}
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if err := out.selectFormat(*format); err != nil {
		return err
	}
	report, err := checkTreeContext(out.ctx, *repo, *manifest)
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
	manifest := fs.String("manifest", profile.DefaultFile, "verified generation profile, relative to repo")
	bootstrap := fs.Bool("bootstrap-legacy", false, "verify frozen Git evidence before migrating legacy output")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return errHelp
		}
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if err := out.selectFormat(*format); err != nil {
		return err
	}
	report, err := checkTreeContext(out.ctx, *repo, *manifest)
	if err != nil {
		return err
	}
	if !qualify.AllPassed(report.Qualification) {
		data := report.data()
		data["error"] = "generate: complete candidate qualification required"
		if err := out.emit("generate", "failed", data); err != nil {
			return err
		}
		return fmt.Errorf("generate: complete candidate qualification required")
	}
	receipt, err := publishChecked(out.ctx, *repo, report, *bootstrap, out.publicationHook)
	if err != nil {
		return err
	}
	return out.emit("generate", receipt.State, receipt)
}

// cmdDiff implements `diff --from M --to M`.
func cmdDiff(args []string, out *printer) error {
	fs := flagSet("diff")
	from := fs.String("from", "", "source manifest")
	to := fs.String("to", "", "target manifest")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return errHelp
		}
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if *from == "" || *to == "" {
		return usageError("diff", "--from and --to are required")
	}
	if err := out.selectFormat(*format); err != nil {
		return err
	}
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
	candidate := fs.String("candidate", "", "new candidate directory")
	repo := fs.String("repo", ".", "repository root for disjoint candidate validation")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return errHelp
		}
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if *source == "" || *revision == "" || *sum == "" || *candidate == "" {
		return usageError("update", "--source, --revision, --sha256, and --candidate are all required")
	}
	if err := out.selectFormat(*format); err != nil {
		return err
	}
	return runUpdate(*repo, *source, *revision, *sum, *candidate, out)
}

// cmdRecover implements journal-based recovery.
func cmdRecover(args []string, out *printer) error {
	fs := flagSet("recover")
	action := fs.String("action", "", "inspect|complete|restore")
	repo := fs.String("repo", ".", "repository root")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return errHelp
		}
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if *action == "" {
		return usageError("recover", "--action inspect|complete|restore is required")
	}
	if err := out.selectFormat(*format); err != nil {
		return err
	}
	return runRecover(*action, *repo, out)
}

// cmdDiagnostic reports generator and environment diagnostics.
func cmdDiagnostic(args []string, out *printer) error {
	fs := flagSet("diagnostic")
	repo := fs.String("repo", ".", "repository root")
	format := fs.String("format", "text", "output format")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return errHelp
		}
		return errUsage
	}
	if err := requireNoPositional(fs); err != nil {
		return err
	}
	if err := out.selectFormat(*format); err != nil {
		return err
	}
	data := map[string]any{
		"repo":            repoRelative(*repo, *repo),
		"schema-loader":   "strict, exact-number tokens",
		"resolution":      "local-only",
		"lint-exclusions": "none",
	}
	return out.emit("diagnostic", "ok", data)
}

func statusOf(r *checkReport) string {
	if !qualify.AllPassed(r.Qualification) {
		return "unqualified"
	}
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
