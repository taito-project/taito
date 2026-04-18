package github

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRefWithVersion(t *testing.T) {
	// When version is provided, no API call should be made.
	ref, err := ResolveRef("owner", "repo", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref != "v1.0.0" {
		t.Errorf("ref = %q, want %q", ref, "v1.0.0")
	}
}

func TestResolveRefDefaultBranch(t *testing.T) {
	// Mock the GitHub API.
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		// Verify the Accept header.
		if accept := r.Header.Get("Accept"); accept != "application/vnd.github+json" {
			t.Errorf("Accept header = %q, want %q", accept, "application/vnd.github+json")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"default_branch": "develop",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	// Override the API URL by using a custom HTTP client that rewrites the URL.
	// Instead, we'll test with a helper that accepts a base URL.
	// For simplicity, we test the real function by temporarily overriding http.DefaultClient.
	origTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{
		base:    server.URL,
		wrapped: http.DefaultTransport,
	}
	defer func() { http.DefaultTransport = origTransport }()

	ref, err := ResolveRef("testowner", "testrepo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref != "develop" {
		t.Errorf("ref = %q, want %q", ref, "develop")
	}
}

func TestResolveRefNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/ghost/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}
	defer func() { http.DefaultTransport = origTransport }()

	_, err := ResolveRef("ghost", "missing", "")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestDownloadTarball(t *testing.T) {
	// Build a minimal tar.gz with a wrapper directory (like GitHub produces).
	tarball := buildTestTarball(t, "testowner-testrepo-abc123", map[string]string{
		"taito.spec": `{"type":"skill","name":"test-skill"}`,
		"SKILL.md":   "# Test Skill",
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/tarball/main", func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("Accept"); accept != "application/vnd.github+json" {
			t.Errorf("Accept header = %q, want %q", accept, "application/vnd.github+json")
		}
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(tarball)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}
	defer func() { http.DefaultTransport = origTransport }()

	workspace, err := DownloadTarball("testowner", "testrepo", "main")
	if err != nil {
		t.Fatalf("DownloadTarball: %v", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()

	// Verify tarball was saved.
	tarballPath := TarballPath(workspace)
	if _, err := os.Stat(tarballPath); err != nil {
		t.Errorf("expected tarball at %s: %v", tarballPath, err)
	}

	// Verify SourceDir discovers the extracted folder.
	sourceDir := SourceDir(workspace)
	if filepath.Base(sourceDir) != "testowner-testrepo-abc123" {
		t.Errorf("SourceDir base = %q, want %q", filepath.Base(sourceDir), "testowner-testrepo-abc123")
	}

	// Verify files are accessible through SourceDir.
	specPath := filepath.Join(sourceDir, "taito.spec")
	if _, err := os.Stat(specPath); err != nil {
		t.Errorf("expected taito.spec at %s: %v", specPath, err)
	}
	mdPath := filepath.Join(sourceDir, "SKILL.md")
	if _, err := os.Stat(mdPath); err != nil {
		t.Errorf("expected SKILL.md at %s: %v", mdPath, err)
	}
}

func TestDownloadTarballNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/ghost/missing/tarball/main", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}
	defer func() { http.DefaultTransport = origTransport }()

	workspace, err := DownloadTarball("ghost", "missing", "main")
	if err == nil {
		_ = os.RemoveAll(workspace)
		t.Fatal("expected error for 404 response")
	}
}

func TestDownloadTarballWithSubdirectories(t *testing.T) {
	// Build a tarball with nested structure (like a real GitHub repo).
	tarball := buildTestTarball(t, "org-monorepo-abc123", map[string]string{
		"taito.spec":               `{"type":"bundle","name":"mono"}`,
		"skills/helper/taito.spec": `{"type":"skill","name":"helper"}`,
		"skills/helper/SKILL.md":   "# Helper",
		"agents/devops/taito.spec": `{"type":"agent","name":"devops"}`,
		"agents/devops/SKILL.md":   "# Devops",
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/monorepo/tarball/v1.0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(tarball)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}
	defer func() { http.DefaultTransport = origTransport }()

	workspace, err := DownloadTarball("org", "monorepo", "v1.0")
	if err != nil {
		t.Fatalf("DownloadTarball: %v", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()

	sourceDir := SourceDir(workspace)

	// Verify the archive root folder name is preserved.
	if filepath.Base(sourceDir) != "org-monorepo-abc123" {
		t.Errorf("SourceDir base = %q, want %q", filepath.Base(sourceDir), "org-monorepo-abc123")
	}

	// Verify nested structure was extracted correctly.
	for _, path := range []string{
		"taito.spec",
		"skills/helper/taito.spec",
		"skills/helper/SKILL.md",
		"agents/devops/taito.spec",
		"agents/devops/SKILL.md",
	} {
		full := filepath.Join(sourceDir, path)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected %s at %s: %v", path, full, err)
		}
	}
}

// --- Test Helpers ---

// rewriteTransport rewrites requests from api.github.com to the test server.
type rewriteTransport struct {
	base    string
	wrapped http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "api.github.com" {
		req = req.Clone(req.Context())
		req.URL.Scheme = "http"
		req.URL.Host = t.base[len("http://"):]
	}
	return t.wrapped.RoundTrip(req)
}

// buildTestTarball creates a tar.gz in memory with a root wrapper directory
// (simulating GitHub's tarball format). Files is a map of relative path to content.
func buildTestTarball(t *testing.T, rootDir string, files map[string]string) []byte {
	t.Helper()

	tmpFile := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Write root directory entry.
	_ = tw.WriteHeader(&tar.Header{
		Name:     rootDir + "/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})

	// Collect and sort paths to ensure parent directories are created first.
	for relPath, content := range files {
		fullPath := rootDir + "/" + relPath

		// Create parent directories if needed.
		dir := filepath.Dir(fullPath)
		if dir != rootDir {
			_ = tw.WriteHeader(&tar.Header{
				Name:     filepath.ToSlash(dir) + "/",
				Typeflag: tar.TypeDir,
				Mode:     0755,
			})
		}

		_ = tw.WriteHeader(&tar.Header{
			Name:     filepath.ToSlash(fullPath),
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
			Mode:     0644,
		})
		_, _ = tw.Write([]byte(content))
	}

	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
