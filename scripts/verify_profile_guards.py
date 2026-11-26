#!/usr/bin/env python3
"""Prove independent profile, mapping-policy and SDK-context guards reject plants."""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    members = ("go.mod", "go.sum", "tools/versions.json", "schema/generation-profile.json",
               "schema/inputs/2026-07-28/schema.json", "schema/inputs/2026-07-28/manifest.json",
               "schema/sdk-map.json", "schema/semantics.json")
    cases = (
        ("source-pin", ".", "^TestRepairA08ProvenanceAndMapping$/^forged-profile$",
         "if digest(body) != ProfileSHA256 {", "if false {", "FAIL profile bypass reached SDK or output"),
        ("mapping-policy", "./internal/profile", "^TestRepairA08InputPolicy$",
         "func (i *Inputs) validate() error {", "func (i *Inputs) validate() error {\nif i != nil { return nil }", "FAIL invalid reviewed input policy accepted"),
        ("SDK-context", "./internal/profile", "^TestRepairA08SDKContext$",
         "func (i *Inputs) ApplySDK(actual *sdkmap.Result) (*sdkmap.Result, error) {",
         "func (i *Inputs) ApplySDK(actual *sdkmap.Result) (*sdkmap.Result, error) {\nif actual != nil { return actual, nil }", "FAIL SDK context or unsafe mapping accepted"),
    )
    failures = 0
    for name, package, test, anchor, replacement, diagnostic in cases:
        for broken in (False, True):
            with tempfile.TemporaryDirectory(prefix="mcp-profile-control-") as temporary:
                context = Path(temporary)
                module = context / "cmd/mcpgen"
                shutil.copytree(root / "cmd/mcpgen", module,
                                ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
                for relative in members:
                    destination = context / relative
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(root / relative, destination)
                if broken:
                    source = module / "internal/profile/profile.go"
                    text = source.read_text()
                    if text.count(anchor) != 1:
                        failures += report(f"profile.{name}.plant-shape", ["profile guard anchor changed"])
                        continue
                    source.write_text(text.replace(anchor, replacement, 1))
                result = run(["go", "test", "-vet=off", "-count=1", "-run", test, package], module)
                output = result.stdout + result.stderr
                expected = result.returncode == 0 if not broken else result.returncode != 0 and diagnostic in output
                failures += report(f"profile.{name}.{'plant' if broken else 'control'}",
                                   [] if expected else [output], "bypass caught" if broken else "valid guard passed")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
