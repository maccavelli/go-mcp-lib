#!/usr/bin/env python3
"""Qualify corpus binding assertions using scratch-only deliberate mutations."""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    relative = Path("testdata/fixtures/bounds.json")
    original = (root / "cmd/mcpgen" / relative).read_text()
    cases = [
        ("valid", None, None, None),
        ("missing-binding", '\t\t"definition": "Bounded",\n', "", "FAIL fixture binding"),
        ("missing-items-binding", '\t\t"items-definition": "Empty",\n', "", "FAIL fixture binding"),
        ("unknown-binding", '"definition": "Bounded"', '"definition": "Unknown"', "FAIL fixture binding"),
        ("wrong-binding", '"definition": "Bounded"', '"definition": "Empty"', "FAIL minimum token"),
        ("wrong-minimum", '"minimum": 0.30000000000000004', '"minimum": 0.3', "FAIL minimum token"),
        ("wrong-maximum", '"maximum": 1e308', '"maximum": 1e307', "FAIL maximum token"),
        ("wrong-zero-minimum", '"minItems": 0', '"minItems": 1', "FAIL minItems"),
        ("wrong-zero-maximum", '"maxItems": 0', '"maxItems": 1', "FAIL maxItems"),
        ("missing-zero-minimum", '\t\t\t"minItems": 0,\n', "", "FAIL minItems: assertion absent"),
        ("missing-zero-maximum", ',\n\t\t\t"maxItems": 0', "", "FAIL maxItems: assertion absent"),
    ]
    failures = 0
    for name, old, new, expected in cases:
        try:
            with tempfile.TemporaryDirectory(prefix="mcp-fixture-control-") as temporary:
                module = Path(temporary) / "generator"
                shutil.copytree(root / "cmd/mcpgen", module, ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
                fixture = module / relative
                body = original
                if old is not None:
                    if old not in body:
                        raise ValueError("fixture mutation anchor absent")
                    body = body.replace(old, new, 1)
                fixture.write_text(body)
                result = run(["go", "test", "-vet=off", "-count=1", "-run", "^TestFixtureCorpus$/^bounds[.]json$", "./internal/model"], module)
                output = result.stdout + result.stderr
                caught = result.returncode != 0 and expected is not None and expected in output
                valid = result.returncode == 0 if expected is None else caught
                failures += report(f"fixture.{name}", [] if valid else [f"unexpected exit {result.returncode}: {output}"],
                                   "valid control passed" if expected is None else f"caught {expected}")
        except (OSError, ValueError) as exc:
            failures += report(f"fixture.{name}", [str(exc)])
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
