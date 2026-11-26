#!/usr/bin/env python3
"""Prove complete-candidate controls reject broken scratch implementations."""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    cases = (
        ("proposed-identity", "candidate.go", "^TestRepairA11OwnershipAndContainment$/^bad-hash$",
         (("manifest.SHA256[name] != contentSHA256(body)", "false"),),
         "FAIL candidate ownership/containment guard accepted bad-hash"),
        ("historical-identity", "candidate.go", "^TestRepairA11OwnershipAndContainment$/^tampered-legacy$",
         (("hex.EncodeToString(hash.Sum(nil)) != expected[name]", "false"),
          ("identity != expected", "identity != expected && false")),
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
            source = module / "internal/qualify" / filename
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
            result = run(["go", "test", "-vet=off", "-count=1", "-run", test,
                          "./internal/qualify"], module, timeout=600)
            output = result.stdout + result.stderr
            caught = result.returncode != 0 and diagnostic in output
            failures += report(f"candidate.{name}.plant", [] if caught else [output],
                               f"bypass caught: {diagnostic}")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
