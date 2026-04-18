package archive

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/taito-project/taito/internal/spec"
)

func TestCreateOCILayout(t *testing.T) {
	srcDir := createTestSourceTree(t, map[string]string{
		"hello.txt":       "Hello, Taito!",
		"assets/logo.png": "fake image data",
	})
	targetDir := filepath.Join(t.TempDir(), "test-skill-oci")

	if err := CreateOCILayout(srcDir, targetDir, "latest", nil); err != nil {
		t.Fatalf("CreateOCILayout failed: %v", err)
	}

	ociLayout := readOCIImageLayout(t, targetDir)
	if ociLayout.Version != "1.0.0" {
		t.Errorf("Expected oci-layout version 1.0.0, got %s", ociLayout.Version)
	}

	index := readOCIIndex(t, targetDir)
	if len(index.Manifests) == 0 {
		t.Fatal("Expected at least one manifest in index.json")
	}

	blobsDir := filepath.Join(targetDir, "blobs", "sha256")
	if info, err := os.Stat(blobsDir); err != nil || !info.IsDir() {
		t.Fatalf("Expected blobs/sha256 directory at %s", blobsDir)
	}

	manifest := readOCIManifestByDescriptor(t, targetDir, index.Manifests[0])

	if manifest.ArtifactType != "" {
		t.Errorf("Expected no artifactType in manifest body (v1.0 compat), got %s", manifest.ArtifactType)
	}

	if manifest.Config.MediaType != TaitoConfigMediaType {
		t.Errorf("Expected config media type %s, got %s", TaitoConfigMediaType, manifest.Config.MediaType)
	}

	if len(manifest.Layers) != 1 {
		t.Fatalf("Expected 1 layer, got %d", len(manifest.Layers))
	}

	layerDesc := manifest.Layers[0]
	if layerDesc.MediaType != TaitoLayerMediaType {
		t.Errorf("Expected layer media type %s, got %s", TaitoLayerMediaType, layerDesc.MediaType)
	}

	layerFiles := readLayerFiles(t, targetDir, layerDesc)
	assertLayerContainsFileSuffix(t, layerFiles, "hello.txt", "Hello, Taito!")
	assertLayerContainsFileSuffix(t, layerFiles, "assets/logo.png", "fake image data")
}

func TestCreateOCILayoutWithTag(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	targetDir := filepath.Join(t.TempDir(), "tagged-skill-oci")

	tag := "v1.0.0"
	if err := CreateOCILayout(srcDir, targetDir, tag, nil); err != nil {
		t.Fatalf("CreateOCILayout with tag failed: %v", err)
	}

	// Verify the tag appears in index.json annotations.
	indexBytes, err := os.ReadFile(filepath.Join(targetDir, "index.json"))
	if err != nil {
		t.Fatalf("Failed to read index.json: %v", err)
	}

	var index ocispec.Index
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatalf("Failed to parse index.json: %v", err)
	}

	if len(index.Manifests) == 0 {
		t.Fatal("Expected at least one manifest in index.json")
	}

	desc := index.Manifests[0]
	if desc.Annotations[ocispec.AnnotationRefName] != tag {
		t.Errorf("Expected tag %q in manifest annotations, got %q",
			tag, desc.Annotations[ocispec.AnnotationRefName])
	}
}

func TestCreateOCILayoutWithSpec(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	s := &spec.TaitoSpec{
		Type:        spec.TypeSkill,
		Name:        "git-commit-helper",
		Version:     "2.0.0",
		Description: "Helps write conventional commits",
		License:     "MIT",
		Source:      "github.com/taito-project/git-commit-helper",
		Author:      &spec.Author{Name: "Taito"},
	}

	targetDir := filepath.Join(t.TempDir(), "spec-oci")
	tag := "git-commit-helper:2.0.0"

	if err := CreateOCILayout(srcDir, targetDir, tag, s); err != nil {
		t.Fatalf("CreateOCILayout with spec failed: %v", err)
	}

	index := readOCIIndex(t, targetDir)
	manifest := readOCIManifestByDescriptor(t, targetDir, index.Manifests[0])

	// Verify spec metadata in manifest annotations.
	ann := manifest.Annotations
	assertAnnotation(t, ann, "dev.taito.spec.type", "skill")
	assertAnnotation(t, ann, "dev.taito.spec.name", "git-commit-helper")
	assertAnnotation(t, ann, ocispec.AnnotationVersion, "2.0.0")
	assertAnnotation(t, ann, ocispec.AnnotationDescription, "Helps write conventional commits")
	assertAnnotation(t, ann, "dev.taito.spec.license", "MIT")
	assertAnnotation(t, ann, ocispec.AnnotationAuthors, "Taito")
	assertAnnotation(t, ann, ocispec.AnnotationSource, "github.com/taito-project/git-commit-helper")

	// Verify the generated taito.spec is inside the layer.
	layerFiles := readLayerFiles(t, targetDir, manifest.Layers[0])
	assertLayerContainsSpec(t, layerFiles, "git-commit-helper", "2.0.0")
}

func TestCreateOCILayoutOverwrite(t *testing.T) {
	// Verify that packaging to an existing OCI layout directory succeeds
	// (the stale directory is cleaned up automatically).
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	targetDir := filepath.Join(t.TempDir(), "overwrite-oci")

	// First run: create the OCI layout.
	if err := CreateOCILayout(srcDir, targetDir, "v1", nil); err != nil {
		t.Fatalf("First CreateOCILayout failed: %v", err)
	}

	// Verify the first run produced a valid layout.
	if _, err := os.Stat(filepath.Join(targetDir, "oci-layout")); err != nil {
		t.Fatalf("Expected oci-layout after first run: %v", err)
	}

	// Second run: overwrite with different content and tag.
	s := &spec.TaitoSpec{
		Type:    spec.TypeSkill,
		Name:    "overwrite-test",
		Version: "2.0.0",
	}
	if err := CreateOCILayout(srcDir, targetDir, "v2", s); err != nil {
		t.Fatalf("Second CreateOCILayout (overwrite) failed: %v", err)
	}

	// The second run should produce a clean layout with only the v2 manifest.
	indexBytes, err := os.ReadFile(filepath.Join(targetDir, "index.json"))
	if err != nil {
		t.Fatalf("Failed to read index.json after overwrite: %v", err)
	}

	var index ocispec.Index
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatalf("Failed to parse index.json: %v", err)
	}
	if len(index.Manifests) != 1 {
		t.Errorf("Expected exactly 1 manifest after overwrite, got %d", len(index.Manifests))
	}

	// Verify it's the v2 manifest with spec annotations.
	manifestBlobPath := filepath.Join(targetDir, "blobs", "sha256", index.Manifests[0].Digest.Encoded())
	manifestBytes, err := os.ReadFile(manifestBlobPath)
	if err != nil {
		t.Fatalf("Failed to read manifest blob: %v", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	assertAnnotation(t, manifest.Annotations, "dev.taito.spec.name", "overwrite-test")
	assertAnnotation(t, manifest.Annotations, ocispec.AnnotationVersion, "2.0.0")
}

func assertAnnotation(t *testing.T, ann map[string]string, key, want string) {
	t.Helper()
	got, ok := ann[key]
	if !ok {
		t.Errorf("Missing annotation %q (want %q)", key, want)
		return
	}
	if got != want {
		t.Errorf("Annotation %q = %q, want %q", key, got, want)
	}
}

func createTestSourceTree(t *testing.T, files map[string]string) string {
	t.Helper()

	srcDir := t.TempDir()
	for relPath, content := range files {
		fullPath := filepath.Join(srcDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", relPath, err)
		}
	}

	return srcDir
}

func readOCIImageLayout(t *testing.T, targetDir string) ocispec.ImageLayout {
	t.Helper()

	layoutPath := filepath.Join(targetDir, "oci-layout")
	layoutBytes, err := os.ReadFile(layoutPath)
	if err != nil {
		t.Fatalf("Expected oci-layout file at %s: %v", layoutPath, err)
	}

	var layout ocispec.ImageLayout
	if err := json.Unmarshal(layoutBytes, &layout); err != nil {
		t.Fatalf("Failed to parse oci-layout: %v", err)
	}

	return layout
}

func readOCIIndex(t *testing.T, targetDir string) ocispec.Index {
	t.Helper()

	indexPath := filepath.Join(targetDir, "index.json")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Expected index.json at %s: %v", indexPath, err)
	}

	var index ocispec.Index
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatalf("Failed to parse index.json: %v", err)
	}

	return index
}

func readOCIManifestByDescriptor(t *testing.T, targetDir string, desc ocispec.Descriptor) ocispec.Manifest {
	t.Helper()

	manifestBlobPath := filepath.Join(targetDir, "blobs", "sha256", desc.Digest.Encoded())
	manifestBytes, err := os.ReadFile(manifestBlobPath)
	if err != nil {
		t.Fatalf("Failed to read manifest blob: %v", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	return manifest
}

func readLayerFiles(t *testing.T, targetDir string, layerDesc ocispec.Descriptor) map[string]string {
	t.Helper()

	layerBlobPath := filepath.Join(targetDir, "blobs", "sha256", layerDesc.Digest.Encoded())
	layerFile, err := os.Open(layerBlobPath)
	if err != nil {
		t.Fatalf("Failed to open layer blob: %v", err)
	}
	defer func() { _ = layerFile.Close() }()

	gzr, err := gzip.NewReader(layerFile)
	if err != nil {
		t.Fatalf("Failed to initialize gzip reader for layer: %v", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	foundFiles := make(map[string]string)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return foundFiles
		}
		if err != nil {
			t.Fatalf("Failed reading tar header: %v", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}

		content, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("Failed to read content of %s: %v", header.Name, err)
		}
		foundFiles[header.Name] = string(content)
	}
}

func assertLayerContainsFileSuffix(t *testing.T, files map[string]string, suffix, wantContent string) {
	t.Helper()

	for name, content := range files {
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		if content != wantContent {
			t.Errorf("%s content mismatch: got %q", suffix, content)
		}
		return
	}

	t.Errorf("Expected to find %s in layer archive", suffix)
}

// assertLayerContainsSpec finds the taito.spec file in the layer, parses it,
// and verifies the name and version fields.
func assertLayerContainsSpec(t *testing.T, files map[string]string, wantName, wantVersion string) {
	t.Helper()

	for name, content := range files {
		if !strings.HasSuffix(name, "taito.spec") {
			continue
		}
		var parsed spec.TaitoSpec
		if err := json.Unmarshal([]byte(content), &parsed); err != nil {
			t.Fatalf("Failed to parse embedded taito.spec: %v", err)
		}
		if parsed.Name != wantName {
			t.Errorf("embedded spec name = %q, want %q", parsed.Name, wantName)
		}
		if parsed.Version != wantVersion {
			t.Errorf("embedded spec version = %q, want %q", parsed.Version, wantVersion)
		}
		return
	}

	t.Error("Expected to find taito.spec in the OCI layer archive")
}
