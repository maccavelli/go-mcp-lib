#!/usr/bin/env python3
"""Exercise alias/source/target controls and their deliberately broken copies."""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    relative = "internal/sdkmap/audit.go"
    failures = 0
    for broken in (False, True):
        with tempfile.TemporaryDirectory(prefix="mcp-alias-control-") as temporary:
            module = Path(temporary) / "generator"
            shutil.copytree(root / "cmd/mcpgen", module,
                            ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
            if broken:
                source = module / relative
                text = source.read_text()
                start = "func aliasViolation(root types.Type, deprecated map[string]bool) string {"
                end = "\nfunc sortedReasons("
                if text.count(start) != 1 or text.count(end) != 1:
                    failures += report("alias.plant-shape", ["alias-audit anchors changed"])
                    continue
                before, body = text.split(start)
                _, after = body.split(end)
                source.write_text(before + start + '\n\treturn ""\n}\n' + end + after)
            result = run(["go", "test", "-vet=off", "-count=1", "-run",
                          "^TestRepairA09AliasClosure$", "./internal/sdkmap"], module)
            output = result.stdout + result.stderr
            expected = result.returncode == 0 if not broken else (
                result.returncode != 0 and "FAIL alias closure" in output)
            failures += report("alias.bypass-plant" if broken else "alias.valid",
                               [] if expected else [output],
                               "unsafe closure accepted by plant and caught by tests" if broken
                               else "safe and forbidden closure cases passed")
    for broken in (False, True):
        with tempfile.TemporaryDirectory(prefix="mcp-sdk-inventory-") as temporary:
            context = Path(temporary)
            module = context / "cmd/mcpgen"
            shutil.copytree(root / "cmd/mcpgen", module,
                            ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
            for relative in ("go.mod", "go.sum", "tools/versions.json"):
                destination = context / relative
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(root / relative, destination)
            if broken:
                source = module / "internal/sdkmap/sdkmap.go"
                text = source.read_text()
                anchor = "var Deprecations = []string{"
                if text.count(anchor) != 1:
                    failures += report("alias.deprecation-plant-shape", ["deprecation anchor changed"])
                    continue
                source.write_text(text.replace(anchor, anchor + '\n\t"Meta",', 1))
            result = run(["go", "test", "-vet=off", "-count=1", "-run",
                          "^TestRepairA09PinnedInventory$", "./internal/sdkmap"], module)
            output = result.stdout + result.stderr
            expected = result.returncode == 0 if not broken else (
                result.returncode != 0 and "Meta has no source deprecation marker" in output)
            failures += report("alias.false-deprecation-plant" if broken else "alias.inventory-control",
                               [] if expected else [output],
                               "false frozen deprecation caught" if broken else "source facts and ownership policy passed")
    for broken in (False, True):
        with tempfile.TemporaryDirectory(prefix="mcp-sdk-targets-") as temporary:
            module = Path(temporary) / "generator"
            shutil.copytree(root / "cmd/mcpgen", module,
                            ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
            if broken:
                source = module / "internal/sdkmap/audit.go"
                text = source.read_text()
                anchor = "func includeTarget(result *auditResult, target string, pkg *types.Package, deprecated map[string]bool) {"
                if text.count(anchor) != 1:
                    failures += report("alias.target-plant-shape", ["target audit anchor changed"])
                    continue
                source.write_text(text.replace(anchor, anchor + '\n\tif target == "windows/amd64" { return }', 1))
            result = run(["go", "test", "-vet=off", "-count=1", "-run",
                          "^TestRepairA09TargetExports$", "./internal/sdkmap"], module)
            output = result.stdout + result.stderr
            expected = result.returncode == 0 if not broken else (
                result.returncode != 0 and "FAIL target-dependent forbidden alias accepted" in output
                and "FAIL target-dependent non-type accepted" in output)
            failures += report("alias.target-omission-plant" if broken else "alias.target-control",
                               [] if expected else [output],
                               "omitted Windows audit caught" if broken else "three-target export-data controls passed")
    with tempfile.TemporaryDirectory(prefix="mcp-sdk-api-plant-") as temporary:
        module = Path(temporary) / "generator"
        shutil.copytree(root / "cmd/mcpgen", module,
                        ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
        source = module / "internal/sdkmap/api.go"
        text = source.read_text()
        anchor = "PointerMethods: methods(types.NewPointer(value))"
        if text.count(anchor) != 1:
            failures += report("alias.API-plant-shape", ["compiler API anchor changed"])
        else:
            source.write_text(text.replace(anchor, "PointerMethods: methods(value)", 1))
            result = run(["go", "test", "-vet=off", "-count=1", "-run",
                          "^TestRepairA09CompilerAPI$", "./internal/sdkmap"], module)
            output = result.stdout + result.stderr
            caught = result.returncode != 0 and "FAIL compiler API method sets" in output
            failures += report("alias.API-pointer-method-plant", [] if caught else [output],
                               "missing pointer method caught")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
