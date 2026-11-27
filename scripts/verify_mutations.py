"""Prove canonical assertions and exact numbers with genuine scratch bypasses.

Raw duplicate/token/depth controls have one owner, verify_raw_guards.py. This
helper does not rerun them or accept unrelated compiler failures as evidence.
"""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def snapshot(root: Path, destination: Path) -> None:
    for name in ("go.mod", "go.sum"):
        shutil.copy2(root / name, destination / name)
    for name in ("codec", "internal/validation", "schema/inputs"):
        shutil.copytree(root / name, destination / name)


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    failures = 0
    cases = (
        ("required", "internal/validation/validation.go", "compiler.AddResource(canonical, owned.document)",
         "compiler.AddResource(canonical, withoutRequired(owned.document))",
         '^TestRepairA16RequiredAssertion$', "FAIL required assertion removed",
         '\nfunc withoutRequired(doc any) any { if object, ok := doc.(map[string]any); ok { delete(object, "required") }; return doc }\n'),
        ("rounding", "codec/v2.go", "projection.UseNumber()", "// deliberately round the exact dynamic projection",
         '^TestRepairA16ExactNeighbors$', "FAIL exact adjacent integer", ""),
        ("locations", "internal/validation/protocol.go", "if err := rule.check(text); err != nil {",
         "if err := rule.check(text); false && err != nil {", '^TestRepairA17Locations$', "FAIL mandated location assertion bypassed", ""),
        ("stale-cache", "internal/validation/validation.go", "key := a.cacheKey(identity, canonical, owned.body)",
         "key := identity", '^TestRepairA16ValidationProfile$/^content-bound-cache$', "FAIL changed schema reused stale compilation", ""),
        ("schema-pointer", "internal/validation/diagnostic.go", 'message = strings.Join(locations, ", ") + ": " + message',
         '_ = strings.Join(locations, ", ") // deliberately omit the composed schema pointer', '^TestRepairA16ValidationProfile$/^draft-vocabulary-and-pointer$', "FAIL unqualified profile declaration", ""),
    )
    with tempfile.TemporaryDirectory(prefix="mcp-validator-plants-") as temporary:
        base = Path(temporary)
        control = base / "control"
        control.mkdir()
        snapshot(root, control)
        selected = "^(TestRepairA16RequiredAssertion|TestRepairA16ExactNeighbors|TestRepairA16ValidationProfile|TestRepairA17Locations)$"
        command = ["go", "test", "-vet=off", "-count=1", "-run", selected, "./internal/validation"]
        result = run(command, control)
        failures += report("validator.control", [] if result.returncode == 0 else [result.stdout + result.stderr])
        for name, relative, anchor, replacement, test, diagnostic, suffix in cases:
            scratch = base / name
            scratch.mkdir()
            snapshot(root, scratch)
            path = scratch / relative
            body = path.read_text()
            if body.count(anchor) != 1:
                failures += report("validator." + name, ["plant anchor changed"])
                continue
            path.write_text(body.replace(anchor, replacement, 1) + suffix)
            result = run(["go", "test", "-vet=off", "-count=1", "-run", test, "./internal/validation"], scratch)
            output = result.stdout + result.stderr
            caught = result.returncode != 0 and diagnostic in output and "[build failed]" not in output
            failures += report("validator." + name, [] if caught else [output], "genuine bypass caught: " + diagnostic)
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
