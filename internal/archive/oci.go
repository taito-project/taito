package archive

import (
	"bytes"
	"context"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/taito-project/taito/internal/ociref"
	"github.com/taito-project/taito/internal/spec"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/oci"
)

const (
	// TaitoLayerMediaType is the media type for a taito tar.gz layer in an OCI artifact.
	TaitoLayerMediaType = "application/vnd.taito.layer.v1.tar+gzip"

	// TaitoArtifactType is the artifact type for a packaged taito artifact.
	// This value is stored on the local OCI index descriptor for identification
	// purposes. It does NOT appear in the manifest blob itself, ensuring
	// compatibility with registries that reject custom artifact types (e.g. GitLab).
	TaitoArtifactType = "application/vnd.taito.v1"

	// TaitoConfigMediaType is the config media type used in taito OCI manifests.
	// We use the standard OCI image config type for maximum registry compatibility.
	TaitoConfigMediaType = ocispec.MediaTypeImageConfig

	// AnnotationSpecType is the OCI manifest annotation key for the taito spec type
	// (skill, agent, or bundle).
	AnnotationSpecType = "dev.taito.spec.type"

	// AnnotationSpecName is the OCI manifest annotation key for the taito spec name.
	AnnotationSpecName = "dev.taito.spec.name"

	// AnnotationSpecLicense is the OCI manifest annotation key for the taito spec license.
	AnnotationSpecLicense = "dev.taito.spec.license"
)

// CreateOCILayout packages the source directory into an OCI Image Layout at the
// target path. The source directory is compressed into a tar.gz blob which
// becomes the single layer of the OCI artifact.
//
// The manifest uses OCI Image Manifest v1.0 format with a standard config media
// type (application/vnd.oci.image.config.v1+json) for maximum registry
// compatibility. The artifactType field does NOT appear in the manifest blob
// (only on the local index descriptor), avoiding rejection by registries that
// don't support OCI 1.1 (e.g. GitLab).
//
// If s is non-nil, the spec metadata is written into the OCI manifest annotations
// and a generated taito.spec file is injected into the layer archive.
func CreateOCILayout(source string, target string, tag string, s *spec.TaitoSpec) error {
	ctx := context.Background()

	// Extract the short tag portion from a full OCI reference.
	// e.g. "ghcr.io/org/name:v1.0.0" → "v1.0.0", "ghcr.io/org/name" → "latest".
	shortTag := ociref.TagFromReference(tag)

	// 1. Create the tar.gz content in memory to use as the OCI layer blob.
	var buf bytes.Buffer
	if err := writeTarGz(source, &buf, s); err != nil {
		return err
	}
	layerBytes := buf.Bytes()

	// 2. Remove any existing output to ensure a clean OCI layout.
	// This gives Docker-like "override" behavior on re-runs.
	if err := os.RemoveAll(target); err != nil {
		return err
	}

	// 3. Create an OCI layout store at the target directory.
	store, err := oci.New(target)
	if err != nil {
		return err
	}
	store.AutoSaveIndex = true

	// 4. Push the tar.gz bytes as a layer blob.
	layerDesc, err := oras.PushBytes(ctx, store, TaitoLayerMediaType, layerBytes)
	if err != nil {
		return err
	}

	// 5. Push an empty config blob with the standard OCI image config type.
	// Using the standard type ensures every registry (including GitLab)
	// accepts the manifest.
	configBytes := []byte("{}")
	configDesc := content.NewDescriptorFromBytes(TaitoConfigMediaType, configBytes)
	if err := store.Push(ctx, configDesc, bytes.NewReader(configBytes)); err != nil {
		return err
	}

	// 6. Build manifest annotations from the spec.
	annotations := buildAnnotations(shortTag, s)

	// 7. Pack an OCI v1.0 manifest referencing the config and layer.
	// Using v1.0 with an explicit ConfigDescriptor avoids the OCI 1.1
	// artifactType field in the manifest blob. The artifactType only
	// appears on the local index.json descriptor (set below).
	packOpts := oras.PackManifestOptions{
		ConfigDescriptor:    &configDesc,
		Layers:              []ocispec.Descriptor{layerDesc},
		ManifestAnnotations: annotations,
	}

	manifestDesc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_0, "", packOpts)
	if err != nil {
		return err
	}

	// 8. Override the descriptor's ArtifactType for the local index.json.
	// This is local-only metadata — it does NOT appear in the manifest blob
	// that gets pushed to registries. It allows local tools to quickly
	// identify taito artifacts without parsing the manifest.
	manifestDesc.ArtifactType = TaitoArtifactType

	// 9. Tag the manifest so it appears in index.json.
	if err := store.Tag(ctx, manifestDesc, shortTag); err != nil {
		return err
	}

	return nil
}

// buildAnnotations constructs the OCI manifest annotations from a tag and spec.
func buildAnnotations(shortTag string, s *spec.TaitoSpec) map[string]string {
	annotations := map[string]string{
		ocispec.AnnotationTitle: shortTag,
	}
	if s == nil {
		return annotations
	}

	annotations[AnnotationSpecType] = s.Type
	annotations[AnnotationSpecName] = s.Name
	if s.Version != "" {
		annotations[ocispec.AnnotationVersion] = s.Version
	}
	if s.Description != "" {
		annotations[ocispec.AnnotationDescription] = s.Description
	}
	if s.License != "" {
		annotations[AnnotationSpecLicense] = s.License
	}
	if s.Author != nil && s.Author.Name != "" {
		annotations[ocispec.AnnotationAuthors] = s.Author.Name
	}
	if s.Source != "" {
		annotations[ocispec.AnnotationSource] = s.Source
	}
	return annotations
}
