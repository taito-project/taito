package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/taito-project/taito/internal/archive"
	"github.com/taito-project/taito/internal/fsutil"
	"github.com/taito-project/taito/internal/ociref"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// NewRepository creates an authenticated remote.Repository for the given OCI
// reference (e.g. "ghcr.io/org/my-skill:1.0.0"). Credentials are loaded from
// the taito credential store.
func NewRepository(reference string) (*remote.Repository, error) {
	repo, err := remote.NewRepository(reference)
	if err != nil {
		return nil, fmt.Errorf("invalid reference %q: %w", reference, err)
	}

	store, err := NewCredentialStore()
	if err != nil {
		return nil, fmt.Errorf("credential store: %w", err)
	}

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(store),
	}

	return repo, nil
}

// Push copies all content from a local OCI layout directory to a remote
// repository. The tag is extracted from the reference. Returns the pushed
// manifest descriptor.
func Push(ctx context.Context, localOCIPath string, reference string) (ocispec.Descriptor, error) {
	// Open the local OCI layout.
	localStore, err := oci.New(localOCIPath)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("open local OCI layout: %w", err)
	}

	// Create authenticated remote repository.
	repo, err := NewRepository(reference)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	// Extract tag from reference.
	tag := TagFromReference(reference)

	// Copy from local store to remote.
	desc, err := oras.Copy(ctx, localStore, tag, repo, tag, oras.DefaultCopyOptions)
	if err != nil {
		return ocispec.Descriptor{}, WrapAuthError(reference, fmt.Errorf("push: %w", err))
	}

	return desc, nil
}

// Pull copies an artifact from a remote repository into a local OCI layout
// directory. The artifact is first pulled to a temporary directory, validated,
// and only moved to the final targetPath on success. If validation fails, the
// pulled data is discarded.
//
// Returns the manifest descriptor and the validated spec type.
func Pull(ctx context.Context, reference string, targetPath string) (desc ocispec.Descriptor, specType string, warnings []string, err error) {
	// Create authenticated remote repository.
	repo, repoErr := NewRepository(reference)
	if repoErr != nil {
		return ocispec.Descriptor{}, "", nil, repoErr
	}

	// Pull to a temporary directory first.
	tmpDir, tmpErr := os.MkdirTemp("", "taito-pull-*")
	if tmpErr != nil {
		return ocispec.Descriptor{}, "", nil, fmt.Errorf("create temp dir: %w", tmpErr)
	}
	// Always clean up temp dir on error paths.
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	tmpStore, storeErr := oci.New(tmpDir)
	if storeErr != nil {
		return ocispec.Descriptor{}, "", nil, fmt.Errorf("create temp OCI store: %w", storeErr)
	}
	tmpStore.AutoSaveIndex = true

	// Extract tag from reference.
	tag := TagFromReference(reference)

	// Copy from remote to temp local store.
	desc, copyErr := oras.Copy(ctx, repo, tag, tmpStore, tag, oras.DefaultCopyOptions)
	if copyErr != nil {
		return ocispec.Descriptor{}, "", nil, WrapAuthError(reference, fmt.Errorf("pull: %w", copyErr))
	}

	// Validate the pulled artifact.
	specType, warnings, valErr := ValidateTaitoArtifact(ctx, tmpStore, tag)
	if valErr != nil {
		return ocispec.Descriptor{}, "", nil, valErr
	}

	// Validation passed — move to final target.
	// Remove any existing target first for clean overwrite.
	if rmErr := os.RemoveAll(targetPath); rmErr != nil {
		return ocispec.Descriptor{}, "", nil, fmt.Errorf("clean target: %w", rmErr)
	}
	if moveErr := moveDir(tmpDir, targetPath); moveErr != nil {
		return ocispec.Descriptor{}, "", nil, fmt.Errorf("move to target: %w", moveErr)
	}

	return desc, specType, warnings, nil
}

// ValidateTaitoArtifact checks that a pulled OCI layout contains a valid taito
// artifact. Returns the spec type on success and any non-fatal warnings.
//
// Hard errors (artifact is discarded):
//   - dev.taito.spec.type annotation missing
//   - dev.taito.spec.type not in {skill, agent, bundle}
//
// Warnings (artifact is kept):
//   - dev.taito.spec.name annotation missing
func ValidateTaitoArtifact(ctx context.Context, store *oci.Store, reference string) (specType string, warnings []string, err error) {
	// Fetch the manifest bytes.
	desc, err := store.Resolve(ctx, reference)
	if err != nil {
		return "", nil, fmt.Errorf("resolve manifest: %w", err)
	}

	rc, err := store.Fetch(ctx, desc)
	if err != nil {
		return "", nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer func() {
		if cerr := rc.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close manifest reader: %w", cerr)
		}
	}()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
		return "", nil, fmt.Errorf("decode manifest: %w", err)
	}

	// Check dev.taito.spec.type annotation.
	// This is the primary identification mechanism for taito artifacts.
	// We do not check artifactType or config.mediaType because:
	// - artifactType is an OCI 1.1 field that some registries strip/reject
	// - config.mediaType uses the standard OCI image config type for compatibility
	annotations := manifest.Annotations
	if annotations == nil {
		return "", nil, fmt.Errorf("not a valid taito artifact: manifest has no annotations")
	}

	specType, ok := annotations[archive.AnnotationSpecType]
	if !ok || specType == "" {
		return "", nil, fmt.Errorf("not a valid taito artifact: missing %s annotation", archive.AnnotationSpecType)
	}

	validTypes := map[string]bool{"skill": true, "agent": true, "bundle": true}
	if !validTypes[specType] {
		return "", nil, fmt.Errorf("not a valid taito artifact: unknown spec type %q", specType)
	}

	// Non-fatal warnings.
	if _, hasName := annotations[archive.AnnotationSpecName]; !hasName {
		warnings = append(warnings, "missing "+archive.AnnotationSpecName+" annotation")
	}

	return specType, warnings, nil
}

// TagFromReference extracts the tag portion from an OCI reference string.
// This delegates to ociref.TagFromReference to avoid import cycles.
func TagFromReference(reference string) string {
	return ociref.TagFromReference(reference)
}

// moveDir moves src to dst. It tries os.Rename first (fast, same-filesystem),
// and falls back to a recursive copy + remove when the rename fails with
// EXDEV (cross-device link), which happens when /tmp and the target are on
// different filesystems.
func moveDir(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Check for cross-device link error.
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) || !errors.Is(linkErr.Err, syscall.EXDEV) {
		return err
	}

	// Fall back to recursive copy.
	if err := fsutil.CopyDir(src, dst, fsutil.CopyDirOptions{}); err != nil {
		return fmt.Errorf("copy fallback: %w", err)
	}
	return os.RemoveAll(src)
}

func registryHostFromReference(reference string) string {
	if reference == "" {
		return "registry"
	}

	if idx := strings.Index(reference, "/"); idx != -1 {
		return reference[:idx]
	}

	return reference
}

// IsAuthError reports whether err looks like a registry authentication failure.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"unauthorized",
		"authentication required",
		"authentication failed",
		"access denied",
		"denied",
		"invalid username/password",
		"invalid token",
		"token expired",
		"expired token",
		"status code 401",
		"401 unauthorized",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}

	return false
}

// WrapAuthError rewrites registry auth failures with clearer login guidance.
func WrapAuthError(reference string, err error) error {
	if !IsAuthError(err) {
		return err
	}

	host := registryHostFromReference(reference)
	return fmt.Errorf(
		"authentication failed for %s: stored credentials may be missing or expired; run 'taito login %s' and retry: %w",
		host,
		host,
		err,
	)
}
