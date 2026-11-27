"""Qualify evidence retention without repeating product compilation."""

import json
from pathlib import Path
import tempfile
import unittest

from r2_evidence import migration_evidence


class TestMigrationEvidence(unittest.TestCase):
    def test_valid_and_broken_logs(self):
        with tempfile.TemporaryDirectory() as directory:
            log = Path(directory) / "stdout"
            record = {"version": 1, "proposed-sha256": {"owned.go": "a" * 64}}
            event = {"Action": "output", "Package": "github.com/maccavelli/go-mcp-lib/cmd/mcpgen",
                     "Test": "TestRepairA11ActualProduct",
                     "Output": "    fixture.go:1: R2_MIGRATION " + json.dumps(record) + "\n"}
            row = json.dumps(event) + "\n"
            terminal = json.dumps(dict(event, Action="pass", Output="")) + "\n"
            output = "test-logs " + json.dumps({"module": "generator", "directory": directory})
            log.write_text(row + terminal)
            self.assertEqual(migration_evidence(output), record)
            body = event["Output"]
            log.write_text("".join(json.dumps(dict(event, Output=body[i:i+13])) + "\n"
                                   for i in range(0, len(body), 13)) + terminal)
            self.assertEqual(migration_evidence(output), record)
            for name, body in (("missing", ""), ("duplicate", row + row),
                               ("malformed", "{broken\n"), ("unfinished", row)):
                with self.subTest(name=name):
                    log.write_text(body)
                    with self.assertRaises(ValueError):
                        migration_evidence(output)
            with self.assertRaises(ValueError):
                migration_evidence("")
            log.write_text(row + terminal)
            with self.assertRaises(ValueError):
                migration_evidence(output + "\n" + output)


if __name__ == "__main__":
    unittest.main(verbosity=2)
