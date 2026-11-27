"""Prove the SDK bridge's legacy wire policy with a compilable scratch bypass."""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def snapshot(root: Path, destination: Path) -> None:
    for name in ("go.mod", "go.sum"):
        shutil.copy2(root / name, destination / name)
    for name in ("codec", "internal/validation", "protocol", "mcp/internal/sdkbridge"):
        shutil.copytree(root / name, destination / name, ignore=shutil.ignore_patterns(".mcpgen-*"))


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    failures = 0
    with tempfile.TemporaryDirectory(prefix="mcp-sdk-guards-") as temporary:
        scratch = Path(temporary)
        snapshot(root, scratch)
        command = ["go", "test", "-vet=off", "-count=1", "-run", "^TestRepairA16SDKDifferentials$", "./mcp/internal/sdkbridge"]
        result = run(command, scratch)
        failures += report("sdk.control", [] if result.returncode == 0 else [result.stdout + result.stderr])
        path = scratch / "mcp/internal/sdkbridge/codec.go"
        body = path.read_text()
        anchor = "body, err := legacyjson.Marshal(source)"
        if body.count(anchor) != 1:
            return failures + report("sdk.legacy-policy", ["plant anchor changed"])
        path.write_text(body.replace(anchor, "body, err := codec.Marshal(source, codec.DefaultBounds())", 1))
        result = run(command, scratch)
        output = result.stdout + result.stderr
        caught = result.returncode != 0 and "FAIL bridge legacy collection policy" in output and "[build failed]" not in output
        failures += report("sdk.legacy-policy", [] if caught else [output], "wrong encoder options caught: FAIL bridge legacy collection policy")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
