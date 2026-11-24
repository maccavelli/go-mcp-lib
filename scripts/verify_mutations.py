"""Stage-5 mutation plants.

Each mutation breaks one invariant in a scratch copy of the tree; the
real test suite must fail against the mutated copy. Mutations live in a
temp directory — the repository is never touched.
"""

import os
import shutil
import subprocess
import sys
import tempfile

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def run_tests(tree):
    env = dict(os.environ)
    env.update({
        "GOWORK": "off", "GOTOOLCHAIN": "local",
        "GOPROXY": "off", "GOSUMDB": "off",
    })
    return subprocess.run(
        ["go", "test", "./codec/", "./internal/validation/"],
        cwd=tree, env=env, capture_output=True, text=True,
    )


def mutate(tree, path, old, new):
    target = os.path.join(tree, path)
    body = open(target).read()
    if old not in body:
        print(f"FAIL plant {path}: pattern not found")
        return False
    open(target, "w").write(body.replace(old, new, 1))
    return True


def fresh_copy(base, slot):
    dst = os.path.join(base, f"tree-{slot}")
    for item in ("codec", "internal", "go.mod", "go.sum", "schema"):
        s, d = os.path.join(REPO, item), os.path.join(dst, item)
        if os.path.isdir(s):
            shutil.copytree(s, d)
        elif os.path.exists(s):
            os.makedirs(os.path.dirname(d) or dst, exist_ok=True)
            shutil.copy2(s, d)
    return dst


def main():
    failures = []
    base = tempfile.mkdtemp(prefix="mcpgen-mutations-")
    try:
        # Plant 1: remove the numeric token bound from the codec.
        tree = fresh_copy(base, "token-bound")
        if mutate(tree, "codec/codec.go",
                  'if len(token) > bounds.MaxTokenBytes {',
                  'if false && len(token) > bounds.MaxTokenBytes {'):
            r = run_tests(tree)
            caught = r.returncode != 0 and "NumericTokenBound" in r.stdout + r.stderr
            print(("pass " if caught else "FAIL ") +
                  "numeric token bound mutation caught")
            if not caught:
                failures.append("token-bound")

        # Plant 2: round numbers through float64 in exact decode.
        tree = fresh_copy(base, "rounding")
        if mutate(tree, "internal/validation/exact.go",
                  "dec.UseNumber()",
                  "_ = dec"):
            r = run_tests(tree)
            caught = r.returncode != 0
            print(("pass " if caught else "FAIL ") +
                  "float64 rounding mutation caught")
            if not caught:
                failures.append("rounding")

        # Plant 3: bypass raw duplicate-name detection.
        tree = fresh_copy(base, "duplicate-bypass")
        if mutate(tree, "codec/codec.go",
                  'if strings.Contains(err.Error(), "duplicate object member") {',
                  'if false && strings.Contains(err.Error(), "duplicate object member") {'):
            r = run_tests(tree)
            caught = r.returncode != 0 and "Duplicate" in r.stdout + r.stderr
            print(("pass " if caught else "FAIL ") +
                  "duplicate bypass mutation caught")
            if not caught:
                failures.append("duplicate-bypass")
    finally:
        shutil.rmtree(base, ignore_errors=True)

    print(f"mutation verification: {len(failures)} failure(s): {failures}")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
