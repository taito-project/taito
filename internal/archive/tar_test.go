package archive

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/taito-project/taito/internal/spec"
)

func TestCreateTarGz(t *testing.T) {
	// 1. Setup: Create a temporary source directory with some files
	srcDir := t.TempDir()

	file1Path := filepath.Join(srcDir, "hello.txt")
	err := os.WriteFile(file1Path, []byte("Hello, Taito!"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	subDir := filepath.Join(srcDir, "assets")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	file2Path := filepath.Join(subDir, "logo.png")
	err = os.WriteFile(file2Path, []byte("fake image data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// 2. Define the target output file
	targetDir := t.TempDir()
	targetFile := filepath.Join(targetDir, "test-skill.tar.gz")

	// 3. Execute with nil spec (backwards-compatible behavior)
	err = CreateTarGz(srcDir, targetFile, nil)
	if err != nil {
		t.Fatalf("CreateTarGz failed: %v", err)
	}

	// 4. Verification: Check if the file exists
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Fatalf("Expected tar.gz file %s was not created", targetFile)
	}

	// 5. Deep Verification: Open the archive and check its contents
	foundFiles := readTarGzFiles(t, targetFile)

	baseName := filepath.Base(srcDir)
	expectedFile1 := filepath.ToSlash(filepath.Join(baseName, "hello.txt"))
	expectedFile2 := filepath.ToSlash(filepath.Join(baseName, "assets", "logo.png"))

	if _, ok := foundFiles[expectedFile1]; !ok {
		t.Errorf("Expected to find %s in archive, but didn't", expectedFile1)
	}
	if _, ok := foundFiles[expectedFile2]; !ok {
		t.Errorf("Expected to find %s in archive, but didn't", expectedFile2)
	}
}

func TestCreateTarGzWithSpec(t *testing.T) {
	srcDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	// Write an existing taito.spec that should be replaced by the generated one.
	if err := os.WriteFile(filepath.Join(srcDir, "taito.spec"), []byte(`{"type":"skill","name":"old-name"}`), 0644); err != nil {
		t.Fatalf("Failed to create existing taito.spec: %v", err)
	}

	s := &spec.TaitoSpec{
		Type:    spec.TypeSkill,
		Name:    "git-commit-helper",
		Version: "1.0.0",
	}

	targetFile := filepath.Join(t.TempDir(), "test.tar.gz")

	if err := CreateTarGz(srcDir, targetFile, s); err != nil {
		t.Fatalf("CreateTarGz with spec failed: %v", err)
	}

	foundFiles := readTarGzFiles(t, targetFile)
	baseName := filepath.Base(srcDir)

	// The generated taito.spec should be in the archive.
	specPath := filepath.ToSlash(filepath.Join(baseName, "taito.spec"))
	specContent, ok := foundFiles[specPath]
	if !ok {
		t.Fatal("Expected to find taito.spec in archive")
	}

	// It should contain the generated spec, not the original file.
	var parsed spec.TaitoSpec
	if err := json.Unmarshal([]byte(specContent), &parsed); err != nil {
		t.Fatalf("Failed to parse embedded taito.spec: %v", err)
	}
	if parsed.Name != "git-commit-helper" {
		t.Errorf("embedded spec name = %q, want %q", parsed.Name, "git-commit-helper")
	}
	if parsed.Version != "1.0.0" {
		t.Errorf("embedded spec version = %q, want %q", parsed.Version, "1.0.0")
	}
	if parsed.Type != spec.TypeSkill {
		t.Errorf("embedded spec type = %q, want %q", parsed.Type, spec.TypeSkill)
	}

	// The original source files should still be present.
	mainPath := filepath.ToSlash(filepath.Join(baseName, "main.go"))
	if _, ok := foundFiles[mainPath]; !ok {
		t.Error("Expected to find main.go in archive")
	}
}

// readTarGzFiles opens a tar.gz file and returns a map of path -> content for
// all regular files in the archive.
func readTarGzFiles(t *testing.T, path string) map[string]string {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Failed to open tar.gz file: %v", err)
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("Failed to initialize gzip reader: %v", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	files := make(map[string]string)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed reading tar header: %v", err)
		}
		if header.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("Failed to read content of %s: %v", header.Name, err)
			}
			files[header.Name] = string(content)
		}
	}

	return files
}
