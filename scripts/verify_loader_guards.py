#!/usr/bin/env python3
"""Prove loader/model regressions fail against the frozen broken baseline."""

from pathlib import Path
import shutil
import subprocess
import tempfile

from gate_common import report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    failures = 0
    cases = (("loader", "load", "TestRepairA05StrictLoader", "FAIL strict loader accepted deliberately broken"),
             ("model", "model", "TestRepairA07Vocabulary", "FAIL unqualified or malformed keyword accepted"))
    for name, package, test, diagnostic in cases:
        relative = f"internal/{package}/{package}.go"
        # This immutable baseline is the R1 phase commit, before the R2 repair.
        baseline = subprocess.run(["git", "show", "28a9284:cmd/mcpgen/" + relative],
                                  cwd=root, capture_output=True, check=False)
        if baseline.returncode:
            failures += report(f"{name}.baseline", ["frozen baseline unavailable"])
            continue
        for broken in (False, True):
            with tempfile.TemporaryDirectory(prefix=f"mcp-{name}-control-") as temporary:
                module = Path(temporary) / "generator"
                shutil.copytree(root / "cmd/mcpgen", module,
                                ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
                if broken:
                    source = baseline.stdout
                    if package == "model":
                        # Keep all test sources. The new IR field is a compile
                        # compatibility shim only: baseline parsing still omits
                        # every R2 vocabulary fix. This makes the plant fail on
                        # malformed acceptance, not an unrelated missing field.
                        anchor = b"type Node struct {"
                        if source.count(anchor) != 1:
                            failures += report("model.baseline-shape", ["Node declaration changed"])
                            continue
                        source = source.replace(anchor, anchor + b"\nAdditional *Node", 1)
                    (module / relative).write_bytes(source)
                result = run(["go", "test", "-vet=off", "-count=1", "-run", f"^{test}$", f"./internal/{package}"], module)
                output = result.stdout + result.stderr
                expected = result.returncode == 0 if not broken else result.returncode != 0 and diagnostic in output
                failures += report(f"{name}.broken-baseline" if broken else f"{name}.valid", [] if expected else [output],
                                   "regression tests caught malformed acceptance" if broken else "all cases passed")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
