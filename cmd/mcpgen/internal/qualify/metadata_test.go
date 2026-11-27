package qualify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepairA12OperationalMetadata(t *testing.T) {
	root := candidateFixture(t)
	candidate, err := Assemble(root, proposedFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = candidate.Close() }()
	for _, name := range []string{"protocol/.mcpgen-lock", "protocol/.mcpgen-state/token/blob"} {
		candidateWrite(t, root, name, []byte("operational state"))
	}
	if err := candidate.VerifySource(); err != nil {
		t.Errorf("FAIL owned operational metadata invalidated source: %v", err)
	}
	candidateWrite(t, root, "protocol/manual.go", []byte("package protocol"))
	if err := candidate.VerifySource(); err == nil {
		t.Error("FAIL metadata exclusion hid handwritten source edit")
	}
	if _, err := os.Stat(filepath.Join(candidate.Directory(), "protocol/.mcpgen-lock")); !os.IsNotExist(err) {
		t.Error("FAIL operational metadata copied into candidate")
	}
}
