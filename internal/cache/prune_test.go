package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{2411724, "2.3 MB"},
		{1073741824, "1.0 GB"},
		{1181116006, "1.1 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestItemWord(t *testing.T) {
	if got := ItemWord(1); got != "item" {
		t.Errorf("ItemWord(1) = %q, want %q", got, "item")
	}
	if got := ItemWord(0); got != "items" {
		t.Errorf("ItemWord(0) = %q, want %q", got, "items")
	}
	if got := ItemWord(3); got != "items" {
		t.Errorf("ItemWord(3) = %q, want %q", got, "items")
	}
}

func TestDisplayName(t *testing.T) {
	dirEntry := PruneEntry{Name: "devops-agent-1.0.0-oci", IsDir: true, Size: 100}
	if got := DisplayName(dirEntry); got != "devops-agent-1.0.0-oci/" {
		t.Errorf("DisplayName(dir) = %q, want trailing /", got)
	}

	fileEntry := PruneEntry{Name: "git-helper-0.2.0.tar.gz", IsDir: false, Size: 100}
	if got := DisplayName(fileEntry); got != "git-helper-0.2.0.tar.gz" {
		t.Errorf("DisplayName(file) = %q, want no trailing /", got)
	}
}

func TestDirSize(t *testing.T) {
	tmp := t.TempDir()

	subDir := filepath.Join(tmp, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmp, "file1"), make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2"), make([]byte, 200), 0644); err != nil {
		t.Fatal(err)
	}

	size, err := DirSize(tmp)
	if err != nil {
		t.Fatalf("DirSize error: %v", err)
	}
	if size != 300 {
		t.Errorf("DirSize = %d, want 300", size)
	}
}

func TestScanPackagesNonExistentDir(t *testing.T) {
	entries, err := ScanPackages("/nonexistent/path/packages")
	if err != nil {
		t.Fatalf("ScanPackages error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty slice for nonexistent dir, got %d entries", len(entries))
	}
}

func TestScanPackagesEmptyDir(t *testing.T) {
	tmp := t.TempDir()
	packagesDir := filepath.Join(tmp, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatal(err)
	}

	entries, err := ScanPackages(packagesDir)
	if err != nil {
		t.Fatalf("ScanPackages error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty slice for empty dir, got %d entries", len(entries))
	}
}

func TestScanPackagesWithItems(t *testing.T) {
	tmp := t.TempDir()
	packagesDir := filepath.Join(tmp, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(packagesDir, "git-helper-0.2.0.tar.gz")
	if err := os.WriteFile(tarFile, make([]byte, 1024), 0644); err != nil {
		t.Fatal(err)
	}

	ociDir := filepath.Join(packagesDir, "devops-agent-1.0.0-oci")
	if err := os.MkdirAll(ociDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ociDir, "index.json"), make([]byte, 512), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ScanPackages(packagesDir)
	if err != nil {
		t.Fatalf("ScanPackages error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	names := map[string]PruneEntry{}
	for _, e := range entries {
		names[e.Name] = e
	}

	tar, ok := names["git-helper-0.2.0.tar.gz"]
	if !ok {
		t.Fatal("expected tar.gz entry not found")
	}
	if tar.IsDir {
		t.Error("tar.gz entry should not be a directory")
	}
	if tar.Size != 1024 {
		t.Errorf("tar.gz size = %d, want 1024", tar.Size)
	}

	oci, ok := names["devops-agent-1.0.0-oci"]
	if !ok {
		t.Fatal("expected OCI entry not found")
	}
	if !oci.IsDir {
		t.Error("OCI entry should be a directory")
	}
	if oci.Size != 512 {
		t.Errorf("OCI dir size = %d, want 512", oci.Size)
	}
}

func TestRemovePackages(t *testing.T) {
	tmp := t.TempDir()
	packagesDir := filepath.Join(tmp, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(packagesDir, "helper.tar.gz")
	if err := os.WriteFile(tarFile, make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}

	ociDir := filepath.Join(packagesDir, "agent-oci")
	if err := os.MkdirAll(ociDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ociDir, "blob"), make([]byte, 50), 0644); err != nil {
		t.Fatal(err)
	}

	entries := []PruneEntry{
		{Name: "helper.tar.gz", IsDir: false, Size: 100},
		{Name: "agent-oci", IsDir: true, Size: 50},
	}

	if err := RemovePackages(packagesDir, entries); err != nil {
		t.Fatalf("RemovePackages error: %v", err)
	}

	remaining, err := os.ReadDir(packagesDir)
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 remaining items, got %d", len(remaining))
	}

	if _, err := os.Stat(packagesDir); err != nil {
		t.Errorf("packages directory should still exist: %v", err)
	}
}

func TestDryRunDoesNotRemove(t *testing.T) {
	tmp := t.TempDir()
	packagesDir := filepath.Join(tmp, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(packagesDir, "test.tar.gz")
	if err := os.WriteFile(tarFile, make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ScanPackages(packagesDir)
	if err != nil {
		t.Fatalf("ScanPackages error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// Do NOT call RemovePackages — this is what dry-run does.

	if _, err := os.Stat(tarFile); err != nil {
		t.Errorf("file should still exist after dry-run scan: %v", err)
	}
}
