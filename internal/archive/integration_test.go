package archive

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/taito-project/taito/internal/spec"
)

// examplesDir returns the absolute path to the taito.spec/examples directory.
// It walks up from the test file's location to find the project root.
func examplesDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	// filename is internal/archive/integration_test.go
	// project root is two levels up
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	dir := filepath.Join(projectRoot, "taito.spec", "examples", "bundle-repo")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("examples directory not found at %s: %v", dir, err)
	}
	return dir
}

// TestIntegrationOCIPackageFromSpec exercises the full pipeline: load a real
// taito.spec from the examples directory, package as OCI, and verify the
// output contains a valid OCI layout with correct annotations.
func TestIntegrationOCIPackageFromSpec(t *testing.T) {
	examples := examplesDir(t)
	agentDir := filepath.Join(examples, "agents", "devops-agent")

	// Load and validate the spec (same steps the CLI command performs).
	specFile := filepath.Join(agentDir, "taito.spec")
	s, err := spec.Load(specFile)
	if err != nil {
		t.Fatalf("spec.Load(%q) failed: %v", specFile, err)
	}
	warnings, err := spec.Validate(s)
	if err != nil {
		t.Fatalf("spec.Validate failed: %v", err)
	}
	// The example spec is clean — no warnings expected.
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}

	// Package into an OCI layout.
	targetDir := filepath.Join(t.TempDir(), "devops-agent-1.0.0-oci")
	tag := "devops-agent:1.0.0"
	if err := CreateOCILayout(agentDir, targetDir, tag, s); err != nil {
		t.Fatalf("CreateOCILayout failed: %v", err)
	}

	// Verify the OCI layout marker file.
	ociLayoutBytes, err := os.ReadFile(filepath.Join(targetDir, "oci-layout"))
	if err != nil {
		t.Fatalf("missing oci-layout file: %v", err)
	}
	var ociLayout ocispec.ImageLayout
	if err := json.Unmarshal(ociLayoutBytes, &ociLayout); err != nil {
		t.Fatalf("invalid oci-layout: %v", err)
	}
	if ociLayout.Version != "1.0.0" {
		t.Errorf("oci-layout version = %q, want 1.0.0", ociLayout.Version)
	}

	// Verify index.json.
	manifest := readOCIManifest(t, targetDir)

	// Verify manifest uses v1.0 compatible format (no artifactType in body).
	if manifest.ArtifactType != "" {
		t.Errorf("expected no artifactType in manifest body, got %q", manifest.ArtifactType)
	}
	if manifest.Config.MediaType != TaitoConfigMediaType {
		t.Errorf("config mediaType = %q, want %q", manifest.Config.MediaType, TaitoConfigMediaType)
	}

	// Verify annotations match the spec.
	ann := manifest.Annotations
	assertAnnotation(t, ann, "dev.taito.spec.type", "agent")
	assertAnnotation(t, ann, "dev.taito.spec.name", "devops-agent")
	assertAnnotation(t, ann, ocispec.AnnotationVersion, "1.0.0")
	assertAnnotation(t, ann, ocispec.AnnotationDescription, "A DevOps agent that combines commit helpers and documentation generation")
	assertAnnotation(t, ann, "dev.taito.spec.license", "Apache-2.0")

	// Verify the layer contains the expected files.
	layerFiles := readOCILayerFiles(t, targetDir, manifest.Layers[0])
	foundSpec := false
	foundSkillMD := false
	for name := range layerFiles {
		if strings.HasSuffix(name, "taito.spec") {
			foundSpec = true
		}
		if strings.HasSuffix(name, "SKILL.md") {
			foundSkillMD = true
		}
	}
	if !foundSpec {
		t.Error("expected taito.spec in OCI layer")
	}
	if !foundSkillMD {
		t.Error("expected SKILL.md in OCI layer")
	}
}

// TestIntegrationTarGzPackageFromSpec exercises the full pipeline for tar.gz
// format: load spec, package, verify the archive.
func TestIntegrationTarGzPackageFromSpec(t *testing.T) {
	examples := examplesDir(t)
	skillDir := filepath.Join(examples, "skills", "git-commit-helper")

	specFile := filepath.Join(skillDir, "taito.spec")
	s, err := spec.Load(specFile)
	if err != nil {
		t.Fatalf("spec.Load failed: %v", err)
	}
	if _, err := spec.Validate(s); err != nil {
		t.Fatalf("spec.Validate failed: %v", err)
	}

	targetFile := filepath.Join(t.TempDir(), "git-commit-helper-latest.tar.gz")
	if err := CreateTarGz(skillDir, targetFile, s); err != nil {
		t.Fatalf("CreateTarGz failed: %v", err)
	}

	// Verify the archive exists and contains the generated taito.spec.
	files := readTarGzFiles(t, targetFile)

	var specContent string
	foundSkillMD := false
	for name, content := range files {
		if strings.HasSuffix(name, "taito.spec") {
			specContent = content
		}
		if strings.HasSuffix(name, "SKILL.md") {
			foundSkillMD = true
		}
	}

	if specContent == "" {
		t.Fatal("expected taito.spec in tar.gz archive")
	}
	if !foundSkillMD {
		t.Error("expected SKILL.md in tar.gz archive")
	}

	// The embedded spec should contain the generated data, not the original file.
	var parsed spec.TaitoSpec
	if err := json.Unmarshal([]byte(specContent), &parsed); err != nil {
		t.Fatalf("failed to parse embedded taito.spec: %v", err)
	}
	if parsed.Name != "git-commit-helper" {
		t.Errorf("embedded spec name = %q, want %q", parsed.Name, "git-commit-helper")
	}
	if parsed.Type != spec.TypeSkill {
		t.Errorf("embedded spec type = %q, want %q", parsed.Type, spec.TypeSkill)
	}
}

// TestIntegrationOCIOverwrite verifies that packaging to the same target
// directory twice succeeds — the stale directory from the first run is
// cleaned up automatically.
func TestIntegrationOCIOverwrite(t *testing.T) {
	examples := examplesDir(t)
	agentDir := filepath.Join(examples, "agents", "devops-agent")

	specFile := filepath.Join(agentDir, "taito.spec")
	s, err := spec.Load(specFile)
	if err != nil {
		t.Fatalf("spec.Load failed: %v", err)
	}

	targetDir := filepath.Join(t.TempDir(), "overwrite-test-oci")
	tag := "devops-agent:1.0.0"

	// First run.
	if err := CreateOCILayout(agentDir, targetDir, tag, s); err != nil {
		t.Fatalf("First CreateOCILayout failed: %v", err)
	}

	// Second run to the same target — must not fail.
	if err := CreateOCILayout(agentDir, targetDir, tag, s); err != nil {
		t.Fatalf("Second CreateOCILayout (overwrite) failed: %v", err)
	}

	// Verify the result is a clean single-manifest layout.
	indexBytes, err := os.ReadFile(filepath.Join(targetDir, "index.json"))
	if err != nil {
		t.Fatalf("Failed to read index.json: %v", err)
	}
	var index ocispec.Index
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatalf("Failed to parse index.json: %v", err)
	}
	if len(index.Manifests) != 1 {
		t.Errorf("expected 1 manifest after overwrite, got %d", len(index.Manifests))
	}
}

// TestIntegrationOCIAnnotations verifies that all spec fields are correctly
// mapped to OCI manifest annotations when a fully-populated spec is used.
func TestIntegrationOCIAnnotations(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Test Skill"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	s := &spec.TaitoSpec{
		TaitoVersion: "0.1.0",
		Type:         spec.TypeSkill,
		Name:         "full-annotation-test",
		Version:      "3.1.4",
		Description:  "A skill with every field set",
		License:      "MIT",
		Source:       "github.com/taito-project/full-annotation-test",
		Author:       &spec.Author{Name: "Taito Team"},
		Keywords:     []string{"test", "annotations"},
	}

	targetDir := filepath.Join(t.TempDir(), "annotations-oci")
	tag := "full-annotation-test:3.1.4"

	if err := CreateOCILayout(srcDir, targetDir, tag, s); err != nil {
		t.Fatalf("CreateOCILayout failed: %v", err)
	}

	manifest := readOCIManifest(t, targetDir)
	ann := manifest.Annotations

	// Custom taito annotations.
	assertAnnotation(t, ann, "dev.taito.spec.type", "skill")
	assertAnnotation(t, ann, "dev.taito.spec.name", "full-annotation-test")
	assertAnnotation(t, ann, "dev.taito.spec.license", "MIT")

	// Standard OCI annotations.
	assertAnnotation(t, ann, ocispec.AnnotationVersion, "3.1.4")
	assertAnnotation(t, ann, ocispec.AnnotationDescription, "A skill with every field set")
	assertAnnotation(t, ann, ocispec.AnnotationAuthors, "Taito Team")
	assertAnnotation(t, ann, ocispec.AnnotationSource, "github.com/taito-project/full-annotation-test")
	assertAnnotation(t, ann, ocispec.AnnotationTitle, "3.1.4")
}

// TestIntegrationTarGzSpecReplacement verifies that when the source directory
// contains an existing taito.spec file, the archive contains the generated
// spec (from the TaitoSpec struct), not the original file.
func TestIntegrationTarGzSpecReplacement(t *testing.T) {
	srcDir := t.TempDir()

	// Write an "old" spec that should be replaced.
	oldSpec := `{"type":"skill","name":"old-name","version":"0.0.1"}`
	if err := os.WriteFile(filepath.Join(srcDir, "taito.spec"), []byte(oldSpec), 0644); err != nil {
		t.Fatalf("failed to write old taito.spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Old Skill"), 0644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// The "new" spec that should end up in the archive.
	s := &spec.TaitoSpec{
		Type:    spec.TypeAgent,
		Name:    "new-name",
		Version: "2.0.0",
	}

	targetFile := filepath.Join(t.TempDir(), "replacement-test.tar.gz")
	if err := CreateTarGz(srcDir, targetFile, s); err != nil {
		t.Fatalf("CreateTarGz failed: %v", err)
	}

	files := readTarGzFiles(t, targetFile)

	// Find the embedded taito.spec.
	var specContent string
	for name, content := range files {
		if strings.HasSuffix(name, "taito.spec") {
			specContent = content
			break
		}
	}
	if specContent == "" {
		t.Fatal("expected taito.spec in archive")
	}

	var parsed spec.TaitoSpec
	if err := json.Unmarshal([]byte(specContent), &parsed); err != nil {
		t.Fatalf("failed to parse embedded taito.spec: %v", err)
	}

	// Must be the new spec, not the old one.
	if parsed.Name != "new-name" {
		t.Errorf("embedded spec name = %q, want %q (old spec was not replaced)", parsed.Name, "new-name")
	}
	if parsed.Version != "2.0.0" {
		t.Errorf("embedded spec version = %q, want %q", parsed.Version, "2.0.0")
	}
	if parsed.Type != spec.TypeAgent {
		t.Errorf("embedded spec type = %q, want %q", parsed.Type, spec.TypeAgent)
	}
}

// --- Helpers ---

// readOCIManifest reads the first manifest from an OCI layout directory.
func readOCIManifest(t *testing.T, targetDir string) ocispec.Manifest {
	t.Helper()

	indexBytes, err := os.ReadFile(filepath.Join(targetDir, "index.json"))
	if err != nil {
		t.Fatalf("failed to read index.json: %v", err)
	}

	var index ocispec.Index
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatalf("failed to parse index.json: %v", err)
	}
	if len(index.Manifests) == 0 {
		t.Fatal("no manifests in index.json")
	}

	manifestBlobPath := filepath.Join(targetDir, "blobs", "sha256", index.Manifests[0].Digest.Encoded())
	manifestBytes, err := os.ReadFile(manifestBlobPath)
	if err != nil {
		t.Fatalf("failed to read manifest blob: %v", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}
	return manifest
}

// readOCILayerFiles extracts all regular files from an OCI layer blob.
func readOCILayerFiles(t *testing.T, targetDir string, layerDesc ocispec.Descriptor) map[string]string {
	t.Helper()

	layerBlobPath := filepath.Join(targetDir, "blobs", "sha256", layerDesc.Digest.Encoded())
	layerFile, err := os.Open(layerBlobPath)
	if err != nil {
		t.Fatalf("failed to open layer blob: %v", err)
	}
	defer func() { _ = layerFile.Close() }()

	gzr, err := gzip.NewReader(layerFile)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
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
			t.Fatalf("failed reading tar header: %v", err)
		}
		if header.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("failed to read %s: %v", header.Name, err)
			}
			files[header.Name] = string(content)
		}
	}
	return files
}
