package qualify

import (
	"encoding/json"
	"fmt"
	"maps"
)

// Migration is review evidence, not publication authority. It names exact old
// and proposed bytes, including both manifests, after complete compilation.
type Migration struct {
	Version       int               `json:"version"`
	Revision      string            `json:"revision"`
	SchemaSHA256  string            `json:"schema-sha256"`
	ProfileSHA256 string            `json:"profile-sha256"`
	Previous      map[string]string `json:"previous-sha256"`
	Proposed      map[string]string `json:"proposed-sha256"`
	Qualification []*Report         `json:"compile-qualification"`
}

// MigrationJSON retains the already qualified inventory without re-emitting or
// recompiling it. Native runtime qualification remains separate evidence.
func (c *Candidate) MigrationJSON(revision, schema, profile string, reports []*Report) ([]byte, error) {
	if !AllPassed(reports) {
		return nil, fmt.Errorf("qualify: migration requires complete compilation")
	}
	if err := c.VerifySource(); err != nil {
		return nil, err
	}
	if c.Proposed["generated-manifest.json"] == "" || revision == "" || len(schema) != 64 || len(profile) != 64 {
		return nil, fmt.Errorf("qualify: migration identity is incomplete")
	}
	return json.Marshal(Migration{Version: 1, Revision: revision, SchemaSHA256: schema,
		ProfileSHA256: profile, Previous: maps.Clone(c.Previous), Proposed: maps.Clone(c.Proposed), Qualification: reports})
}
