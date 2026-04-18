package tarutil

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// createTestTarGz creates a tar.gz file with the given entries.
// Each entry is a map of path -> content.
func createTestTarGz(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	gw := gzip.NewWriter(f)
	defer func() { _ = gw.Close() }()

	tw := tar.NewWriter(gw)
	defer func() { _ = tw.Close() }()

	for name, content := range entries {
		if err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0644,
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestResolveRelPath_NoOptions(t *testing.T) {
	path, ok := resolveRelPath("foo/bar.txt", ExtractOptions{})
	if !ok || path != "foo/bar.txt" {
		t.Errorf("got (%q, %v), want (foo/bar.txt, true)", path, ok)
	}

	_, ok = resolveRelPath("/", ExtractOptions{})
	if ok {
		t.Error("trailing slash only should be skipped")
	}
}

func TestResolveRelPath_StripComponents(t *testing.T) {
	path, ok := resolveRelPath("root/sub/file.txt", ExtractOptions{StripComponents: 1})
	if !ok || path != "sub/file.txt" {
		t.Errorf("got (%q, %v), want (sub/file.txt, true)", path, ok)
	}

	_, ok = resolveRelPath("root/", ExtractOptions{StripComponents: 1})
	if ok {
		t.Error("stripped to empty should be skipped")
	}
}

func TestResolveRelPath_SubdirFilter(t *testing.T) {
	path, ok := resolveRelPath("bundle/skills/helper/file.txt", ExtractOptions{SubdirFilter: "bundle/skills/helper"})
	if !ok || path != "file.txt" {
		t.Errorf("got (%q, %v), want (file.txt, true)", path, ok)
	}

	_, ok = resolveRelPath("bundle/other/file.txt", ExtractOptions{SubdirFilter: "bundle/skills/helper"})
	if ok {
		t.Error("non-matching prefix should be skipped")
	}
}

func TestIsInsideDir(t *testing.T) {
	if !isInsideDir("/target/sub/file.txt", "/target") {
		t.Error("should be inside")
	}
	if isInsideDir("/other/file.txt", "/target") {
		t.Error("should not be inside")
	}
	// Path traversal
	if isInsideDir("/target/../etc/passwd", "/target") {
		t.Error("path traversal should be rejected")
	}
}

func TestExtract_StripComponents(t *testing.T) {
	tarPath := createTestTarGz(t, map[string]string{
		"root/file.txt":       "hello",
		"root/sub/nested.txt": "world",
	})

	outDir := filepath.Join(t.TempDir(), "out")
	err := Extract(tarPath, outDir, ExtractOptions{StripComponents: 1, CleanTarget: true})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "file.txt"))
	if err != nil {
		t.Fatalf("file.txt not found: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("file.txt content = %q, want hello", data)
	}

	data, err = os.ReadFile(filepath.Join(outDir, "sub", "nested.txt"))
	if err != nil {
		t.Fatalf("sub/nested.txt not found: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("sub/nested.txt content = %q, want world", data)
	}
}

func TestExtract_SubdirFilter(t *testing.T) {
	tarPath := createTestTarGz(t, map[string]string{
		"root/skills/a/file.txt": "included",
		"root/skills/b/file.txt": "excluded",
	})

	outDir := filepath.Join(t.TempDir(), "out")
	err := Extract(tarPath, outDir, ExtractOptions{SubdirFilter: "root/skills/a", CleanTarget: true})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "file.txt"))
	if err != nil {
		t.Fatalf("file.txt not found: %v", err)
	}
	if string(data) != "included" {
		t.Errorf("file.txt content = %q, want included", data)
	}

	// b should not exist
	if _, err := os.Stat(filepath.Join(outDir, "b")); err == nil {
		t.Error("skills/b should not have been extracted")
	}
}

func TestWithTarGz_InvalidFile(t *testing.T) {
	err := WithTarGz("/nonexistent/path.tar.gz", func(tr *tar.Reader) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFindEntry(t *testing.T) {
	tarPath := createTestTarGz(t, map[string]string{
		"root/taito.spec": `{"type":"skill"}`,
		"root/other.txt":  "other",
	})

	err := WithTarGz(tarPath, func(tr *tar.Reader) error {
		_, r, err := FindEntry(tr, func(name string, h *tar.Header) bool {
			return name == "root/taito.spec"
		})
		if err != nil {
			return err
		}
		buf := make([]byte, 100)
		n, _ := r.Read(buf)
		if string(buf[:n]) != `{"type":"skill"}` {
			t.Errorf("unexpected content: %s", buf[:n])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("FindEntry: %v", err)
	}
}

func TestFindEntry_NotFound(t *testing.T) {
	tarPath := createTestTarGz(t, map[string]string{
		"root/other.txt": "other",
	})

	err := WithTarGz(tarPath, func(tr *tar.Reader) error {
		_, _, err := FindEntry(tr, func(name string, h *tar.Header) bool {
			return name == "nonexistent"
		})
		return err
	})
	if err == nil {
		t.Error("expected error for missing entry")
	}
}
