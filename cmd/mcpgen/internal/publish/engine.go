// Package publish implements rooted, journaled publication. Per-file replacement
// is not a multi-file transaction; the verified manifest is always published last.
package publish

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
)

// JournalName names the recoverable publication journal below the output root.
const JournalName = ".mcpgen-journal.json"

// LockName names the exclusive publication and recovery lock.
const LockName = ".mcpgen-lock"
const stateName = ".mcpgen-state"
const filePending = ".mcpgen-file.pending"
const journalPending = ".mcpgen-journal.pending"

// Hook is a per-invocation fault/checkpoint seam; production normally leaves it nil.
type Hook func(point, name string) error

// Options contains the already qualified bytes and independently verified prior
// inventory. Bootstrap must verify frozen Git objects before legacy publication.
type Options struct {
	Previous   map[string]string
	Files      map[string][]byte
	Bootstrap  func(context.Context) error
	Revalidate func() error
	Hook       Hook
}

// Entry binds an output name to its original and candidate content identities.
type Entry struct {
	Name   string `json:"name"`
	Before string `json:"original-sha256,omitempty"`
	After  string `json:"candidate-sha256,omitempty"`
	Mode   uint32 `json:"original-mode,omitempty"`
}

type stageRef struct {
	Token  string `json:"token"`
	Digest string `json:"index-sha256"`
}

// Journal retains no absolute paths or caller-controlled staging filenames.
type Journal struct {
	Version     int        `json:"version"`
	Token       string     `json:"token"`
	State       string     `json:"state"`
	Intent      string     `json:"intent,omitempty"`
	StageSHA256 string     `json:"stage-index-sha256"`
	Files       []Entry    `json:"files"`
	Garbage     []stageRef `json:"previous-stages,omitempty"`
}

// Receipt describes checked native guarantees without promising power-loss or
// multi-file atomicity. Current backup bytes remain available for recovery.
type Receipt struct {
	State      string   `json:"state"`
	Changed    []string `json:"changed"`
	Durability []string `json:"durability"`
}

type engine struct {
	repo, root           *os.Root
	identity             os.FileInfo
	hook                 Hook
	limitedDirectorySync bool
	journalSHA           string
	directories          map[string]os.FileInfo
}

func open(repo string, create bool, hook Hook) (*engine, error) {
	base, err := os.OpenRoot(repo)
	if err != nil {
		return nil, err
	}
	info, err := base.Lstat("protocol")
	if errors.Is(err, os.ErrNotExist) && create {
		err = base.Mkdir("protocol", 0o755)
		if err == nil {
			info, err = base.Lstat("protocol")
		}
	}
	if err != nil {
		return nil, errors.Join(err, base.Close())
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.Join(fmt.Errorf("publish: output directory is aliased or not a directory"), base.Close())
	}
	if hook != nil {
		if err := hook("before-root-open", "protocol"); err != nil {
			return nil, errors.Join(err, base.Close())
		}
	}
	root, err := base.OpenRoot("protocol")
	if err != nil {
		return nil, errors.Join(err, base.Close())
	}
	e := &engine{repo: base, root: root, identity: info, hook: hook, directories: map[string]os.FileInfo{}}
	if err := e.check(context.Background(), "open", "protocol"); err != nil {
		return nil, errors.Join(err, e.close())
	}
	return e, nil
}

func (e *engine) close() error { return errors.Join(e.root.Close(), e.repo.Close()) }

func (e *engine) check(ctx context.Context, point, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if e.hook != nil {
		if err := e.hook(point, name); err != nil {
			return err
		}
	}
	current, err := e.repo.Lstat("protocol")
	if err != nil || !os.SameFile(e.identity, current) || current.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("publish: destination directory identity changed")
	}
	if info, err := e.root.Lstat(stateName); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("publish: aliased staging directory")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for name, previous := range e.directories {
		current, err := e.root.Lstat(name)
		if err != nil || !os.SameFile(previous, current) || current.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("publish: staging directory identity changed: %s", name)
		}
	}
	return nil
}

func token() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func validToken(value string) bool {
	body, err := hex.DecodeString(value)
	return err == nil && len(body) == 16 && value == strings.ToLower(value)
}

type lock struct {
	file     *os.File
	identity os.FileInfo
	body     []byte
}

func (e *engine) acquire(ctx context.Context) (*lock, error) {
	if err := e.check(ctx, "lock", LockName); err != nil {
		return nil, err
	}
	id, err := token()
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(struct {
		Version int    `json:"version"`
		Token   string `json:"token"`
		PID     int    `json:"pid"`
	}{1, id, os.Getpid()})
	if err != nil {
		return nil, err
	}
	f, err := e.root.OpenFile(LockName, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("publish: exclusive lock conflict; never removed by age: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		return nil, errors.Join(err, f.Close(), e.root.Remove(LockName))
	}
	l := &lock{f, info, body}
	_, err = f.Write(body)
	if err == nil {
		err = f.Sync()
	}
	if err != nil {
		return nil, errors.Join(err, e.release(l, false))
	}
	return l, nil
}

func (e *engine) release(l *lock, verify bool) error {
	closeErr := l.file.Close()
	info, err := e.root.Lstat(LockName)
	if err != nil || !os.SameFile(l.identity, info) {
		return errors.Join(closeErr, fmt.Errorf("publish: lock identity changed"), err)
	}
	if verify {
		body, err := ownership.Read(e.root, LockName)
		if err != nil || string(body) != string(l.body) {
			return errors.Join(closeErr, fmt.Errorf("publish: lock owner changed"), err)
		}
	}
	return errors.Join(closeErr, e.root.Remove(LockName))
}

func (e *engine) syncDir(ctx context.Context, directory string) error {
	if err := e.check(ctx, "directory-sync", directory); err != nil {
		return err
	}
	f, err := e.root.Open(directory)
	if err != nil {
		return err
	}
	syncErr := f.Sync()
	closeErr := f.Close()
	// A standard Windows directory handle cannot be flushed with File.Sync.
	// Report this native limitation, while keeping every file-sync error fatal.
	if runtime.GOOS == "windows" && errors.Is(syncErr, os.ErrPermission) {
		e.limitedDirectorySync = true
		syncErr = nil
	}
	return errors.Join(syncErr, closeErr)
}

func (e *engine) receipt(state string, changed []string) *Receipt {
	durability := []string{"file write/sync/close checked", "per-file replacement; manifest last", "no universal power-loss guarantee"}
	if e.limitedDirectorySync {
		durability = append(durability, "directory sync unavailable through the native standard-library handle")
	} else {
		durability = append(durability, "directory sync checked")
	}
	return &Receipt{State: state, Changed: changed, Durability: durability}
}

// Publish installs the exact qualified bytes. It never re-renders a candidate.
func Publish(ctx context.Context, repo string, options Options) (_ *Receipt, failure error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	after, err := ownership.Candidate(options.Files)
	if err != nil {
		return nil, err
	}
	e, err := open(repo, true, options.Hook)
	if err != nil {
		return nil, err
	}
	defer func() { failure = errors.Join(failure, e.close()) }()
	l, err := e.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { failure = errors.Join(failure, e.release(l, true)) }()
	previous, legacy, err := ownership.Existing(e.root, "")
	if err != nil {
		return nil, err
	}
	if !maps.Equal(previous, options.Previous) {
		return nil, fmt.Errorf("publish: prior inventory changed after qualification")
	}
	if legacy {
		if options.Bootstrap == nil {
			return nil, fmt.Errorf("publish: explicit verified legacy bootstrap required")
		}
		if err := options.Bootstrap(ctx); err != nil {
			return nil, err
		}
	}
	prior, err := e.readJournal()
	if err != nil {
		return nil, err
	}
	if prior != nil {
		if prior.State != "complete" && prior.State != "restored" {
			return nil, fmt.Errorf("publish: unfinished journal requires recovery")
		}
		if err := e.verifyCurrent(prior, true); err != nil {
			return nil, err
		}
	} else {
		for _, name := range []string{filePending, journalPending} {
			if _, err := e.root.Lstat(name); !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("publish: unowned pending metadata: %s", name)
			}
		}
	}
	if err := e.collisions(previous, after); err != nil {
		return nil, err
	}
	if options.Revalidate != nil {
		if err := options.Revalidate(); err != nil {
			return nil, err
		}
	}
	if prior != nil {
		if err := e.pending(ctx); err != nil {
			return nil, err
		}
		if err := e.cleanup(ctx, prior); err != nil {
			return nil, err
		}
	}
	if maps.Equal(previous, after) {
		return e.receipt("unchanged", []string{}), nil
	}
	id, err := token()
	if err != nil {
		return nil, err
	}
	j := &Journal{Version: 2, Token: id, State: "staged", Files: []Entry{}}
	if prior != nil {
		j.Garbage = append(slices.Clone(prior.Garbage), stageRef{prior.Token, prior.StageSHA256})
		if len(j.Garbage) > 8 {
			return nil, fmt.Errorf("publish: prior stage cleanup required")
		}
	}
	all := maps.Clone(after)
	for name, sum := range previous {
		all[name] = sum
	}
	blobs := map[string][]byte{}
	for _, name := range manifestLast(all) {
		entry := Entry{Name: name, Before: previous[name], After: after[name]}
		if entry.Before != "" {
			body, err := ownership.Read(e.root, name)
			if err != nil || ownership.Digest(body) != entry.Before {
				return nil, fmt.Errorf("publish: original changed: %s", name)
			}
			info, err := e.root.Lstat(name)
			if err != nil {
				return nil, err
			}
			entry.Mode = uint32(info.Mode().Perm())
			blobs[entry.Before] = body
		}
		if entry.After != "" {
			blobs[entry.After] = options.Files[name]
		}
		j.Files = append(j.Files, entry)
	}
	if err := e.stage(ctx, j, blobs); err != nil {
		return nil, err
	}
	if options.Revalidate != nil {
		if err := options.Revalidate(); err != nil {
			return nil, err
		}
	}
	if err := e.writeJournal(ctx, j); err != nil {
		return nil, err
	}
	if err := e.check(ctx, "staged", ""); err != nil {
		return nil, err
	}
	return e.apply(ctx, j, "complete")
}

func manifestLast[V any](values map[string]V) []string {
	names := ownership.Sorted(values)
	index := slices.Index(names, ownership.ManifestName)
	if index >= 0 {
		names = append(append(names[:index:index], names[index+1:]...), ownership.ManifestName)
	}
	return names
}

func (e *engine) collisions(previous, proposed map[string]string) error {
	entries, err := fs.ReadDir(e.root.FS(), ".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		for name := range proposed {
			if strings.EqualFold(entry.Name(), name) && (entry.Name() != name || previous[name] == "") {
				return fmt.Errorf("publish: unowned or case-colliding output: %s", entry.Name())
			}
		}
	}
	return nil
}

// Inspect verifies the journal/staged bytes without locking or writing anything.
func Inspect(repo string) (_ *Journal, failure error) {
	e, err := open(repo, false, nil)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { failure = errors.Join(failure, e.close()) }()
	j, err := e.readJournal()
	if err != nil || j == nil {
		return j, err
	}
	return j, e.verifyCurrent(j, j.State == "complete" || j.State == "restored")
}

// Recover completes or restores the latest verified transaction. A stale lock
// still requires operator resolution; recovery never guesses process liveness.
func Recover(ctx context.Context, repo, action string, hook Hook) (_ *Receipt, failure error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if action != "complete" && action != "restore" {
		return nil, fmt.Errorf("publish: invalid recovery action")
	}
	e, err := open(repo, false, hook)
	if err != nil {
		return nil, err
	}
	defer func() { failure = errors.Join(failure, e.close()) }()
	l, err := e.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { failure = errors.Join(failure, e.release(l, true)) }()
	j, err := e.readJournal()
	if err != nil {
		return nil, err
	}
	if j == nil {
		return e.receipt("no-journal", []string{}), nil
	}
	if err := e.verifyCurrent(j, j.State == "complete" || j.State == "restored"); err != nil {
		return nil, err
	}
	if err := e.pending(ctx); err != nil {
		return nil, err
	}
	return e.apply(ctx, j, action)
}

func (e *engine) current(name string) (string, error) {
	body, err := ownership.Read(e.root, name)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return ownership.Digest(body), nil
}

func (e *engine) verifyCurrent(j *Journal, terminal bool) error {
	for _, entry := range j.Files {
		sum, err := e.current(entry.Name)
		if err != nil {
			return err
		}
		inFlightMissing := sum == "" && j.Intent == entry.Name
		if terminal {
			want := entry.After
			if j.State == "restored" {
				want = entry.Before
			}
			if sum != want {
				return fmt.Errorf("publish: terminal output changed: %s", entry.Name)
			}
		} else if sum != entry.Before && sum != entry.After && !inFlightMissing {
			return fmt.Errorf("publish: manual edit conflicts with recovery: %s", entry.Name)
		}
	}
	return nil
}

func (e *engine) apply(ctx context.Context, j *Journal, action string) (*Receipt, error) {
	if err := e.verifyCurrent(j, j.State == "complete" || j.State == "restored"); err != nil {
		return nil, err
	}
	changed := []string{}
	for _, entry := range j.Files {
		want, mode := entry.After, fs.FileMode(0o644)
		if action == "restore" {
			want, mode = entry.Before, fs.FileMode(entry.Mode)
		}
		have, err := e.current(entry.Name)
		if err != nil {
			return nil, err
		}
		if have == want {
			continue
		}
		j.State, j.Intent = "applying", entry.Name
		if err := e.writeJournal(ctx, j); err != nil {
			return nil, err
		}
		if err := e.check(ctx, "intent", entry.Name); err != nil {
			return nil, err
		}
		if current, err := e.current(entry.Name); err != nil || current != have {
			return nil, fmt.Errorf("publish: intervening output edit: %s", entry.Name)
		}
		if want == "" {
			if err := e.check(ctx, "delete", entry.Name); err != nil {
				return nil, err
			}
			if current, err := e.current(entry.Name); err != nil || current != have {
				return nil, fmt.Errorf("publish: intervening deletion edit: %s", entry.Name)
			}
			if err := e.root.Remove(entry.Name); err != nil {
				return nil, err
			}
			if err := e.syncDir(ctx, "."); err != nil {
				return nil, err
			}
		} else {
			body, err := ownership.Read(e.root, stagePath(j.Token, want))
			if err != nil || ownership.Digest(body) != want {
				return nil, fmt.Errorf("publish: staged content changed: %s", entry.Name)
			}
			if err := e.replace(ctx, entry.Name, filePending, body, mode, func() error {
				current, err := e.current(entry.Name)
				if err != nil || current != have {
					return fmt.Errorf("publish: intervening replacement edit: %s", entry.Name)
				}
				return nil
			}); err != nil {
				return nil, err
			}
		}
		changed = append(changed, entry.Name)
		if err := e.check(ctx, "after-replace", entry.Name); err != nil {
			return nil, err
		}
	}
	j.Intent = ""
	j.State = "complete"
	if action == "restore" {
		j.State = "restored"
	}
	if err := e.writeJournal(ctx, j); err != nil {
		return nil, err
	}
	if err := e.cleanup(ctx, j); err != nil {
		return nil, err
	}
	return e.receipt(j.State, changed), nil
}
