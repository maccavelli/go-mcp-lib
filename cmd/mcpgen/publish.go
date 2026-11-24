package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Publication journaling: `generate` stages candidate bytes on the same
// filesystem as the destination, records original owned bytes and hashes,
// acquires an exclusive lock, publishes in deterministic order, removes
// only verified stale manifest-owned outputs, writes the manifest last,
// and leaves a journal an interrupted run can be recovered from.

// journal is the recovery record written before any replacement.
type journal struct {
	Version  int           `json:"version"`
	Repo     string        `json:"repo"`
	Dest     string        `json:"dest"`
	Created  string        `json:"created"`
	Owner    string        `json:"owner"`
	Files    []journalFile `json:"files"`
	Removing []string      `json:"removing,omitempty"`
	State    string        `json:"state"` // staged | published | complete
}

type journalFile struct {
	Name      string `json:"name"`
	Original  string `json:"original-sha256,omitempty"` // empty when the file is new
	Candidate string `json:"candidate-sha256"`
}

const journalName = ".mcpgen-journal.json"

// publish executes the checked emission: lock, stage, journal, replace in
// deterministic order, remove verified stale outputs, manifest last.
// An interrupted report (missing manifest-owned files, no conflicting
// edits) is completable; genuine drift is refused by the caller.
func publish(repo string, report *checkReport) (*pubReport, error) {
	if !report.ok() && !report.interrupted() && !report.firstGeneration() {
		return nil, fmt.Errorf("generate: refusing to publish over drift: %v", report.DriftFiles)
	}
	dest := filepath.Join(repo, "protocol")
	scratch, err := os.MkdirTemp("", "mcpgen-publish-*")
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	sdkDir, err := pinnedSDKDir()
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	fresh, err := renderProtocol(repo, sdkDir, scratch)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	// Lock: one active generator per repository.
	lock, err := acquireLock(dest)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lock.release() }()
	// Journal the originals before touching anything.
	j := &journal{
		Version: 1,
		Repo:    repo,
		Dest:    dest,
		Created: time.Now().UTC().Format(time.RFC3339),
		Owner:   lock.holder(),
		State:   "staged",
	}
	names := make([]string, 0, len(fresh))
	for name := range fresh {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		candidateSum := hashBytes(fresh[name])
		original := ""
		if data, err := os.ReadFile(filepath.Join(dest, name)); err == nil {
			original = hashBytes(data)
		}
		j.Files = append(j.Files, journalFile{Name: name, Original: original, Candidate: candidateSum})
	}
	if err := writeJournal(dest, j); err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	// Replace in deterministic order. Unchanged bytes are left untouched
	// so modification times are preserved across regenerations.
	pub := &pubReport{Journal: journalName}
	for _, name := range names {
		destPath := filepath.Join(dest, name)
		if data, err := os.ReadFile(destPath); err == nil && hashBytes(data) == hashBytes(fresh[name]) {
			continue
		}
		if err := writeSynced(destPath, fresh[name]); err != nil {
			j.State = "staged-incomplete"
			_ = writeJournal(dest, j)
			return nil, fmt.Errorf("generate: %s: %w", name, err)
		}
		pub.Published = append(pub.Published, name)
	}
	// Remove verified stale manifest-owned outputs only.
	stale, err := staleOwned(dest, names)
	if err != nil {
		return nil, err
	}
	for _, name := range stale {
		if err := os.Remove(filepath.Join(dest, name)); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("generate: stale %s: %w", name, err)
		}
		pub.Removed = append(pub.Removed, name)
	}
	j.Removing = pub.Removed
	// Manifest last, then journal complete. An unchanged manifest keeps
	// its mtime like every other unchanged file.
	manifestData := manifestBytes(report.Inventory)
	manifestPath := filepath.Join(dest, "generated-manifest.json")
	if data, err := os.ReadFile(manifestPath); err != nil || hashBytes(data) != hashBytes(manifestData) {
		if err := writeSynced(manifestPath, manifestData); err != nil {
			return nil, fmt.Errorf("generate: manifest: %w", err)
		}
	}
	j.State = "complete"
	if err := writeJournal(dest, j); err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	return pub, nil
}

func manifestBytes(files []string) []byte {
	data, err := json.MarshalIndent(emitManifest{Revision: "2026-07-28", Files: files}, "", "  ")
	if err != nil {
		return nil
	}
	return append(data, '\n')
}

// writeSynced writes a file and syncs it so recoverable state precedes
// irreversible replacement.
func writeSynced(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return f.Sync()
}

// staleOwned returns manifest-owned files absent from the new emission.
func staleOwned(dest string, current []string) ([]string, error) {
	manifest, err := loadJSONFile[emitManifest](filepath.Join(dest, "generated-manifest.json"))
	if err != nil {
		// No manifest: nothing is owned, nothing is stale.
		return nil, nil
	}
	currentSet := make(map[string]bool, len(current))
	for _, name := range current {
		currentSet[name] = true
	}
	var stale []string
	for _, name := range manifest.Files {
		if !currentSet[name] {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	return stale, nil
}

func hashBytes(data []byte) string {
	sum := newSHA256()
	sum.write(data)
	return sum.hex()
}

func writeJournal(dest string, j *journal) error {
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	return writeSynced(filepath.Join(dest, journalName), append(data, '\n'))
}

func readJournal(dest string) (*journal, error) {
	j, err := loadJSONFile[journal](filepath.Join(dest, journalName))
	if err != nil {
		return nil, err
	}
	return &j, nil
}
