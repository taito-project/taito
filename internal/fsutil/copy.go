// Package fsutil provides shared filesystem utilities used across taito's
// internal packages.
package fsutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyDirOptions configures how CopyDir behaves.
type CopyDirOptions struct {
	// CleanTarget removes the destination directory before copying,
	// ensuring a fresh copy with no leftover files.
	CleanTarget bool
}

// CopyDir recursively copies the directory tree from src to dst.
// If opts.CleanTarget is true, dst is removed before copying.
func CopyDir(src, dst string, opts CopyDirOptions) error {
	if opts.CleanTarget {
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("clean target: %w", err)
		}
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("create target: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath, CopyDirOptions{}); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile copies a single file from src to dst, preserving permissions.
func copyFile(src, dst string) (retErr error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if cerr := dstFile.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	_, retErr = io.Copy(dstFile, srcFile)
	return retErr
}
