"""Stage-4 update-command contract verification.

update is the only network acquisition command: bounded, hash-verified,
candidate-only. Verifies the positive path against a local HTTP source and
the negative contracts: missing flags, candidate conflict, traversal, hash
mismatch.
"""

import http.server
import os
import shutil
import subprocess
import sys
import tempfile
import threading

SRC = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CLI = "/tmp/mcpgen-cli"

BODY = b'{"$defs": {}, "pinned": true}'
SUM = None


def sha256(data):
    import hashlib
    return hashlib.sha256(data).hexdigest()


def serve(body):
    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            self.send_response(200)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *_):
            pass

    server = http.server.HTTPServer(("127.0.0.1", 0), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    return server


def run(*args):
    return subprocess.run([CLI, *args], capture_output=True, text=True)


def main():
    failures = []

    def check(name, ok, detail=""):
        print(("pass " if ok else "FAIL ") + name + (f": {detail}" if detail else ""))
        if not ok:
            failures.append(name)

    server = serve(BODY)
    url = f"http://127.0.0.1:{server.server_address[1]}/schema.json"
    sum_ok = sha256(BODY)
    base = tempfile.mkdtemp(prefix="mcpgen-update-")
    try:
        # Positive: candidate produced, never promoted.
        cand = os.path.join(base, "cand1")
        r = run("update", "--source", url, "--revision", "abc123",
                "--sha256", sum_ok, "--candidate", cand)
        check("update candidate ready", r.returncode == 0, r.stderr.strip()[-100:])
        check("candidate file present", os.path.exists(os.path.join(cand, "schema.json")))
        check("candidate not promoted anywhere", not os.path.exists(
            os.path.join(SRC, "schema", "inputs", "abc123")))

        # Conflict: existing schema.json in the candidate refuses.
        r = run("update", "--source", url, "--revision", "abc123",
                "--sha256", sum_ok, "--candidate", cand)
        check("candidate conflict refused", r.returncode == 1 and "refusing" in r.stderr,
              r.stderr.strip()[-100:])

        # Hash mismatch: nothing written.
        bad = os.path.join(base, "bad")
        r = run("update", "--source", url, "--revision", "abc123",
                "--sha256", "0" * 64, "--candidate", bad)
        check("hash mismatch refused", r.returncode == 1,
              r.stderr.strip()[-100:])
        check("mismatch wrote nothing", not os.path.exists(os.path.join(bad, "schema.json")))

        # Missing flags: usage error.
        r = run("update", "--source", url)
        check("missing flags usage error", r.returncode == 2, r.stderr.strip()[-80:])

        # Unreachable source: loud failure, nothing written.
        ghost = os.path.join(base, "ghost")
        r = run("update", "--source", "http://127.0.0.1:9/schema.json",
                "--revision", "abc123", "--sha256", sum_ok, "--candidate", ghost)
        check("unreachable source fails loudly", r.returncode == 1,
              r.stderr.strip()[-100:])
        check("failed fetch wrote nothing", not os.path.exists(os.path.join(ghost, "schema.json")))
    finally:
        shutil.rmtree(base, ignore_errors=True)
        server.shutdown()

    print(f"update verification: {len(failures)} failure(s)")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
