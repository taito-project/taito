package archive

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/taito-project/taito/internal/spec"
)

// CreateTarGz creates a .tar.gz archive of the source directory and writes it
// to a file at the target path. If s is non-nil, a generated taito.spec JSON
// file is injected at the root of the archive.
func CreateTarGz(source string, target string, s *spec.TaitoSpec) (retErr error) {
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := tarfile.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("close tar file: %w", cerr)
		}
	}()

	return writeTarGz(source, tarfile, s)
}

// writeTarGz compresses the source directory into a tar.gz stream and writes
// it to w. This is the shared implementation used by both CreateTarGz (file
// output) and CreateOCILayout (in-memory blob). If s is non-nil, a generated
// taito.spec file is written as the first entry in the archive.
func writeTarGz(source string, w io.Writer, s *spec.TaitoSpec) (retErr error) {
	gw := gzip.NewWriter(w)
	defer func() {
		if cerr := gw.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("close gzip writer: %w", cerr)
		}
	}()

	tw := tar.NewWriter(gw)
	defer func() {
		if cerr := tw.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("close tar writer: %w", cerr)
		}
	}()

	source = filepath.Clean(source)

	if s != nil {
		if err := injectSpec(tw, filepath.Base(source), s); err != nil {
			return err
		}
	}

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if s != nil && isRootSpec(source, path, info) {
			return nil
		}
		return addWalkEntry(tw, source, path, info)
	})
}

// injectSpec writes a generated taito.spec JSON file as the first entry in
// the tar archive under the given baseName directory.
func injectSpec(tw *tar.Writer, baseName string, s *spec.TaitoSpec) error {
	specJSON, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	header := &tar.Header{
		Name:    filepath.ToSlash(filepath.Join(baseName, "taito.spec")),
		Size:    int64(len(specJSON)),
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = tw.Write(specJSON)
	return err
}

// isRootSpec reports whether path is the root-level taito.spec inside source.
// Child taito.spec files (in bundle subdirectories) return false.
func isRootSpec(source, path string, info os.FileInfo) bool {
	if info.IsDir() || info.Name() != "taito.spec" {
		return false
	}
	rel, err := filepath.Rel(source, path)
	return err == nil && rel == "taito.spec"
}

// addWalkEntry adds a single filepath.Walk entry (directory or regular file)
// to the tar writer. Paths are made relative to the source's parent directory
// so the archive contains the root folder itself.
func addWalkEntry(tw *tar.Writer, source, path string, info os.FileInfo) error {
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(filepath.Dir(source), path)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(relPath)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(tw, file)
	return err
}
