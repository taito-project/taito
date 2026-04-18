package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// IndexEntry holds metadata about a cached package.
type IndexEntry struct {
	Reference string `json:"reference"`
	SpecType  string `json:"specType,omitempty"`
	Format    string `json:"format"`
	CreatedAt string `json:"createdAt"`
}

// Index is the top-level structure for the packages index file.
type Index struct {
	Entries map[string]IndexEntry `json:"entries"`
}

// indexMu serializes concurrent reads/writes to the index file.
var indexMu sync.Mutex

// HashReference computes a 32-character hex digest of the full OCI reference.
// The reference is hashed exactly as provided (no normalization).
func HashReference(reference string) string {
	h := sha256.Sum256([]byte(reference))
	return hex.EncodeToString(h[:])[:32]
}

// PackagePath returns the canonical cache directory path for a given reference.
// The path is <cacheDir>/packages/<sha256[:32]>.
func PackagePath(cacheDir, reference string) string {
	return filepath.Join(cacheDir, "packages", HashReference(reference))
}

// indexPath returns the path to the index.json file.
func indexPath(cacheDir string) string {
	return filepath.Join(cacheDir, "packages", "index.json")
}

// LoadIndex reads the index.json from the cache directory. Returns an empty
// index (not an error) if the file does not exist.
func LoadIndex(cacheDir string) (*Index, error) {
	indexMu.Lock()
	defer indexMu.Unlock()

	return loadIndexLocked(cacheDir)
}

// loadIndexLocked reads the index without acquiring the mutex.
// Caller must hold indexMu.
func loadIndexLocked(cacheDir string) (*Index, error) {
	p := indexPath(cacheDir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{Entries: make(map[string]IndexEntry)}, nil
		}
		return nil, fmt.Errorf("read index: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	if idx.Entries == nil {
		idx.Entries = make(map[string]IndexEntry)
	}
	return &idx, nil
}

// SaveIndex writes the index to disk. The packages directory is created if
// it does not exist.
func SaveIndex(cacheDir string, idx *Index) error {
	indexMu.Lock()
	defer indexMu.Unlock()

	return saveIndexLocked(cacheDir, idx)
}

// saveIndexLocked writes the index without acquiring the mutex.
// Caller must hold indexMu.
func saveIndexLocked(cacheDir string, idx *Index) error {
	dir := filepath.Join(cacheDir, "packages")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create packages dir: %w", err)
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	p := indexPath(cacheDir)
	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}

// AddEntry loads the index, upserts an entry for the given reference, and
// saves it back. This is the convenience function for package/pull commands
// to call after a successful operation.
func AddEntry(cacheDir, reference, specType, format string) error {
	indexMu.Lock()
	defer indexMu.Unlock()

	idx, err := loadIndexLocked(cacheDir)
	if err != nil {
		return err
	}

	hash := HashReference(reference)
	idx.Entries[hash] = IndexEntry{
		Reference: reference,
		SpecType:  specType,
		Format:    format,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	return saveIndexLocked(cacheDir, idx)
}

// RemoveEntry loads the index, removes the entry for the given hash, and
// saves it back.
func RemoveEntry(cacheDir, hash string) error {
	indexMu.Lock()
	defer indexMu.Unlock()

	idx, err := loadIndexLocked(cacheDir)
	if err != nil {
		return err
	}

	delete(idx.Entries, hash)
	return saveIndexLocked(cacheDir, idx)
}

// ClearIndex removes all entries from the index (or removes the file).
func ClearIndex(cacheDir string) error {
	p := indexPath(cacheDir)
	err := os.Remove(p)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove index: %w", err)
	}
	return nil
}
