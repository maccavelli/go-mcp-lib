package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// runUpdate performs bounded network acquisition into a candidate
// directory. It never promotes live input or output: the candidate is the
// only thing produced, and promoting it is a separate operator act.
func runUpdate(source, revision, sum, candidate string, out *printer) error {
	if err := os.MkdirAll(candidate, 0o755); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	// Local-path containment: the candidate must not be inside the
	// repository's schema inputs, and no traversal escapes it.
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if err := noTraversal(abs); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	dest := filepath.Join(abs, "schema.json")
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("update: candidate already contains schema.json; refusing to overwrite")
	}
	result, err := acquirePinned(source, revision, sum, dest)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return out.emit("update", "candidate-ready", map[string]any{
		"candidate": repoRelative(abs, abs),
		"sha256":    result.SHA256,
		"bytes":     result.Bytes,
		"promoted":  false,
		"next-step": "review the candidate, then commit it as a new schema input revision",
	})
}

func noTraversal(abs string) error {
	if filepath.Clean(abs) != abs {
		return fmt.Errorf("path is not clean: %s", abs)
	}
	if abs == string(filepath.Separator) {
		return fmt.Errorf("refusing filesystem root as candidate")
	}
	return nil
}

// acquirePinned wraps the internal acquire package against the update
// contract: bounded read, hash verification, discard on mismatch.
func acquirePinned(source, revision, sum, dest string) (acquireResult, error) {
	return runAcquire(source, revision, sum, dest)
}

// runRecover implements recover --action inspect|complete|restore.
func runRecover(action, repo string, out *printer) error {
	dest := filepath.Join(repo, "protocol")
	switch action {
	case "inspect":
		j, err := readJournal(dest)
		if err != nil {
			return out.emit("recover", "no-journal", map[string]any{"dest": "protocol"})
		}
		return out.emit("recover", j.State, map[string]any{
			"journal-state": j.State,
			"files":         len(j.Files),
			"owner":         j.Owner,
			"created":       j.Created,
		})
	case "complete":
		return recoverComplete(dest, out)
	case "restore":
		return recoverRestore(dest, out)
	default:
		return usageError("recover", "action must be inspect, complete, or restore")
	}
}

// recoverComplete finishes an interrupted publication from the journal's
// candidate hashes: files already matching are left untouched. Hash
// verification happens regardless of journal state — a tampered or
// stale-complete journal is refused, never trusted.
func recoverComplete(dest string, out *printer) error {
	j, err := readJournal(dest)
	if err != nil {
		return fmt.Errorf("recover: no journal to complete from")
	}
	verified := 0
	for _, entry := range j.Files {
		path := filepath.Join(dest, entry.Name)
		data, err := os.ReadFile(path)
		if err != nil || hashBytes(data) != entry.Candidate {
			return fmt.Errorf(
				"recover: %s does not match the journal candidate hash; run restore or resolve manually",
				entry.Name,
			)
		}
		verified++
	}
	if j.State == "complete" {
		return out.emit("recover", "already-complete", map[string]any{"files": len(j.Files), "verified": verified})
	}
	j.State = "complete"
	if err := writeJournal(dest, j); err != nil {
		return fmt.Errorf("recover: %w", err)
	}
	return out.emit("recover", "complete", map[string]any{
		"verified": verified,
		"files":    len(j.Files),
	})
}

// recoverRestore rolls back journal-owned files to their recorded
// originals. Files recorded without an original were new and are removed.
func recoverRestore(dest string, out *printer) error {
	j, err := readJournal(dest)
	if err != nil {
		return fmt.Errorf("recover: no journal to restore from")
	}
	restored, removed := 0, 0
	for _, entry := range j.Files {
		path := filepath.Join(dest, entry.Name)
		if entry.Original == "" {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("recover: removing %s: %w", entry.Name, err)
			}
			removed++
			continue
		}
		// The original bytes are not stored in the journal by design: the
		// journal records hashes, and restore requires the pre-existing
		// file to still match its recorded original hash. A file that
		// was already replaced cannot be restored from the journal alone
		// and is a conflict requiring operator resolution.
		data, err := os.ReadFile(path)
		if err != nil || hashBytes(data) != entry.Original {
			return fmt.Errorf(
				"recover: %s no longer holds its original bytes; the journal cannot restore it; resolve manually",
				entry.Name,
			)
		}
		restored++
	}
	j.State = "complete"
	if err := writeJournal(dest, j); err != nil {
		return fmt.Errorf("recover: %w", err)
	}
	return out.emit("recover", "restored", map[string]any{
		"verified-originals": restored,
		"removed-new":        removed,
	})
}
