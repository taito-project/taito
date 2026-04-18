package cache

import (
	"fmt"
	"os"
	"path/filepath"
)

// PruneEntry represents a single item in the packages cache.
type PruneEntry struct {
	Name  string
	IsDir bool
	Size  int64 // bytes
}

// ScanPackages reads the packages directory and returns all entries with sizes.
// Returns an empty slice (not an error) if the directory does not exist.
// Skips the index.json file.
func ScanPackages(packagesDir string) ([]PruneEntry, error) {
	dirEntries, err := os.ReadDir(packagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []PruneEntry
	for _, de := range dirEntries {
		if de.Name() == "index.json" {
			continue
		}

		fullPath := filepath.Join(packagesDir, de.Name())
		var size int64

		if de.IsDir() {
			s, err := DirSize(fullPath)
			if err != nil {
				return nil, fmt.Errorf("calculating size of %s: %w", de.Name(), err)
			}
			size = s
		} else {
			info, err := de.Info()
			if err != nil {
				return nil, fmt.Errorf("reading info for %s: %w", de.Name(), err)
			}
			size = info.Size()
		}

		entries = append(entries, PruneEntry{
			Name:  de.Name(),
			IsDir: de.IsDir(),
			Size:  size,
		})
	}

	return entries, nil
}

// RemovePackages deletes all listed items from the packages directory.
func RemovePackages(packagesDir string, entries []PruneEntry) error {
	for _, e := range entries {
		fullPath := filepath.Join(packagesDir, e.Name)
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("removing %s: %w", e.Name, err)
		}
	}
	return nil
}

// EntryDisplayName returns a human-readable name for a prune entry.
// If the index contains a mapping for the directory hash, it shows the
// original reference. Otherwise, falls back to the raw name.
func EntryDisplayName(e PruneEntry, idx *Index) string {
	if idx != nil {
		if entry, ok := idx.Entries[e.Name]; ok {
			suffix := ""
			if e.IsDir {
				suffix = "/"
			}
			return entry.Reference + suffix
		}
	}
	return DisplayName(e)
}

// DisplayName returns the entry name with a trailing / for directories.
func DisplayName(e PruneEntry) string {
	if e.IsDir {
		return e.Name + "/"
	}
	return e.Name
}

// DirSize walks a directory tree and sums all file sizes.
func DirSize(path string) (int64, error) {
	var total int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// FormatSize formats bytes into a human-readable string.
func FormatSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ItemWord returns "item" or "items" depending on count.
func ItemWord(count int) string {
	if count == 1 {
		return "item"
	}
	return "items"
}
