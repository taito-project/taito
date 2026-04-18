package tarutil

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EntryMatchFunc is a predicate that decides whether a tar entry matches.
// The name argument is already normalised to forward slashes.
type EntryMatchFunc func(name string, header *tar.Header) bool

// WithTarGz opens a tar.gz file and calls fn with the tar.Reader.
// Resource cleanup (file + gzip reader) is handled automatically with
// proper error propagation via named returns.
func WithTarGz(tarGzPath string, fn func(*tar.Reader) error) (retErr error) {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer func() {
		if cerr := gr.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	return fn(tar.NewReader(gr))
}

// FindEntry iterates over tar entries and returns the header and a reader
// for the first entry where match returns true. The name passed to match
// is normalised to forward slashes via filepath.ToSlash.
//
// If no entry matches, it returns a "not found" error wrapping io.EOF.
func FindEntry(tr *tar.Reader, match EntryMatchFunc) (*tar.Header, io.Reader, error) {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil, nil, fmt.Errorf("entry not found in archive")
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read tar: %w", err)
		}

		name := filepath.ToSlash(header.Name)
		if match(name, header) {
			return header, tr, nil
		}
	}
}
