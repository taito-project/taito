package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/fsutil"
	"github.com/taito-project/taito/internal/install"
	"github.com/taito-project/taito/internal/spec"
)

// ReadSpec reads and parses a taito.spec file directly from a
// filesystem directory (as opposed to from inside an OCI layout's tar.gz layer).
// This is used for GitHub source installs where the repo is extracted to a
// plain directory.
func ReadSpec(dir string) (*spec.TaitoSpec, error) {
	specPath := filepath.Join(dir, "taito.spec")
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read taito.spec: %w", err)
	}

	var s spec.TaitoSpec
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse taito.spec: %w", err)
	}

	return &s, nil
}

// Install installs a skill, agent, or bundle from a plain
// filesystem directory containing a taito.spec file. This is the entry point
// for GitHub source installs.
//
// For skills/agents, the entire directory contents are copied to each tool's
// target directory. For bundles, each include path is resolved and installed
// individually.
func Install(dir string, reference string, s *spec.TaitoSpec, cfg *config.Config) ([]install.InstallResult, error) {
	if len(cfg.Tools) == 0 {
		return nil, fmt.Errorf("no tools configured — run 'taito setup' first")
	}

	if s.Type == spec.TypeBundle {
		return installBundle(dir, reference, s, cfg)
	}

	return installSingle(dir, reference, s, cfg)
}

// installSingle installs a single skill or agent from a directory to
// all configured tools.
func installSingle(dir string, reference string, s *spec.TaitoSpec, cfg *config.Config) ([]install.InstallResult, error) {
	var results []install.InstallResult
	internalRef := install.NormalizeReference(reference)

	for _, tool := range cfg.Tools {
		targetDir, err := install.ResolveToolTarget(tool, s.Type, s.Name, internalRef)
		if err != nil {
			continue
		}

		if err := fsutil.CopyDir(dir, targetDir, fsutil.CopyDirOptions{CleanTarget: true}); err != nil {
			return results, fmt.Errorf("install to %s: %w", tool.Name, err)
		}

		results = append(results, install.InstallResult{
			Name:     s.Name,
			SpecType: s.Type,
			Tool:     install.ToolDisplayName(tool.Name),
			Path:     targetDir,
		})

		if err := install.UpsertEntry(install.InstalledEntry{
			Name:              s.Name,
			SpecType:          s.Type,
			Version:           s.Version,
			Reference:         reference,
			InternalReference: internalRef,
			InstallIn:         []install.InstallLocation{{Tool: tool.Name, Path: targetDir}},
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to track installation of %s: %v\n", s.Name, err)
		}
	}

	return results, nil
}

// installBundle expands a bundle's includes and installs each child
// skill/agent from directory paths.
func installBundle(dir string, reference string, s *spec.TaitoSpec, cfg *config.Config) ([]install.InstallResult, error) {
	if len(s.Includes) == 0 {
		return nil, fmt.Errorf("bundle %q has no includes", s.Name)
	}

	internalRef := install.NormalizeReference(reference)

	bundleID, bundleErr := install.UpsertBundle(install.BundleEntry{
		Name:              s.Name,
		SpecType:          "bundle",
		Version:           s.Version,
		Reference:         reference,
		InternalReference: internalRef,
	})
	if bundleErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to track bundle %s: %v\n", s.Name, bundleErr)
	}

	type childInfo struct {
		spec *spec.TaitoSpec
		dir  string // absolute path to the child directory
	}

	var children []childInfo
	for _, inc := range s.Includes {
		// inc is like "./skills/git-commit-helper/taito.spec"
		clean := filepath.Clean(inc)
		childSpecPath := filepath.Join(dir, clean)
		childDir := filepath.Dir(childSpecPath)

		childSpec, err := ReadSpec(childDir)
		if err != nil {
			return nil, fmt.Errorf("read child spec %q: %w", inc, err)
		}

		children = append(children, childInfo{
			spec: childSpec,
			dir:  childDir,
		})
	}

	var results []install.InstallResult
	for _, child := range children {
		for _, tool := range cfg.Tools {
			targetDir, err := install.ResolveToolTarget(tool, child.spec.Type, child.spec.Name, internalRef)
			if err != nil {
				continue
			}

			if err := fsutil.CopyDir(child.dir, targetDir, fsutil.CopyDirOptions{CleanTarget: true}); err != nil {
				return results, fmt.Errorf("install %s to %s: %w", child.spec.Name, tool.Name, err)
			}

			results = append(results, install.InstallResult{
				Name:     child.spec.Name,
				SpecType: child.spec.Type,
				Tool:     install.ToolDisplayName(tool.Name),
				Path:     targetDir,
			})

			if err := install.UpsertEntry(install.InstalledEntry{
				Name:              child.spec.Name,
				SpecType:          child.spec.Type,
				Version:           s.Version,
				Reference:         reference,
				InternalReference: internalRef,
				BundleID:          bundleID,
				InstallIn:         []install.InstallLocation{{Tool: tool.Name, Path: targetDir}},
			}); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to track installation of %s: %v\n", child.spec.Name, err)
			}
		}
	}

	return results, nil
}
