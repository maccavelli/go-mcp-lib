"""Reject broken R4 migration evidence before changing a retained review."""

import copy
import json
import unittest

from qualify_r4 import reviewed_migration


class ReviewTests(unittest.TestCase):
    def test_evidence(self):
        migration = {"previous-sha256": {"api.json": "old-api", "codecs.go": "old", "generated-manifest.json": "before", "owned.go": "stable"},
                     "proposed-sha256": {"api.json": "exact-import-update", "codecs.go": "new", "generated-manifest.json": "after", "owned.go": "stable"},
                     "compile-qualification": [{"candidate": module, "passed": True,
                         "targets": [{"target": target, "package": True, "tests": True}
                                     for target in ("darwin/arm64", "linux/amd64", "windows/amd64")]}
                                              for module in (".", "cmd/mcpgen")]}
        valid = {"data": {"migration": migration, "drift": ["api.json", "codecs.go", "generated-manifest.json"], "unowned": [], "interrupted": []}}
        self.assertEqual(reviewed_migration(json.dumps(valid), 1, "exact-import-update"), migration)
        for case in ("failed-target", "omitted-module", "extra-file", "removed-file", "extra-drift", "api-drift", "unowned", "interrupted", "wrong-exit"):
            with self.subTest(case=case):
                plant = copy.deepcopy(valid)
                data, proposed = plant["data"], plant["data"]["migration"]
                code = 1
                if case == "failed-target":
                    proposed["compile-qualification"][0]["targets"][0]["tests"] = False
                elif case == "omitted-module":
                    proposed["compile-qualification"].pop()
                elif case == "extra-file":
                    proposed["proposed-sha256"]["owned.go"] = "changed"
                elif case == "removed-file":
                    del proposed["proposed-sha256"]["owned.go"]
                elif case == "extra-drift":
                    data["drift"].append("owned.go")
                elif case == "api-drift":
                    proposed["proposed-sha256"]["api.json"] = "changed-signature-or-other-metadata"
                elif case in {"unowned", "interrupted"}:
                    data[case] = ["fixture"]
                elif case == "wrong-exit":
                    code = 0
                with self.assertRaises(ValueError, msg="FAIL " + case):
                    reviewed_migration(json.dumps(plant), code, "exact-import-update")


if __name__ == "__main__":
    unittest.main()
