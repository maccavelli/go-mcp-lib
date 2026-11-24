package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// digestWriter accumulates a SHA-256 over written bytes.
type digestWriter struct {
	sum [32]byte
	set bool
}

func newSHA256() *digestWriter { return &digestWriter{} }

func (d *digestWriter) write(data []byte) {
	d.sum = sha256.Sum256(data)
	d.set = true
}

func (d *digestWriter) hex() string {
	return hex.EncodeToString(d.sum[:])
}

// genLock is an exclusive per-destination generation lock.
type genLock struct {
	path string
	f    *os.File
}

const lockName = ".mcpgen-lock"

// acquireLock takes the exclusive lock, refusing when another live
// generator holds it. Age alone never justifies stealing: an
// unverifiable owner is a conflict requiring operator resolution.
func acquireLock(dest string) (*genLock, error) {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dest, lockName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o644)
	if err != nil {
		if os.IsExist(err) {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil, fmt.Errorf("generate: lock exists but is unreadable; operator resolution required")
			}
			return nil, fmt.Errorf(
				"generate: another generator holds the lock (holder: %s); resolve or run recover",
				string(data),
			)
		}
		return nil, err
	}
	holder := fmt.Sprintf("pid-%d", os.Getpid())
	if _, err := f.WriteString(holder); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, err
	}
	return &genLock{path: path, f: f}, nil
}

func (l *genLock) holder() string {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return "unknown"
	}
	return string(data)
}

func (l *genLock) release() error {
	if err := l.f.Close(); err != nil {
		return err
	}
	return os.Remove(l.path)
}
