#!/usr/bin/env python3
"""Non-mutating pre-add checks through the canonical runner."""

import sys

from gate import main

if __name__ == "__main__":
    raise SystemExit(main(["pre-add", *sys.argv[1:]]))
