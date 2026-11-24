package acquire

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func pin() (string, Options) {
	body := "schema-body-for-pinning"
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:]), Options{
		Revision: "2026-07-28",
		Commit:   "ac00313ff7f5e9f47b0e0d1ebdf9c0ad6411380b",
	}
}

func TestRunFetchesAndVerifies(t *testing.T) {
	want, opts := pin()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "schema-body-for-pinning")
	}))
	defer server.Close()
	opts.URL = server.URL
	opts.InputsDir = t.TempDir()
	opts.SHA256 = want
	result, err := Run(opts)
	if err != nil {
		t.Fatalf("FAIL run: %v", err)
	}
	if result.SHA256 != want || result.Existing {
		t.Fatalf("FAIL result: %+v want hash %s fresh", result, want)
	}
	data, err := os.ReadFile(filepath.Join(opts.InputsDir, "schema.json"))
	if err != nil || string(data) != "schema-body-for-pinning" {
		t.Fatalf("FAIL written file: %v %q", err, data)
	}
}

func TestRunVerifiesExistingWithoutNetwork(t *testing.T) {
	want, opts := pin()
	opts.InputsDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(opts.InputsDir, "schema.json"), []byte("schema-body-for-pinning"), 0o644); err != nil {
		t.Fatal(err)
	}
	// URL points at a server that fails every request: the existing file
	// must satisfy the pin without any fetch.
	refused := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer refused.Close()
	opts.URL = refused.URL
	opts.SHA256 = want
	result, err := Run(opts)
	if err != nil {
		t.Fatalf("FAIL run: %v", err)
	}
	if !result.Existing {
		t.Fatalf("FAIL expected existing-file path: %+v", result)
	}
}

func TestRunRejectsHashMismatchWithoutWriting(t *testing.T) {
	_, opts := pin()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "tampered-bytes")
	}))
	defer server.Close()
	opts.URL = server.URL
	opts.InputsDir = t.TempDir()
	opts.SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := Run(opts); err == nil {
		t.Fatal("FAIL expected hash mismatch error")
	}
	if _, err := os.Stat(filepath.Join(opts.InputsDir, "schema.json")); !os.IsNotExist(err) {
		t.Fatalf("FAIL mismatched fetch must not be written: %v", err)
	}
	// No temp residue either.
	entries, _ := os.ReadDir(opts.InputsDir)
	if len(entries) != 0 {
		t.Fatalf("FAIL temp residue: %v", entries)
	}
}

func TestRunRejectsExistingMismatch(t *testing.T) {
	_, opts := pin()
	opts.InputsDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(opts.InputsDir, "schema.json"), []byte("corrupted"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts.SHA256 = "1111111111111111111111111111111111111111111111111111111111111111"
	if _, err := Run(opts); err == nil {
		t.Fatal("FAIL expected existing-file mismatch error")
	}
}

func TestRunRequiresPins(t *testing.T) {
	opts := Options{}
	if _, err := Run(opts); err == nil {
		t.Fatal("FAIL expected required-field error")
	}
}
