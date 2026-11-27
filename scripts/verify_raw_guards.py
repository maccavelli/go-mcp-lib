"""Prove R4 raw guards with genuine acceptance bypasses in scratch only."""

from pathlib import Path
import shutil
import tempfile

from gate_common import report, run


def snapshot(root: Path, destination: Path) -> None:
    for name in ("go.mod", "go.sum"):
        shutil.copy2(root / name, destination / name)
    shutil.copytree(root / "codec", destination / "codec")


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    failures = 0
    cases = (("token-limit", "if len(lexeme) > bounds.MaxTokenBytes {", "if false && len(lexeme) > bounds.MaxTokenBytes {", "FAIL strict raw numeric token limit"),
             ("depth-limit", "if decoder.StackDepth() > bounds.MaxDepth {", "if false && decoder.StackDepth() > bounds.MaxDepth {", "FAIL strict raw depth"),
             ("duplicate-bypass", "jsontext.NewDecoder(bytes.NewBuffer(data))", "jsontext.NewDecoder(bytes.NewBuffer(data), jsontext.AllowDuplicateNames(true))", "FAIL strict raw accepted malformed input"))
    with tempfile.TemporaryDirectory(prefix="mcp-raw-guards-") as temporary:
        base = Path(temporary)
        control = base / "control"
        control.mkdir()
        snapshot(root, control)
        command = ["go", "test", "-vet=off", "-count=1", "-run", "^TestRepairA15RawCodec$", "./codec"]
        result = run(command, control)
        failures += report("raw.control", [] if result.returncode == 0 else [result.stdout + result.stderr])
        for name, anchor, replacement, diagnostic in cases:
            scratch = base / name
            scratch.mkdir()
            snapshot(root, scratch)
            path = scratch / "codec/codec.go"
            body = path.read_text()
            if body.count(anchor) != 1:
                failures += report("raw." + name, ["plant anchor changed"])
                continue
            path.write_text(body.replace(anchor, replacement, 1))
            result = run(command, scratch)
            output = result.stdout + result.stderr
            failures += report("raw." + name, [] if result.returncode != 0 and diagnostic in output else [output],
                               "genuine acceptance bypass caught: " + diagnostic)
        legacy = base / "legacy"
        legacy.mkdir()
        snapshot(root, legacy)
        previous = run(["git", "show", "7606ab8:codec/codec.go"], root)
        if previous.returncode:
            failures += report("raw.legacy", [previous.stderr])
        else:
            (legacy / "codec/codec.go").write_text(previous.stdout)
            (legacy / "codec/v2.go").unlink()
            (legacy / "codec/v2_test.go").unlink()
            result = run(command, legacy)
            output = result.stdout + result.stderr
            caught = result.returncode != 0 and "FAIL strict raw accepted malformed input" in output and "FAIL frozen default bounds" in output
            failures += report("raw.legacy", [] if caught else [output], "old EOF/default behavior rejected by maintained production-path tests")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
