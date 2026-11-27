package publish

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
)

func (e *engine) writeFile(ctx context.Context, name string, body []byte, mode os.FileMode) (failure error) {
	if err := e.check(ctx, "write", name); err != nil {
		return err
	}
	root, relative := e.root, name
	if directory := path.Dir(name); directory != "." {
		var err error
		root, err = e.directory(directory)
		if err != nil {
			return err
		}
		defer func() { failure = errors.Join(failure, root.Close()) }()
		relative = path.Base(name)
	}
	f, err := root.OpenFile(relative, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	n, writeErr := f.Write(body)
	if writeErr == nil && n != len(body) {
		writeErr = fmt.Errorf("publish: short write: %s", name)
	}
	if writeErr == nil {
		writeErr = e.check(ctx, "sync", name)
	}
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr == nil && closeErr == nil {
		closeErr = e.check(ctx, "close", name)
	}
	return errors.Join(writeErr, closeErr)
}

func (e *engine) directory(name string) (*os.Root, error) {
	info, err := e.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("publish: aliased stage directory")
	}
	if previous := e.directories[name]; previous != nil && !os.SameFile(previous, info) {
		return nil, fmt.Errorf("publish: changed stage directory")
	}
	root, err := e.root.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	actual, err := root.Stat(".")
	if err != nil || !os.SameFile(info, actual) {
		return nil, errors.Join(fmt.Errorf("publish: raced stage directory"), err, root.Close())
	}
	e.directories[name] = info
	return root, nil
}

func (e *engine) replace(ctx context.Context, name, temporary string, body []byte, mode os.FileMode, recheck func() error) (failure error) {
	if err := e.writeFile(ctx, temporary, body, mode); err != nil {
		return err
	}
	original, err := e.root.Lstat(temporary)
	if err != nil {
		return err
	}
	// A process kill can leave this explicitly owned pending file. Recovery
	// removes only this reserved regular-file name after verifying its journal.
	defer func() {
		if current, err := e.root.Lstat(temporary); err == nil {
			if os.SameFile(original, current) {
				failure = errors.Join(failure, e.root.Remove(temporary))
			} else {
				failure = errors.Join(failure, fmt.Errorf("publish: pending file identity changed"))
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			failure = errors.Join(failure, err)
		}
	}()
	if err := e.check(ctx, "rename", name); err != nil {
		return err
	}
	current, err := e.root.Lstat(temporary)
	if err != nil || !os.SameFile(original, current) {
		return fmt.Errorf("publish: pending replacement identity changed")
	}
	verified, err := ownership.Read(e.root, temporary)
	if err != nil || ownership.Digest(verified) != ownership.Digest(body) {
		return fmt.Errorf("publish: pending replacement bytes changed")
	}
	if recheck != nil {
		if err := recheck(); err != nil {
			return err
		}
	}
	if err := e.root.Rename(temporary, name); err != nil {
		return err
	}
	return e.syncDir(ctx, path.Dir(name))
}

func (e *engine) pending(ctx context.Context) error {
	for _, name := range []string{filePending, journalPending} {
		if _, err := e.root.Lstat(name); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if _, err := ownership.Read(e.root, name); err != nil {
			return err
		}
		if err := e.check(ctx, "pending-cleanup", name); err != nil {
			return err
		}
		if err := e.root.Remove(name); err != nil {
			return err
		}
	}
	return e.syncDir(ctx, ".")
}
