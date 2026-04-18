// Package tarutil provides shared utilities for extracting tar.gz archives.
package tarutil

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractOptions configures how a tar.gz archive is extracted.
type ExtractOptions struct {
	// StripComponents removes this many leading path components from each
	// entry (similar to tar --strip-components). For example, 1 strips the
	// archive root directory.
	StripComponents int

	// SubdirFilter, when non-empty, extracts only entries under this prefix
	// and strips it from output paths. Takes precedence over StripComponents.
	SubdirFilter string

	// CleanTarget removes the target directory before extracting.
	CleanTarget bool
}

// Extract opens a tar.gz file at tarGzPath and extracts it to targetDir
// according to the given options.
func Extract(tarGzPath, targetDir string, opts ExtractOptions) error {
	if err := prepareTarget(targetDir, opts.CleanTarget); err != nil {
		return err
	}

	return WithTarGz(tarGzPath, func(tr *tar.Reader) error {
		return extractEntries(tr, targetDir, opts)
	})
}

// extractEntries iterates over tar entries and writes matching ones to targetDir.
func extractEntries(tr *tar.Reader, targetDir string, opts ExtractOptions) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		relPath, ok := resolveRelPath(header.Name, opts)
		if !ok {
			continue
		}

		outPath := filepath.Join(targetDir, filepath.FromSlash(relPath))
		if !isInsideDir(outPath, targetDir) {
			continue
		}

		if err := writeEntry(outPath, header, tr); err != nil {
			return err
		}
	}
}

// prepareTarget ensures the target directory exists, optionally cleaning it first.
func prepareTarget(targetDir string, clean bool) error {
	if clean {
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("clean target: %w", err)
		}
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	return nil
}

// resolveRelPath computes the relative output path for a tar entry based on
// the extract options. Returns (relPath, true) if the entry should be
// extracted, or ("", false) if it should be skipped.
func resolveRelPath(name string, opts ExtractOptions) (string, bool) {
	name = filepath.ToSlash(name)

	if opts.SubdirFilter != "" {
		return resolveSubdirPath(name, opts.SubdirFilter)
	}

	if opts.StripComponents > 0 {
		return resolveStrippedPath(name, opts.StripComponents)
	}

	// No stripping — use path as-is, but skip empty names.
	trimmed := strings.TrimSuffix(name, "/")
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

// resolveSubdirPath extracts only entries under subdirFilter and strips the prefix.
func resolveSubdirPath(name, subdirFilter string) (string, bool) {
	prefix := subdirFilter + "/"
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	rel := strings.TrimPrefix(name, prefix)
	if rel == "" {
		return "", false
	}
	return rel, true
}

// resolveStrippedPath strips the given number of leading path components.
func resolveStrippedPath(name string, stripCount int) (string, bool) {
	parts := strings.SplitN(name, "/", stripCount+1)
	if len(parts) <= stripCount {
		return "", false
	}
	rel := parts[stripCount]
	if rel == "" {
		return "", false
	}
	return rel, true
}

// isInsideDir checks that outPath is within targetDir (prevents path traversal).
func isInsideDir(outPath, targetDir string) bool {
	return strings.HasPrefix(filepath.Clean(outPath), filepath.Clean(targetDir))
}

// writeEntry writes a single tar entry (directory or regular file) to disk.
func writeEntry(outPath string, header *tar.Header, tr io.Reader) error {
	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(outPath, 0755)
	case tar.TypeReg:
		return writeFile(outPath, header, tr)
	default:
		return nil
	}
}

// writeFile writes a regular file entry to disk.
func writeFile(outPath string, header *tar.Header, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
	if err != nil {
		return err
	}
	if _, err := io.Copy(outFile, r); err != nil {
		_ = outFile.Close()
		return err
	}
	return outFile.Close()
}
