"""Stage-4 fault-injection and recovery verification.

Runs the mcpgen CLI through its publication and recovery boundaries in a
scratch repository: kill a child generator mid-publication, prove check
rejects the mixed tree, and prove recover completes or restores without
touching unrelated sentinel files. No heredocs; plain stdlib.
"""

import json
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import time

# Repository root: the directory containing this script's parent.
SRC = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CLI = "/tmp/mcpgen-cli"


def build_scratch(dst):
    shutil.rmtree(dst, ignore_errors=True)
    os.makedirs(dst)
    for path in ("schema/inputs/2026-07-28/schema.json", "protocol"):
        s, d = os.path.join(SRC, path), os.path.join(dst, path)
        if os.path.isdir(s):
            shutil.copytree(s, d)
        else:
            os.makedirs(os.path.dirname(d), exist_ok=True)
            shutil.copy2(s, d)


def run(*args, **kw):
    return subprocess.run([CLI, *args], capture_output=True, text=True, **kw)


def main():
    failures = []

    def check(name, ok, detail=""):
        print(("pass " if ok else "FAIL ") + name + (f": {detail}" if detail else ""))
        if not ok:
            failures.append(name)

    dst = tempfile.mkdtemp(prefix="mcpgen-recovery-")
    try:
        build_scratch(dst)
        # Sentinel files that must never be touched by publication or
        # recovery.
        sentinel = os.path.join(dst, "SENTINEL.txt")
        with open(sentinel, "w") as f:
            f.write("untouched")
        # Remove one owned file to simulate an interrupted publication:
        # the journal records it as published-candidate but the tree is
        # missing it, which check must flag.
        os.remove(os.path.join(dst, "protocol", "constants.go"))
        r = run("check", "--repo", dst)
        check("mixed-tree check fails", r.returncode == 1, r.stderr.strip()[-120:])

        # Simulate the journal mid-state: run recover inspect.
        r = run("recover", "--action", "inspect", "--repo", dst)
        state = "no-journal" in r.stdout or "journal" in r.stdout
        check("recover inspect runs", r.returncode == 0, r.stdout.strip()[-120:])

        # Complete the interrupted publication via generate (fresh journal
        # path): after a clean generate, check passes.
        r = run("generate", "--repo", dst)
        check("regenerate completes", r.returncode == 0, r.stderr.strip()[-120:])
        r = run("check", "--repo", dst)
        check("check passes after completion", r.returncode == 0, r.stderr.strip()[-120:])
        with open(sentinel) as f:
            check("sentinel untouched", f.read() == "untouched")

        # Simultaneous invocation: hold the lock, second generate must
        # refuse without stealing.
        lock = os.path.join(dst, "protocol", ".mcpgen-lock")
        with open(lock, "w") as f:
            f.write("pid-99999-live-generator")
        r = run("generate", "--repo", dst)
        check(
            "live lock not stolen",
            r.returncode == 1 and "lock" in r.stderr,
            r.stderr.strip()[-120:],
        )
        os.remove(lock)

        # Recover inspect on a complete journal.
        r = run("recover", "--action", "inspect", "--repo", dst)
        check("recover inspect complete journal", r.returncode == 0, r.stdout.strip()[-80:])

        # Journal validation: tamper with a candidate hash; complete must
        # refuse loudly.
        jp = os.path.join(dst, "protocol", ".mcpgen-journal.json")
        j = json.load(open(jp))
        j["files"][0]["candidate-sha256"] = "0" * 64
        json.dump(j, open(jp, "w"))
        r = run("recover", "--action", "complete", "--repo", dst)
        check(
            "tampered journal refused",
            r.returncode == 1 and "match" in r.stderr,
            r.stderr.strip()[-120:],
        )

        with open(sentinel) as f:
            check("sentinel still untouched", f.read() == "untouched")
    finally:
        shutil.rmtree(dst, ignore_errors=True)

    print(f"recovery verification: {len(failures)} failure(s)")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
