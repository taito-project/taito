package registry

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/oci"

	"github.com/taito-project/taito/internal/archive"
	"github.com/taito-project/taito/internal/spec"
)

// buildTestOCILayout creates a minimal OCI layout in a temp dir using the
// real archive.CreateOCILayout function so that the manifest structure
// matches production. Returns the store and temp path.
func buildTestOCILayout(t *testing.T, s *spec.TaitoSpec, tag string) (*oci.Store, string) {
	t.Helper()

	// Create a source directory with a dummy file.
	srcDir := t.TempDir()
	writeTestFile(t, srcDir, "hello.txt", "hello world")

	// Create the OCI layout.
	targetDir := t.TempDir() + "/oci"
	if err := archive.CreateOCILayout(srcDir, targetDir, tag, s); err != nil {
		t.Fatalf("CreateOCILayout: %v", err)
	}

	store, err := oci.New(targetDir)
	if err != nil {
		t.Fatalf("oci.New: %v", err)
	}
	return store, targetDir
}

// buildCustomOCILayout creates an OCI layout with custom annotations, using
// the same v1.0 manifest structure as production (standard config media type,
// no artifactType in manifest body). For testing validation edge cases that
// can't be produced by the normal CreateOCILayout.
func buildCustomOCILayout(t *testing.T, annotations map[string]string, tag string) (*oci.Store, string) {
	t.Helper()
	ctx := context.Background()

	targetDir := t.TempDir() + "/oci"
	store, err := oci.New(targetDir)
	if err != nil {
		t.Fatalf("oci.New: %v", err)
	}
	store.AutoSaveIndex = true

	// Push a dummy layer.
	layerBytes := []byte("fake layer content")
	layerDesc, err := oras.PushBytes(ctx, store, archive.TaitoLayerMediaType, layerBytes)
	if err != nil {
		t.Fatalf("PushBytes: %v", err)
	}

	// Push empty config with standard media type.
	configBytes := []byte("{}")
	configDesc := content.NewDescriptorFromBytes(archive.TaitoConfigMediaType, configBytes)
	if err := store.Push(ctx, configDesc, bytes.NewReader(configBytes)); err != nil {
		t.Fatalf("push config: %v", err)
	}

	// Pack v1.0 manifest with explicit ConfigDescriptor.
	packOpts := oras.PackManifestOptions{
		ConfigDescriptor:    &configDesc,
		Layers:              []ocispec.Descriptor{layerDesc},
		ManifestAnnotations: annotations,
	}
	manifestDesc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_0, "", packOpts)
	if err != nil {
		t.Fatalf("PackManifest: %v", err)
	}

	manifestDesc.ArtifactType = archive.TaitoArtifactType

	if err := store.Tag(ctx, manifestDesc, tag); err != nil {
		t.Fatalf("Tag: %v", err)
	}

	return store, targetDir
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := dir + "/" + name
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile(%s): %v", path, err)
	}
}

// --- tagFromReference tests ---

func TestTagFromReference(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ghcr.io/org/name:v1.0.0", "v1.0.0"},
		{"ghcr.io/org/name:latest", "latest"},
		{"ghcr.io/org/name", "latest"},
		{"localhost:5000/myskill:1.0.0", "1.0.0"},
		{"localhost:5000/myskill", "latest"},
		{"registry.example.com/org/repo:tag", "tag"},
		{"ghcr.io/org/name@sha256:abc123", "sha256:abc123"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := TagFromReference(tc.input)
			if got != tc.want {
				t.Errorf("tagFromReference(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- NewRepository tests ---

func TestNewRepositoryValid(t *testing.T) {
	repo, err := NewRepository("ghcr.io/org/my-skill:1.0.0")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repo")
	}
}

func TestNewRepositoryInvalidReference(t *testing.T) {
	_, err := NewRepository("")
	if err == nil {
		t.Error("expected error for empty reference")
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unauthorized", err: errors.New("401 unauthorized"), want: true},
		{name: "expired token", err: errors.New("token expired while fetching manifest"), want: true},
		{name: "denied", err: errors.New("pull: denied"), want: true},
		{name: "other error", err: errors.New("dial tcp timeout"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsAuthError(tc.err)
			if got != tc.want {
				t.Fatalf("IsAuthError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestWrapAuthError(t *testing.T) {
	err := WrapAuthError("ghcr.io/org/pkg:v1.0.0", errors.New("401 unauthorized"))
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if !contains(err.Error(), "taito login ghcr.io") {
		t.Fatalf("expected login guidance, got: %v", err)
	}
	if !contains(err.Error(), "missing or expired") {
		t.Fatalf("expected expired credential hint, got: %v", err)
	}
}

func TestWrapAuthErrorLeavesNonAuthErrorUntouched(t *testing.T) {
	orig := errors.New("dial tcp timeout")
	wrapped := WrapAuthError("ghcr.io/org/pkg:v1.0.0", orig)
	if wrapped != orig {
		t.Fatal("expected non-auth error to be returned unchanged")
	}
}

// --- ValidateTaitoArtifact tests ---

func TestValidateValidSkill(t *testing.T) {
	tag := "test-skill:1.0.0"
	store, _ := buildCustomOCILayout(t, map[string]string{
		"dev.taito.spec.type": "skill",
		"dev.taito.spec.name": "my-skill",
	}, tag)

	ctx := context.Background()
	specType, warnings, err := ValidateTaitoArtifact(ctx, store, tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if specType != "skill" {
		t.Errorf("specType = %q, want %q", specType, "skill")
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestValidateValidAgent(t *testing.T) {
	tag := "test-agent:1.0.0"
	store, _ := buildCustomOCILayout(t, map[string]string{
		"dev.taito.spec.type": "agent",
		"dev.taito.spec.name": "my-agent",
	}, tag)

	ctx := context.Background()
	specType, _, err := ValidateTaitoArtifact(ctx, store, tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if specType != "agent" {
		t.Errorf("specType = %q, want %q", specType, "agent")
	}
}

func TestValidateValidBundle(t *testing.T) {
	tag := "test-bundle:1.0.0"
	store, _ := buildCustomOCILayout(t, map[string]string{
		"dev.taito.spec.type": "bundle",
		"dev.taito.spec.name": "my-bundle",
	}, tag)

	ctx := context.Background()
	specType, _, err := ValidateTaitoArtifact(ctx, store, tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if specType != "bundle" {
		t.Errorf("specType = %q, want %q", specType, "bundle")
	}
}

func TestValidateMissingSpecType(t *testing.T) {
	tag := "no-type:1.0.0"
	store, _ := buildCustomOCILayout(t, map[string]string{
		"dev.taito.spec.name": "my-skill",
	}, tag)

	ctx := context.Background()
	_, _, err := ValidateTaitoArtifact(ctx, store, tag)
	if err == nil {
		t.Fatal("expected error for missing spec type")
	}
	if !contains(err.Error(), "missing dev.taito.spec.type") {
		t.Errorf("error should mention missing annotation, got: %v", err)
	}
}

func TestValidateUnknownSpecType(t *testing.T) {
	tag := "bad-type:1.0.0"
	store, _ := buildCustomOCILayout(t, map[string]string{
		"dev.taito.spec.type": "foobar",
		"dev.taito.spec.name": "my-skill",
	}, tag)

	ctx := context.Background()
	_, _, err := ValidateTaitoArtifact(ctx, store, tag)
	if err == nil {
		t.Fatal("expected error for unknown spec type")
	}
	if !contains(err.Error(), "unknown spec type") {
		t.Errorf("error should mention unknown spec type, got: %v", err)
	}
}

func TestValidateEmptySpecType(t *testing.T) {
	tag := "empty-type:1.0.0"
	store, _ := buildCustomOCILayout(t, map[string]string{
		"dev.taito.spec.type": "",
		"dev.taito.spec.name": "my-skill",
	}, tag)

	ctx := context.Background()
	_, _, err := ValidateTaitoArtifact(ctx, store, tag)
	if err == nil {
		t.Fatal("expected error for empty spec type")
	}
}

func TestValidateNoAnnotations(t *testing.T) {
	tag := "no-annot:1.0.0"
	store, _ := buildCustomOCILayout(t, nil, tag)

	ctx := context.Background()
	_, _, err := ValidateTaitoArtifact(ctx, store, tag)
	if err == nil {
		t.Fatal("expected error for no taito annotations")
	}
	if !contains(err.Error(), "not a valid taito artifact") {
		t.Errorf("error should mention 'not a valid taito artifact', got: %v", err)
	}
}

func TestValidateMissingName(t *testing.T) {
	tag := "no-name:1.0.0"
	store, _ := buildCustomOCILayout(t, map[string]string{
		"dev.taito.spec.type": "skill",
	}, tag)

	ctx := context.Background()
	specType, warnings, err := ValidateTaitoArtifact(ctx, store, tag)
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if specType != "skill" {
		t.Errorf("specType = %q, want %q", specType, "skill")
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !contains(warnings[0], "dev.taito.spec.name") {
		t.Errorf("warning should mention dev.taito.spec.name, got: %s", warnings[0])
	}
}

func TestValidateWithRealOCILayout(t *testing.T) {
	// Test against an OCI layout created by the real CreateOCILayout function.
	s := &spec.TaitoSpec{
		Type:    "agent",
		Name:    "devops-agent",
		Version: "1.0.0",
	}

	tag := "devops-agent:1.0.0"
	store, _ := buildTestOCILayout(t, s, tag)

	// CreateOCILayout stores the manifest under the short tag ("1.0.0"),
	// not the full reference. Use the short tag to resolve.
	ctx := context.Background()
	specType, warnings, err := ValidateTaitoArtifact(ctx, store, "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if specType != "agent" {
		t.Errorf("specType = %q, want %q", specType, "agent")
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
