package qualify

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRepairA11MigrationInventory(t *testing.T) {
	root := candidateFixture(t)
	copyLegacyFixture(t, root)
	proposed := proposedFixture(t)
	candidate, err := Assemble(root, proposed)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = candidate.Close() }()
	if _, err := candidate.MigrationJSON("fixture", strings.Repeat("1", 64), strings.Repeat("2", 64), nil); err == nil {
		t.Fatal("FAIL unqualified migration inventory accepted")
	}
	reports, err := candidate.Compile(func([]string, string, ...string) (int, string) { return 0, "" }, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body, err := candidate.MigrationJSON("fixture", strings.Repeat("1", 64), strings.Repeat("2", 64), reports)
	if err != nil {
		t.Fatal(err)
	}
	var record Migration
	if err := json.Unmarshal(body, &record); err != nil {
		t.Fatal(err)
	}
	if len(record.Previous) != 4 || len(record.Proposed) != 2 || record.Previous["generated-manifest.json"] != legacyManifestSHA256 || record.Proposed["owned.go"] != contentSHA256(proposed["owned.go"]) || record.Proposed["generated-manifest.json"] != contentSHA256(proposed["generated-manifest.json"]) {
		t.Errorf("FAIL migration inventory identities: %s", body)
	}
	candidateWrite(t, root, "handwritten.go", []byte("intervening edit"))
	if _, err := candidate.MigrationJSON("fixture", strings.Repeat("1", 64), strings.Repeat("2", 64), reports); err == nil {
		t.Error("FAIL migration accepted changed source")
	}
}
