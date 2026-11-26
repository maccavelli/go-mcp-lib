#!/usr/bin/env python3
"""Prove production graph validation rejects excessive acyclic depth."""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    failures = 0
    for broken in (False, True):
        with tempfile.TemporaryDirectory(prefix="mcp-reference-control-") as temporary:
            module = Path(temporary) / "generator"
            shutil.copytree(root / "cmd/mcpgen", module,
                            ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
            if broken:
                source = module / "internal/resolve/resolve.go"
                text = source.read_text()
                start = "func (r *Registry) Validate() error {"
                end = "\n// Resolve resolves"
                if text.count(start) != 1 or text.count(end) != 1:
                    failures += report("reference.plant-shape", ["graph validation anchors changed"])
                    continue
                before, body = text.split(start)
                _, after = body.split(end)
                source.write_text(before + start + "\n\treturn nil\n}\n" + end + after)
            result = run(["go", "test", "-vet=off", "-count=1", "-run",
                          "^TestRepairA06References$/^acyclic-depth-in-validation$", "./internal/resolve"], module)
            output = result.stdout + result.stderr
            expected = result.returncode == 0 if not broken else (
                result.returncode != 0 and "FAIL over-depth graph accepted" in output)
            failures += report("reference.bypass-plant" if broken else "reference.valid",
                               [] if expected else [output],
                               "over-depth graph bypass caught" if broken else "depth rejection passed")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
