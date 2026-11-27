"""Prove publication guards with broken scratch implementations, never live output."""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    cases = (
        ("profile-alias", "profile/profile.go", "./internal/profile", "^TestRepairA14ProfileSelection$",
         "if info.Mode()&os.ModeSymlink != 0 {", "if info.Mode()&os.ModeSymlink != 0 && false {",
         "FAIL aliased profile accepted"),
        ("profile-flag", "commands.go", ".", "^TestRepairA14ProfileFlag$/^check$",
         "report, err := checkTreeContext(out.ctx, *repo, *manifest)",
         "_ = manifest; report, err := checkTreeContext(out.ctx, *repo, profile.DefaultFile)", "FAIL CLI profile flag contract"),
        ("child-scope", "publish/engine_test.go", "./internal/publish", "^TestRepairA13ChildScope$/^nonce$",
         'if err != nil || !validToken(string(nonce)) || string(nonce) != os.Getenv("MCP_PUBLICATION_NONCE") {',
         'if (err != nil || !validToken(string(nonce)) || string(nonce) != os.Getenv("MCP_PUBLICATION_NONCE")) && false {',
         "FAIL child fixture scope guard"),
        ("portable-names", "ownership/ownership.go", "./internal/ownership", "^TestRepairA12Names$",
         "func Name(name string) error {", "func Name(name string) error {\nif name != \"\" { return nil }",
         "FAIL unsafe output name accepted"),
        ("manifest-last", "publish/engine.go", "./internal/publish", "^TestRepairA13Publication$",
         "if index >= 0 {", "if index >= 0 && false {", "FAIL manifest promoted before data"),
        ("manual-edit", "publish/engine.go", "./internal/publish", "^TestRepairA13ManualAndCorrupt$/^new-manual-complete$",
         'else if sum != entry.Before && sum != entry.After && !inFlightMissing {',
         "else if sum != entry.Before && sum != entry.After && !inFlightMissing && false {", "FAIL manual/corrupt recovery accepted"),
        ("exclusive-lock", "publish/engine.go", "./internal/publish", "^TestRepairA12Containment$/^stale-lock$",
         "os.O_CREATE|os.O_EXCL|os.O_RDWR", "os.O_CREATE|os.O_RDWR", "FAIL containment/conflict guard"),
        ("open-race", "publish/engine.go", "./internal/publish", "^TestRepairA12OpenRace$",
         "!os.SameFile(e.identity, current)", "false", "FAIL open directory race guard"),
        ("bounded-fetch", "acquire/acquisition.go", "./internal/acquire", "^TestRepairA14Acquisition$/^oversized$",
         "io.LimitReader(response.Body, maxBytes+1)", "io.LimitReader(response.Body, maxBytes)", "FAIL acquisition guard accepted oversized"),
        ("semantic-diff", "diff.go", ".", "^TestRepairA14SemanticDiff$",
         "if reflect.DeepEqual(before, after) && had == has {", "if reflect.DeepEqual(before, before) {", "FAIL semantic diff omitted"),
        ("strict-format", "cli.go", ".", "^TestRepairA14CLI$/^diagnostic-format$",
         'if format != "text" && format != "json" {', 'if format != "text" && format != "json" && false {', "FAIL CLI exit diagnostic/format"),
        ("metadata-scope", "qualify/candidate.go", "./internal/qualify", "^TestRepairA12OperationalMetadata$",
         'strings.HasPrefix(name, "protocol/.mcpgen-")', 'strings.HasPrefix(name, "protocol/")', "FAIL metadata exclusion hid handwritten source edit"),
        ("installed-review", "candidate_runtime_test.go", ".", "^TestRepairA11InstalledReview$/^before$",
         "!maps.Equal(current.Previous, current.Proposed)", "false", "FAIL installed review guard before"),
        ("cli-fixture-scope", "cli_publication_test.go", ".", "^TestRepairA13CLIScope$",
         "if err != nil || decodeErr != nil || len(decoded) != 16 || string(body) != nonce {",
         "if (err != nil || decodeErr != nil || len(decoded) != 16 || string(body) != nonce) && false {", "FAIL CLI fixture nonce guard accepted"),
        ("git-output-bound", "publication.go", ".", "^TestRepairA13GitBounds$",
         "if b.body.Len()+len(body) > 8<<20 {", "if b.body.Len()+len(body) > 8<<20 && false {", "FAIL bootstrap Git output bound bypassed"),
    )
    failures = 0
    for name, relative, package, test, anchor, replacement, diagnostic in cases:
        with tempfile.TemporaryDirectory(prefix="mcp-publication-plant-") as temporary:
            module = Path(temporary) / "cmd/mcpgen"
            shutil.copytree(root / "cmd/mcpgen", module,
                            ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
            source = module / relative if package == "." else module / "internal" / relative
            body = source.read_text()
            count = body.count(anchor)
            expected = 2 if name == "profile-flag" else 1
            if count != expected:
                failures += report(f"publication.{name}.plant-shape", ["guard anchor changed"])
                continue
            source.write_text(body.replace(anchor, replacement, 1))
            if name in {"profile-alias", "cli-fixture-scope"}:
                for path in ("go.mod", "go.sum", "tools/versions.json", "schema/generation-profile.json",
                             "schema/inputs/2026-07-28/schema.json", "schema/inputs/2026-07-28/manifest.json",
                             "schema/sdk-map.json", "schema/semantics.json"):
                    target = Path(temporary) / path
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(root / path, target)
            result = run(["go", "test", "-vet=off", "-count=1", "-run", test, package], module)
            output = result.stdout + result.stderr
            caught = result.returncode != 0 and diagnostic in output
            failures += report(f"publication.{name}.plant", [] if caught else [output],
                               f"bypass caught: {diagnostic}")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
