// Package ownership verifies output inventories independently of emission.
// A filename or generated-code header never establishes replacement authority.
package ownership

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"sort"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

// ManifestName names the generated output ownership record.
const ManifestName = "generated-manifest.json"

// LegacyCommit identifies the immutable baseline eligible for explicit bootstrap.
const LegacyCommit = "28a9284515ce5e82892387519929b2ab86caffb4"

// LegacyManifestSHA256 independently pins the historical hashless manifest.
const LegacyManifestSHA256 = "6d658b8c2f55c9caced6feb1d974ff6c7145b10b1da7d8a95e184107fd0b1c2b"

var legacyHashes = map[string]string{
	"aliases.go":   "51e46c107b583dcec119dcfd0435e7ef1fcf5a8c24f8d35480946d159e3a1f33",
	"constants.go": "d9681ac8ee6e586dd3e32c16fdb8bb53cf73c065e9e5bbaa6f0d467f1cbf3a62",
	"owned.go":     "0ec535d11557f5dd50ff6a4ef9772b6c54f05e06533ff2b86acc007c0948ab3e",
}

// LegacyHashes returns independent historical pins, excluding the manifest.
func LegacyHashes() map[string]string { return maps.Clone(legacyHashes) }

// Manifest is the versioned content-hashed output contract.
type Manifest struct {
	Version  int               `json:"version"`
	Revision string            `json:"revision"`
	Files    []string          `json:"files"`
	SHA256   map[string]string `json:"sha256"`
}

// Digest returns a lower-case SHA-256 identity.
func Digest(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }

// Name accepts only portable, unaliased, flat output names.
func Name(name string) error {
	if name == "" || name[0] == '.' || strings.HasSuffix(name, ".") {
		return fmt.Errorf("ownership: unsafe output name %q", name)
	}
	for _, r := range name {
		portable := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-'
		if !portable {
			return fmt.Errorf("ownership: unsafe output name %q", name)
		}
	}
	stem, _, _ := strings.Cut(strings.ToUpper(name), ".")
	if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" ||
		len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '1' && stem[3] <= '9' {
		return fmt.Errorf("ownership: reserved output name %q", name)
	}
	return nil
}

// Hash validates an independently supplied content identity.
func Hash(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && strings.ToLower(value) == value
}

// Parse verifies strict shape, portable names and complete hashed ownership.
// The frozen legacy bytes are recognized as migration input, never by shape.
func Parse(body []byte) (map[string]string, bool, error) {
	if Digest(body) == LegacyManifestSHA256 {
		result := LegacyHashes()
		result[ManifestName] = LegacyManifestSHA256
		return result, true, nil
	}
	if _, err := load.New().LoadBytes(ManifestName, body); err != nil {
		return nil, false, fmt.Errorf("ownership: invalid manifest: %w", err)
	}
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil || manifest.Version != 2 || manifest.Revision == "" || len(manifest.Files) == 0 || len(manifest.Files) > 256 || len(manifest.SHA256) != len(manifest.Files) || !sort.StringsAreSorted(manifest.Files) {
		return nil, false, fmt.Errorf("ownership: unsupported or incomplete hashed manifest")
	}
	seen := map[string]bool{}
	result := map[string]string{ManifestName: Digest(body)}
	for _, name := range manifest.Files {
		if err := Name(name); err != nil {
			return nil, false, err
		}
		if name == ManifestName || seen[strings.ToLower(name)] || !Hash(manifest.SHA256[name]) {
			return nil, false, fmt.Errorf("ownership: colliding or unhashed output %q", name)
		}
		seen[strings.ToLower(name)] = true
		result[name] = manifest.SHA256[name]
	}
	return result, false, nil
}

// Candidate verifies every proposed byte against a hashed manifest.
func Candidate(files map[string][]byte) (map[string]string, error) {
	manifest, found := files[ManifestName]
	if !found {
		return nil, fmt.Errorf("ownership: proposed manifest missing")
	}
	expected, legacy, err := Parse(manifest)
	if err != nil {
		return nil, err
	}
	if legacy || len(expected) != len(files) {
		return nil, fmt.Errorf("ownership: proposed inventory differs")
	}
	for name, sum := range expected {
		body, found := files[name]
		if !found || Digest(body) != sum {
			return nil, fmt.Errorf("ownership: proposed output identity differs: %s", name)
		}
	}
	return expected, nil
}

// Read refuses symlinks, special files and raced identities and bounds reads.
func Read(root *os.Root, name string) ([]byte, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() > 8<<20 {
		return nil, fmt.Errorf("ownership: not a bounded regular file: %s", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	after, statErr := file.Stat()
	if statErr != nil || !os.SameFile(before, after) {
		return nil, errors.Join(fmt.Errorf("ownership: file identity changed: %s", name), statErr, file.Close())
	}
	body, readErr := io.ReadAll(io.LimitReader(file, (8<<20)+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return nil, err
	}
	if len(body) > 8<<20 {
		return nil, fmt.Errorf("ownership: file exceeds read bound: %s", name)
	}
	return body, nil
}

// Existing verifies every prior owned file. An absent manifest establishes no
// ownership; its caller still rejects all proposed-name collisions.
func Existing(root *os.Root, prefix string) (map[string]string, bool, error) {
	if prefix != "" {
		info, err := root.Lstat(strings.TrimSuffix(prefix, "/"))
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, false, nil
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, false, fmt.Errorf("ownership: output directory is absent, aliased or not a directory")
		}
	}
	body, err := Read(root, prefix+ManifestName)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	expected, legacy, err := Parse(body)
	if err != nil {
		return nil, false, err
	}
	infos := []os.FileInfo{}
	for _, name := range Sorted(expected) {
		body, err := Read(root, prefix+name)
		if err != nil || Digest(body) != expected[name] {
			return nil, false, fmt.Errorf("ownership: prior output missing or changed: %s", name)
		}
		info, err := root.Lstat(prefix + name)
		if err != nil {
			return nil, false, err
		}
		for _, prior := range infos {
			if os.SameFile(info, prior) {
				return nil, false, fmt.Errorf("ownership: aliased owned outputs")
			}
		}
		infos = append(infos, info)
	}
	return expected, legacy, nil
}

// Sorted returns deterministic inventory keys.
func Sorted[V any](values map[string]V) []string {
	result := make([]string, 0, len(values))
	for name := range values {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
