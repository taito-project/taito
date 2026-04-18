package install

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/spec"
	"github.com/taito-project/taito/internal/tarutil"
	"oras.land/oras-go/v2/content/oci"
)

// InstallResult holds the outcome of installing a single skill/agent into a
// single tool directory.
type InstallResult struct {
	Name     string
	SpecType string
	Tool     string // tool display name
	Path     string // absolute install path
}

// ReadSpecFromLayout opens an OCI layout directory, reads the layer blob, and
// extracts the taito.spec from inside the tar.gz. This is the canonical way to
// get metadata from a packaged artifact — never from the manifest annotations.
func ReadSpecFromLayout(layoutPath string) (*spec.TaitoSpec, error) {
	layerPath, err := ResolveLayerBlobPath(layoutPath)
	if err != nil {
		return nil, err
	}

	return ReadSpecFromTarGz(layerPath)
}

// Install extracts a skill or agent from an OCI layout into all configured
// tool directories. For bundles, it expands the includes and installs each
// child individually.
//
// Returns the list of successful installations and the reference used for
// tracking (may be empty for local-path installs).
func Install(layoutPath string, reference string, s *spec.TaitoSpec, cfg *config.Config) ([]InstallResult, error) {
	if len(cfg.Tools) == 0 {
		return nil, fmt.Errorf("no tools configured — run 'taito setup' first")
	}

	if s.Type == spec.TypeBundle {
		return installBundle(layoutPath, reference, s, cfg)
	}

	return installSingle(layoutPath, reference, s, cfg)
}

// installSingle installs a single skill or agent to all configured tools.
func installSingle(layoutPath string, reference string, s *spec.TaitoSpec, cfg *config.Config) ([]InstallResult, error) {
	layerPath, err := ResolveLayerBlobPath(layoutPath)
	if err != nil {
		return nil, err
	}

	var results []InstallResult
	internalRef := NormalizeReference(reference)

	for _, tool := range cfg.Tools {
		targetDir, err := ResolveToolTarget(tool, s.Type, s.Name, internalRef)
		if err != nil {
			// Skip tools where we can't resolve the path.
			continue
		}

		if err := extractTarGzToDir(layerPath, "", targetDir); err != nil {
			return results, fmt.Errorf("install to %s: %w", tool.Name, err)
		}

		results = append(results, InstallResult{
			Name:     s.Name,
			SpecType: s.Type,
			Tool:     ToolDisplayName(tool.Name),
			Path:     targetDir,
		})

		// Track installation.
		if err := UpsertEntry(InstalledEntry{
			Name:              s.Name,
			SpecType:          s.Type,
			Version:           s.Version,
			Reference:         reference,
			InternalReference: internalRef,
			InstallIn:         []InstallLocation{{Tool: tool.Name, Path: targetDir}},
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to track installation of %s: %v\n", s.Name, err)
		}
	}

	return results, nil
}

// installBundle expands the bundle's includes and installs each child
// skill/agent to all configured tools.
func installBundle(layoutPath string, reference string, s *spec.TaitoSpec, cfg *config.Config) ([]InstallResult, error) {
	if len(s.Includes) == 0 {
		return nil, fmt.Errorf("bundle %q has no includes", s.Name)
	}

	internalRef := NormalizeReference(reference)

	bundleID, bundleErr := UpsertBundle(BundleEntry{
		Name:              s.Name,
		SpecType:          "bundle",
		Version:           s.Version,
		Reference:         reference,
		InternalReference: internalRef,
	})
	if bundleErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to track bundle %s: %v\n", s.Name, bundleErr)
	}

	layerPath, err := ResolveLayerBlobPath(layoutPath)
	if err != nil {
		return nil, err
	}

	archRoot, err := DiscoverArchiveRoot(layerPath)
	if err != nil {
		return nil, fmt.Errorf("discover archive root: %w", err)
	}

	children, err := resolveChildSpecs(layerPath, archRoot, s.Includes)
	if err != nil {
		return nil, err
	}

	// Install each child to each tool.
	var results []InstallResult
	for _, child := range children {
		for _, tool := range cfg.Tools {
			targetDir, err := ResolveToolTarget(tool, child.spec.Type, child.spec.Name, internalRef)
			if err != nil {
				continue
			}

			if err := extractTarGzSubdirToDir(layerPath, child.archDir, targetDir); err != nil {
				return results, fmt.Errorf("install %s to %s: %w", child.spec.Name, tool.Name, err)
			}

			results = append(results, InstallResult{
				Name:     child.spec.Name,
				SpecType: child.spec.Type,
				Tool:     ToolDisplayName(tool.Name),
				Path:     targetDir,
			})

			if err := UpsertEntry(InstalledEntry{
				Name:              child.spec.Name,
				SpecType:          child.spec.Type,
				Version:           s.Version,
				Reference:         reference,
				InternalReference: internalRef,
				BundleID:          bundleID,
				InstallIn:         []InstallLocation{{Tool: tool.Name, Path: targetDir}},
			}); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to track installation of %s: %v\n", child.spec.Name, err)
			}
		}
	}

	return results, nil
}

// childInfo holds a parsed child spec and its archive directory prefix.
type childInfo struct {
	spec    *spec.TaitoSpec
	archDir string
}

// resolveChildSpecs reads all child specs from a tar.gz archive based on the
// bundle's include paths.
func resolveChildSpecs(layerPath, archRoot string, includes []string) ([]childInfo, error) {
	var children []childInfo
	for _, inc := range includes {
		clean := filepath.Clean(inc)
		childSpecRel := strings.TrimPrefix(clean, "./")
		childDirRel := filepath.Dir(childSpecRel)

		archSpecPath := filepath.ToSlash(filepath.Join(archRoot, childSpecRel))
		childSpec, err := ReadSpecEntryFromTarGz(layerPath, archSpecPath)
		if err != nil {
			return nil, fmt.Errorf("read child spec %q: %w", inc, err)
		}

		children = append(children, childInfo{
			spec:    childSpec,
			archDir: filepath.ToSlash(filepath.Join(archRoot, childDirRel)),
		})
	}
	return children, nil
}

// ToolDisplayName returns the display name for a tool, falling back to the
// raw tool name if no known tool matches.
func ToolDisplayName(toolName string) string {
	if kt := FindKnownTool(toolName); kt != nil {
		return kt.DisplayName
	}
	return toolName
}

// ResolveToolTarget returns the absolute install path for a skill/agent in
// a given tool.
func ResolveToolTarget(tool config.ToolConfig, specType, name, internalReference string) (string, error) {
	var base string
	var err error

	switch specType {
	case spec.TypeSkill:
		base, err = tool.SkillsDir()
	case spec.TypeAgent:
		base, err = tool.AgentsDir()
	default:
		return "", fmt.Errorf("unsupported spec type %q for install", specType)
	}
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(internalReference))
	shortHash := hex.EncodeToString(hash[:])[:5]
	dirName := fmt.Sprintf("%s-taito-%s", name, shortHash)

	return filepath.Join(base, dirName), nil
}

// resolveLayerBlobPath reads the OCI layout's index.json and manifest to find
// the absolute path to the first layer blob.
func ResolveLayerBlobPath(layoutPath string) (string, error) {
	ctx := context.Background()

	store, err := oci.New(layoutPath)
	if err != nil {
		return "", fmt.Errorf("open OCI layout: %w", err)
	}

	// Read index.json to get the manifest descriptor.
	indexData, err := os.ReadFile(filepath.Join(layoutPath, "index.json"))
	if err != nil {
		return "", fmt.Errorf("read index.json: %w", err)
	}

	var index ocispec.Index
	if err := json.Unmarshal(indexData, &index); err != nil {
		return "", fmt.Errorf("parse index.json: %w", err)
	}
	if len(index.Manifests) == 0 {
		return "", fmt.Errorf("no manifests in OCI layout")
	}

	// Fetch and parse the manifest.
	manifestDesc := index.Manifests[0]
	rc, err := store.Fetch(ctx, manifestDesc)
	if err != nil {
		return "", fmt.Errorf("fetch manifest: %w", err)
	}
	defer func() { _ = rc.Close() }()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
		return "", fmt.Errorf("decode manifest: %w", err)
	}
	if len(manifest.Layers) == 0 {
		return "", fmt.Errorf("no layers in manifest")
	}

	layerDesc := manifest.Layers[0]
	return filepath.Join(layoutPath, "blobs", "sha256", layerDesc.Digest.Encoded()), nil
}

// ReadSpecFromTarGz opens a tar.gz file and finds the first root-level
// taito.spec entry, returning the parsed spec.
func ReadSpecFromTarGz(tarGzPath string) (*spec.TaitoSpec, error) {
	var s *spec.TaitoSpec
	err := tarutil.WithTarGz(tarGzPath, func(tr *tar.Reader) error {
		_, r, err := tarutil.FindEntry(tr, isRootSpecEntry)
		if err != nil {
			return fmt.Errorf("no taito.spec found in artifact layer")
		}
		s, err = parseSpecFromReader(r)
		return err
	})
	return s, err
}

// isRootSpecEntry matches a root-level taito.spec (e.g. "bundle-repo/taito.spec"
// has exactly two path components and ends with taito.spec).
func isRootSpecEntry(name string, h *tar.Header) bool {
	parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
	return len(parts) == 2 && parts[1] == "taito.spec" && h.Typeflag == tar.TypeReg
}

// parseSpecFromReader reads all bytes from r and unmarshals them as a TaitoSpec.
func parseSpecFromReader(r io.Reader) (*spec.TaitoSpec, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read taito.spec: %w", err)
	}
	var s spec.TaitoSpec
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse taito.spec: %w", err)
	}
	return &s, nil
}

// ReadSpecEntryFromTarGz reads a specific taito.spec entry from a tar.gz
// archive by its full archive path (e.g., "bundle-repo/skills/helper/taito.spec").
func ReadSpecEntryFromTarGz(tarGzPath, entryPath string) (*spec.TaitoSpec, error) {
	var s *spec.TaitoSpec
	err := tarutil.WithTarGz(tarGzPath, func(tr *tar.Reader) error {
		match := func(name string, h *tar.Header) bool {
			return name == entryPath && h.Typeflag == tar.TypeReg
		}
		_, r, err := tarutil.FindEntry(tr, match)
		if err != nil {
			return fmt.Errorf("entry %q not found in archive", entryPath)
		}
		s, err = parseSpecFromReader(r)
		return err
	})
	return s, err
}

// DiscoverArchiveRoot reads the first entry of a tar.gz to determine the root
// directory name (e.g., "bundle-repo" from "bundle-repo/taito.spec").
func DiscoverArchiveRoot(tarGzPath string) (string, error) {
	var root string
	err := tarutil.WithTarGz(tarGzPath, func(tr *tar.Reader) error {
		header, err := tr.Next()
		if err != nil {
			return fmt.Errorf("read first tar entry: %w", err)
		}

		name := filepath.ToSlash(strings.TrimSuffix(header.Name, "/"))
		parts := strings.SplitN(name, "/", 2)
		root = parts[0]
		return nil
	})
	return root, err
}

// extractTarGzToDir extracts a tar.gz archive to a target directory, stripping
// the archive's root directory prefix.
func extractTarGzToDir(tarGzPath, prefix, targetDir string) error {
	return tarutil.Extract(tarGzPath, targetDir, tarutil.ExtractOptions{
		StripComponents: 1,
		CleanTarget:     true,
	})
}

// extractTarGzSubdirToDir extracts only files under a specific subdirectory
// of a tar.gz archive to a target directory. The subdirectory prefix is
// stripped from output paths.
func extractTarGzSubdirToDir(tarGzPath, archSubdir, targetDir string) error {
	return tarutil.Extract(tarGzPath, targetDir, tarutil.ExtractOptions{
		SubdirFilter: archSubdir,
		CleanTarget:  true,
	})
}

// IsOCILayout checks whether the given path looks like an OCI Image Layout
// directory (i.e. contains an oci-layout file).
func IsOCILayout(path string) bool {
	_, err := os.Stat(filepath.Join(path, "oci-layout"))
	return err == nil
}

// FindKnownTool looks up a tool name in the known tools list.
func FindKnownTool(name string) *config.KnownTool {
	for _, kt := range config.KnownTools() {
		if kt.Name == name {
			return &kt
		}
	}
	return nil
}

// NormalizeReference removes version tags from a reference string.
// For OCI (e.g., ghcr.io/org/repo:1.0.0) it removes the tag.
// For GitHub (e.g., github.com/org/repo@v1) it removes the ref.
// Local paths are returned as-is.
func NormalizeReference(ref string) string {
	if IsLocalPath(ref) || ref == "" {
		return ref
	}
	if idx := strings.LastIndex(ref, "@"); idx != -1 {
		return ref[:idx]
	}
	if lastSlash := strings.LastIndex(ref, "/"); lastSlash != -1 {
		if idx := strings.LastIndex(ref[lastSlash:], ":"); idx != -1 {
			return ref[:lastSlash+idx]
		}
	} else if idx := strings.LastIndex(ref, ":"); idx != -1 {
		return ref[:idx]
	}
	return ref
}

// IsLocalPath returns true if the source looks like a filesystem path
// (starts with ., /, or ~).
func IsLocalPath(source string) bool {
	if source == "" {
		return false
	}
	return source[0] == '.' || source[0] == '/' || source[0] == '~'
}
