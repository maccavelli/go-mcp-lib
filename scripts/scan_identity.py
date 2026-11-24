"""Pre-add identity scan for stage 5.

Scans the files staged for the stage-5 commit for the local account
name and host-specific path prefixes. Describes findings; never prints
the identifiers themselves.
"""

import os
import subprocess
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Derived from the environment at run time, never embedded.
ACCOUNT = os.environ.get("USER", "")
HOME = os.environ.get("HOME", "")
HOST_MARKERS = ["/Users/", "/home/", HOME] if HOME else ["/Users/", "/home/"]
MARKERS = [m for m in [ACCOUNT, *HOST_MARKERS] if m and len(m) > 3]


def main():
    r = subprocess.run(
        ["git", "diff", "--cached", "--name-only"],
        cwd=REPO, capture_output=True, text=True,
    )
    staged = [f for f in r.stdout.splitlines() if f]
    if not staged:
        r = subprocess.run(
            ["git", "status", "--porcelain"],
            cwd=REPO, capture_output=True, text=True,
        )
        staged = [
            line[3:].strip()
            for line in r.stdout.splitlines()
            if line and not line.startswith("??")
        ]
    hits = []
    for path in staged:
        full = os.path.join(REPO, path)
        if not os.path.isfile(full) or os.path.getsize(full) > 2_000_000:
            continue
        try:
            body = open(full, encoding="utf-8", errors="replace").read()
        except OSError:
            continue
        for marker in MARKERS:
            if marker in body:
                hits.append((path, "account name or host path prefix"))
    if hits:
        for path, what in hits:
            print(f"FOUND {what} in {path}")
        return 1
    print(f"scanned {len(staged)} tracked-change files for the local "
          "account name and host path prefixes; no hits")
    return 0


if __name__ == "__main__":
    sys.exit(main())
