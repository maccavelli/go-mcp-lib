package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
)

type stageIndex struct {
	Version int      `json:"version"`
	Token   string   `json:"token"`
	Hashes  []string `json:"sha256"`
}

func stageDir(token string) string       { return stateName + "/" + token }
func stagePath(token, sum string) string { return stageDir(token) + "/" + sum }

func stageBody(j *Journal) ([]byte, error) {
	hashes := map[string]bool{}
	for _, entry := range j.Files {
		if entry.Before != "" {
			hashes[entry.Before] = true
		}
		if entry.After != "" {
			hashes[entry.After] = true
		}
	}
	return json.Marshal(stageIndex{1, j.Token, ownership.Sorted(hashes)})
}

func (e *engine) stage(ctx context.Context, j *Journal, blobs map[string][]byte) (failure error) {
	total := 0
	for _, body := range blobs {
		total += len(body)
	}
	if total > 64<<20 {
		return fmt.Errorf("publish: staging exceeds 64 MiB")
	}
	if err := e.check(ctx, "stage", j.Token); err != nil {
		return err
	}
	if err := e.root.MkdirAll(stateName, 0o700); err != nil {
		return err
	}
	parent, err := e.directory(stateName)
	if err != nil {
		return err
	}
	defer func() { failure = errors.Join(failure, parent.Close()) }()
	if err := parent.Mkdir(j.Token, 0o700); err != nil {
		return err
	}
	info, err := parent.Lstat(j.Token)
	if err != nil {
		return err
	}
	e.directories[stageDir(j.Token)] = info
	for _, sum := range ownership.Sorted(blobs) {
		if err := e.writeFile(ctx, stagePath(j.Token, sum), blobs[sum], 0o600); err != nil {
			return err
		}
	}
	body, err := stageBody(j)
	if err != nil {
		return err
	}
	j.StageSHA256 = ownership.Digest(body)
	if err := e.writeFile(ctx, stageDir(j.Token)+"/index.json", body, 0o600); err != nil {
		return err
	}
	for _, name := range []string{stageDir(j.Token), stateName, "."} {
		if err := e.syncDir(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func (e *engine) writeJournal(ctx context.Context, j *Journal) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	if err := e.check(ctx, "journal", JournalName); err != nil {
		return err
	}
	if err := e.replace(ctx, JournalName, journalPending, body, 0o600, func() error {
		current, err := e.current(JournalName)
		if err != nil || current != e.journalSHA {
			return fmt.Errorf("publish: journal changed during operation")
		}
		return nil
	}); err != nil {
		return err
	}
	e.journalSHA = ownership.Digest(body)
	return nil
}

func strictJSON(body []byte, out any) error {
	if _, err := load.New().LoadBytes("publication-metadata", body); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

func (e *engine) readJournal() (*Journal, error) {
	body, err := ownership.Read(e.root, JournalName)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var j Journal
	if err := strictJSON(body, &j); err != nil {
		return nil, fmt.Errorf("publish: corrupt journal: %w", err)
	}
	if j.Version != 2 || !validToken(j.Token) || len(j.Files) == 0 || len(j.Files) > 513 || len(j.Garbage) > 8 ||
		(j.State != "staged" && j.State != "applying" && j.State != "complete" && j.State != "restored") {
		return nil, fmt.Errorf("publish: invalid journal identity, state or bound")
	}
	before, after, entries := map[string]string{}, map[string]string{}, map[string]Entry{}
	folded := map[string]bool{}
	for _, entry := range j.Files {
		if err := ownership.Name(entry.Name); err != nil {
			return nil, err
		}
		if entries[entry.Name].Name != "" || folded[strings.ToLower(entry.Name)] || entry.Before == "" && entry.After == "" ||
			entry.Before != "" && !ownership.Hash(entry.Before) || entry.After != "" && !ownership.Hash(entry.After) || entry.Mode > 0o777 {
			return nil, fmt.Errorf("publish: invalid journal entry")
		}
		entries[entry.Name] = entry
		folded[strings.ToLower(entry.Name)] = true
		if entry.Before != "" {
			before[entry.Name] = entry.Before
		}
		if entry.After != "" {
			after[entry.Name] = entry.After
		}
	}
	if j.Intent != "" && entries[j.Intent].Name == "" {
		return nil, fmt.Errorf("publish: unbound journal intent")
	}
	if (j.State == "complete" || j.State == "restored" || j.State == "staged") && j.Intent != "" || j.State == "applying" && j.Intent == "" {
		return nil, fmt.Errorf("publish: inconsistent journal intent state")
	}
	for i, name := range manifestLast(entries) {
		if j.Files[i].Name != name {
			return nil, fmt.Errorf("publish: noncanonical journal order")
		}
	}
	if len(after) == 0 || after[ownership.ManifestName] == "" {
		return nil, fmt.Errorf("publish: missing candidate manifest")
	}
	index, err := stageBody(&j)
	if err != nil || ownership.Digest(index) != j.StageSHA256 {
		return nil, fmt.Errorf("publish: journal stage identity differs")
	}
	stagedIndex, err := ownership.Read(e.root, stageDir(j.Token)+"/index.json")
	if err != nil || !bytes.Equal(stagedIndex, index) {
		return nil, fmt.Errorf("publish: stage inventory corrupt or absent")
	}
	staging, err := e.directory(stageDir(j.Token))
	if err != nil {
		return nil, err
	}
	if err := staging.Close(); err != nil {
		return nil, err
	}
	blobs := map[string][]byte{}
	total := int64(0)
	readBlob := func(sum string) ([]byte, error) {
		if body, found := blobs[sum]; found {
			return body, nil
		}
		name := stagePath(j.Token, sum)
		info, err := e.root.Lstat(name)
		if err != nil || info.Size() > (64<<20)-total {
			return nil, fmt.Errorf("publish: stage exceeds bounded inventory")
		}
		body, err := ownership.Read(e.root, name)
		if err != nil || ownership.Digest(body) != sum {
			return nil, fmt.Errorf("publish: corrupt staged bytes")
		}
		total += int64(len(body))
		if total > 64<<20 {
			return nil, fmt.Errorf("publish: stage exceeds read bound")
		}
		blobs[sum] = body
		return body, nil
	}
	for _, values := range []map[string]string{before, after} {
		if len(values) == 0 {
			continue
		}
		manifest := values[ownership.ManifestName]
		if manifest == "" {
			return nil, fmt.Errorf("publish: prior inventory lacks manifest")
		}
		body, err := readBlob(manifest)
		if err != nil {
			return nil, err
		}
		parsed, legacy, err := ownership.Parse(body)
		if err != nil || !maps.Equal(parsed, values) || legacy && manifest == after[ownership.ManifestName] {
			return nil, fmt.Errorf("publish: staged ownership inventory differs")
		}
		for name, sum := range values {
			body, err := readBlob(sum)
			if err != nil || ownership.Digest(body) != sum {
				return nil, fmt.Errorf("publish: corrupt staged bytes: %s", name)
			}
		}
	}
	seen := map[string]bool{j.Token: true}
	for _, ref := range j.Garbage {
		if !validToken(ref.Token) || !ownership.Hash(ref.Digest) || seen[ref.Token] {
			return nil, fmt.Errorf("publish: invalid prior stage reference")
		}
		seen[ref.Token] = true
	}
	e.journalSHA = ownership.Digest(body)
	return &j, nil
}

func (e *engine) cleanup(ctx context.Context, j *Journal) error {
	if err := e.check(ctx, "cleanup", ""); err != nil {
		return err
	}
	for len(j.Garbage) != 0 {
		ref := j.Garbage[0]
		if err := e.collect(ctx, ref); err != nil {
			return err
		}
		j.Garbage = slices.Clone(j.Garbage[1:])
		if err := e.writeJournal(ctx, j); err != nil {
			return err
		}
	}
	return nil
}

func (e *engine) collect(ctx context.Context, ref stageRef) (failure error) {
	directory := stageDir(ref.Token)
	entries, err := fs.ReadDir(e.root.FS(), directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	root, err := e.directory(directory)
	if err != nil {
		return err
	}
	defer func() { failure = errors.Join(failure, root.Close()) }()
	indexName := directory + "/index.json"
	body, err := ownership.Read(e.root, indexName)
	if errors.Is(err, os.ErrNotExist) && len(entries) == 0 {
		if err := e.root.Remove(directory); err != nil {
			return err
		}
		delete(e.directories, directory)
		return nil
	}
	if err != nil || ownership.Digest(body) != ref.Digest {
		return fmt.Errorf("publish: previous stage index changed")
	}
	var index stageIndex
	if err := strictJSON(body, &index); err != nil {
		return err
	}
	if index.Version != 1 || index.Token != ref.Token || len(index.Hashes) > 1026 || !slices.IsSorted(index.Hashes) {
		return fmt.Errorf("publish: invalid cleanup inventory")
	}
	expected := map[string]bool{"index.json": true}
	for _, sum := range index.Hashes {
		if !ownership.Hash(sum) || expected[sum] {
			return fmt.Errorf("publish: invalid cleanup hash")
		}
		expected[sum] = true
	}
	// Validate the entire cleanup set before deleting any previously staged byte.
	for _, entry := range entries {
		if !expected[entry.Name()] || !entry.Type().IsRegular() {
			return fmt.Errorf("publish: unowned or aliased stage cleanup entry")
		}
		if entry.Name() != "index.json" {
			body, err := ownership.Read(e.root, directory+"/"+entry.Name())
			if err != nil || ownership.Digest(body) != entry.Name() {
				return fmt.Errorf("publish: changed previous staged bytes")
			}
		}
	}
	for _, name := range append(slices.Clone(index.Hashes), "index.json") {
		if err := e.check(ctx, "cleanup-delete", name); err != nil {
			return err
		}
		if err := root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := e.root.Remove(directory); err != nil {
		return err
	}
	delete(e.directories, directory)
	return e.syncDir(ctx, stateName)
}
