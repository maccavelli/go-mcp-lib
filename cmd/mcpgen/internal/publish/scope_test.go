package publish

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRepairA13ChildScope(t *testing.T) {
	for _, name := range []string{"nonce", "fixed-bytes"} {
		t.Run(name, func(t *testing.T) {
			repo, _ := fixture(t)
			id, err := token()
			if err != nil {
				t.Fatal(err)
			}
			if name == "fixed-bytes" {
				if err := os.WriteFile(filepath.Join(repo, ".publication-child-token"), []byte(id), 0o600); err != nil {
					t.Fatal(err)
				}
				for file, body := range generation(t, map[string]string{"a.go": "different", "stale.go": "old stale", "z.go": "old z"}) {
					if err := os.WriteFile(filepath.Join(repo, "protocol", file), body, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			before := publicBytes(t, repo)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPublicationChild$", "-test.timeout=8s")
			child.Env = append(os.Environ(), "MCP_PUBLICATION_CHILD="+repo, "MCP_PUBLICATION_NONCE="+id, "MCP_PUBLICATION_STOP=none")
			output, err := child.CombinedOutput()
			want := "FAIL child fixture capability"
			if name == "fixed-bytes" {
				want = "FAIL child fixture does not hold independently fixed test bytes"
			}
			if err == nil || !strings.Contains(string(output), want) {
				t.Errorf("FAIL child fixture scope guard %s: %v %s", name, err, output)
			}
			if !reflect.DeepEqual(before, publicBytes(t, repo)) {
				t.Error("FAIL child scope refusal changed fixture")
			}
		})
	}
}
