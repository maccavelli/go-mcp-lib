#!/usr/bin/env python3
"""Prove complete-candidate controls reject broken scratch implementations."""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    cases = (
        ("retained-inventory", "candidate_runtime_test.go", "^TestRepairA11RetainedMigration$/^changed$",
         (("if !bytes.Equal(compact.Bytes(), migration) {", "if !bytes.Equal(compact.Bytes(), migration) && false {"),),
         "FAIL retained inventory guard"),
        ("migration-inventory", "migration.go", "^TestRepairA11MigrationInventory$",
         (("Proposed: maps.Clone(c.Proposed)", "Proposed: maps.Clone(c.Previous)"),),
         "FAIL migration inventory identities"),
        ("proposed-identity", "../ownership/ownership.go", "^TestRepairA11OwnershipAndContainment$/^bad-hash$",
         (("Digest(body) != sum", "Digest(body) != sum && false"),),
         "FAIL candidate ownership/containment guard accepted bad-hash"),
        ("historical-identity", "../ownership/ownership.go", "^TestRepairA11OwnershipAndContainment$/^tampered-legacy$",
         (("Digest(body) != expected[name]", "Digest(body) != expected[name] && false"),),
         "FAIL candidate ownership/containment guard accepted tampered-legacy"),
        ("unowned-output", "candidate.go", "^TestRepairA11OwnershipAndContainment$/^unowned$",
         (("if strings.EqualFold(base, proposedName) {", "if strings.EqualFold(base, proposedName) && false {"),),
         "FAIL candidate ownership/containment guard accepted unowned"),
        ("intervening-edit", "candidate.go", "^TestRepairA11CompleteSnapshot$",
         (("func (c *Candidate) VerifySource() error {", "func (c *Candidate) VerifySource() error {\nif c != nil { return nil }"),),
         "FAIL intervening source edit accepted"),
        ("target-accounting", "candidate.go", "^TestRepairA11ReportAccounting$",
         (("if target.Target != Targets[i] || !target.Package || !target.Tests {",
           "if target.Target != Targets[i] && false || !target.Package && false || !target.Tests && false {"),),
         "FAIL incomplete candidate qualification accepted"),
        ("child-cleanup", "runner.go", "^TestRepairA11BoundedRunner$",
         (("return cancelProcessTree(cmd.Process)", "return cmd.Process.Kill()"),),
         "FAIL compiler timeout left a child running"),
    )
    failures = 0
    for name, filename, test, replacements, diagnostic in cases:
        with tempfile.TemporaryDirectory(prefix="mcp-candidate-plant-") as temporary:
            module = Path(temporary) / "generator"
            shutil.copytree(root / "cmd/mcpgen", module,
                            ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
            package = "." if filename == "candidate_runtime_test.go" else "./internal/qualify"
            source = module / filename if package == "." else module / "internal/qualify" / filename
            text = source.read_text()
            valid = True
            for anchor, replacement in replacements:
                if text.count(anchor) != 1:
                    failures += report(f"candidate.{name}.plant-shape", [f"guard anchor changed in {filename}"])
                    valid = False
                    break
                text = text.replace(anchor, replacement, 1)
            if not valid:
                continue
            source.write_text(text)
            if name == "historical-identity":
                snapshot = module / "internal/qualify/candidate.go"
                snapshot_text = snapshot.read_text()
                anchor = "if identity != expected {"
                assert snapshot_text.count(anchor) == 1
                snapshot.write_text(snapshot_text.replace(anchor, "if identity != expected && false {", 1))
            result = run(["go", "test", "-vet=off", "-count=1", "-run", test,
                          package], module, timeout=600)
            output = result.stdout + result.stderr
            caught = result.returncode != 0 and diagnostic in output
            failures += report(f"candidate.{name}.plant", [] if caught else [output],
                               f"bypass caught: {diagnostic}")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
