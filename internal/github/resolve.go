package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/taito-project/taito/ui"
)

// ResolveResult holds the outcome of resolving a GitHub source reference.
type ResolveResult struct {
	SourceDir string              // path to the extracted source directory (or subdirectory)
	Workspace string              // temp directory the caller must clean up
	Warning   string              // non-empty if the repo was missing a taito.spec
	Items     []ui.SelectableItem // populated if taito.spec is missing and skills/ dir exists
	Version   string              // extracted version (commit SHA or tag)
}

// Resolve parses a GitHub source string, downloads the repository tarball,
// extracts it, and ensures a taito.spec exists in the target directory.
// If the root spec is missing it looks for specs in subdirectories first,
// then falls back to guessing skills/ and agents/ directories.
func Resolve(source string) (*ResolveResult, error) {
	ref, err := Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parse GitHub reference: %w", err)
	}

	gitRef, err := ResolveRef(ref.Owner, ref.Repo, ref.Version)
	if err != nil {
		return nil, fmt.Errorf("resolve GitHub ref: %w", err)
	}

	workspace, err := DownloadTarball(ref.Owner, ref.Repo, gitRef)
	if err != nil {
		return nil, fmt.Errorf("download GitHub tarball: %w", err)
	}

	sourceDir, err := resolveSourceDir(workspace, ref.Subdir)
	if err != nil {
		_ = os.RemoveAll(workspace)
		return nil, err
	}

	// Prefer the user-specified tag/ref (e.g. "v2.0.0") when present.
	// Fall back to the commit SHA extracted from the tarball directory
	// only when no explicit version was provided by the user.
	version := ref.Version
	if version == "" {
		version = ExtractVersion(workspace, ref.Owner, ref.Repo)
	}

	result := &ResolveResult{
		SourceDir: sourceDir,
		Workspace: workspace,
		Version:   version,
	}

	if err := ensureSpec(result, ref); err != nil {
		_ = os.RemoveAll(workspace)
		return nil, err
	}

	return result, nil
}

// resolveSourceDir determines the extracted source directory, optionally
// descending into a subdirectory if one was specified in the reference.
func resolveSourceDir(workspace, subdir string) (string, error) {
	sourceDir := SourceDir(workspace)
	if subdir == "" {
		return sourceDir, nil
	}

	sourceDir = filepath.Join(sourceDir, subdir)
	if _, err := os.Stat(sourceDir); err != nil {
		return "", fmt.Errorf("subdirectory %q not found in repository", subdir)
	}
	return sourceDir, nil
}

// ensureSpec makes sure a taito.spec exists in the result's SourceDir.
// If one already exists on disk it does nothing. Otherwise it tries to
// discover specs in subdirectories, and as a last resort synthesises
// fallback specs by guessing the repository layout.
func ensureSpec(result *ResolveResult, ref *Ref) error {
	specPath := filepath.Join(result.SourceDir, "taito.spec")
	if _, err := os.Stat(specPath); err == nil {
		return nil // spec exists, nothing to do
	}

	// Subdirectory target: write a single fallback spec.
	if ref.Subdir != "" {
		return writeSingleFallbackSpec(result.SourceDir, ref, result.Version)
	}

	// Root target: discover existing specs first.
	specs := findSpecs(result.SourceDir)
	if len(specs) > 0 {
		return writeBundleSpec(specPath, InferName(ref), result.Version, specs)
	}

	// No specs found anywhere — guess from directory layout.
	items := discoverItems(result.SourceDir)
	if err := WriteFallbackSpec(result.SourceDir, InferName(ref), result.Version, "bundle", items); err != nil {
		return fmt.Errorf("write fallback spec: %w", err)
	}

	result.Warning = fmt.Sprintf(
		"No taito.spec found in %s/%s — consider adding taito.spec files and opening a pull request!\n\n Continuing on the assumption that the repo contains 'skills' or 'agents' directories.",
		ref.Owner, ref.Repo,
	)
	return nil
}

// writeSingleFallbackSpec writes a fallback spec for a single subdirectory target.
func writeSingleFallbackSpec(sourceDir string, ref *Ref, version string) error {
	itemType := "skill"
	if strings.HasPrefix(ref.Subdir, "agents/") || strings.HasPrefix(ref.Subdir, "agents\\") {
		itemType = "agent"
	}
	if err := WriteFallbackSpec(sourceDir, InferName(ref), version, itemType, nil); err != nil {
		return fmt.Errorf("write fallback spec: %w", err)
	}
	return nil
}

// findSpecs walks the directory tree and returns relative paths of
// directories that contain a taito.spec file (excluding the root).
func findSpecs(rootDir string) []string {
	rootSpec := filepath.Join(rootDir, "taito.spec")
	var specs []string

	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "taito.spec" || path == rootSpec {
			return nil
		}
		if rel, err := filepath.Rel(rootDir, filepath.Dir(path)); err == nil {
			specs = append(specs, rel)
		}
		return nil
	})

	return specs
}

// discoverItems looks for subdirectories inside skills/ and agents/ and
// returns their relative paths (e.g. "skills/coolskill1", "agents/coolagent1").
// If neither directory exists the result is empty.
func discoverItems(sourceDir string) []string {
	var items []string
	for _, dir := range []string{"skills", "agents"} {
		entries, err := os.ReadDir(filepath.Join(sourceDir, dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				items = append(items, filepath.Join(dir, e.Name()))
			}
		}
	}
	return items
}

// writeBundleSpec writes a root bundle taito.spec that includes the given
// relative spec paths.
func writeBundleSpec(specPath, name, version string, specDirs []string) error {
	includes := make([]string, len(specDirs))
	for i, dir := range specDirs {
		includes[i] = fmt.Sprintf("./%s/taito.spec", filepath.ToSlash(dir))
	}

	data, err := json.MarshalIndent(struct {
		Type     string   `json:"type"`
		Name     string   `json:"name"`
		Version  string   `json:"version,omitempty"`
		Includes []string `json:"includes,omitempty"`
	}{
		Type:     "bundle",
		Name:     name,
		Version:  version,
		Includes: includes,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal bundle spec: %w", err)
	}

	return os.WriteFile(specPath, data, 0644)
}

// WriteFallbackSpec writes a minimal taito.spec into dir so downstream
// install logic works without modification. If items is provided,
// it generates a bundle spec at the root and child specs in the selected subdirectories.
func WriteFallbackSpec(dir string, name string, version string, specType string, items []string) error {
	if len(items) > 0 {
		return writeFallbackBundle(dir, name, version, items)
	}
	return writeFallbackSingle(dir, name, version, specType)
}

// writeFallbackBundle writes a bundle spec at dir and child specs for each item.
func writeFallbackBundle(dir, name, version string, items []string) error {
	var includes []string
	for _, item := range items {
		itemDir := filepath.Join(dir, item)
		itemType := "skill"
		if strings.HasPrefix(item, "agents/") || strings.HasPrefix(item, "agents\\") {
			itemType = "agent"
		}
		if err := writeFallbackSingle(itemDir, filepath.Base(item), version, itemType); err != nil {
			return err
		}
		includes = append(includes, fmt.Sprintf("./%s/taito.spec", filepath.ToSlash(item)))
	}

	data, err := json.MarshalIndent(struct {
		Type     string   `json:"type"`
		Name     string   `json:"name"`
		Version  string   `json:"version,omitempty"`
		Includes []string `json:"includes,omitempty"`
	}{
		Type:     "bundle",
		Name:     name,
		Version:  version,
		Includes: includes,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "taito.spec"), data, 0644)
}

// writeFallbackSingle writes a single skill/agent taito.spec into dir.
func writeFallbackSingle(dir, name, version, specType string) error {
	data, err := json.MarshalIndent(struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		Version string `json:"version,omitempty"`
	}{
		Type:    specType,
		Name:    name,
		Version: version,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "taito.spec"), data, 0644)
}

// InferName derives a reasonable package name from the GitHub reference.
// If a subdirectory is specified, the last path component is used (e.g.
// "agents/devops" → "devops"). Otherwise the owner/repo name is used.
func InferName(ref *Ref) string {
	if ref.Subdir != "" {
		parts := strings.Split(ref.Subdir, "/")
		return fmt.Sprintf("%s/%s/%s", ref.Owner, ref.Repo, parts[len(parts)-1])
	}
	return fmt.Sprintf("%s/%s", ref.Owner, ref.Repo)
}
