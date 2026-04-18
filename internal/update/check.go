package update

import (
	"fmt"
	"strings"

	"github.com/taito-project/taito/internal/github"
	"github.com/taito-project/taito/internal/install"
)

// UpdateResult holds the outcome of checking a single installed package for updates.
type UpdateResult struct {
	ID              string // installed entry ID
	Name            string
	SpecType        string
	CurrentVersion  string
	LatestVersion   string
	Reference       string // original install reference
	UpdateReference string // reference with the new version tag substituted
	HasUpdate       bool
	IsLocal         bool  // true if installed from a local path (no remote to check)
	Error           error // non-nil if the check failed for this entry
	IsBundleChild   bool
	BundleID        string
}

type checkResult struct {
	latestVersion string
	hasUpdate     bool
	err           error
}

// CheckAll loads the installed index and checks every unique package for updates.
// Bundle children are grouped under their parent — only the bundle's reference
// is checked, not each child individually.
func CheckAll() ([]UpdateResult, error) {
	idx, err := install.LoadInstalled()
	if err != nil {
		return nil, fmt.Errorf("load installed packages: %w", err)
	}

	return checkEntries(idx, "")
}

// CheckByID checks a single installed package (by ID) for updates. If the ID
// matches a bundle, all children are included.
func CheckByID(id string) ([]UpdateResult, error) {
	idx, err := install.LoadInstalled()
	if err != nil {
		return nil, fmt.Errorf("load installed packages: %w", err)
	}

	return checkEntries(idx, id)
}

// checkEntries does the actual checking. If filterID is non-empty, only
// entries matching that ID (or belonging to a bundle with that ID) are checked.
func checkEntries(idx *install.InstalledIndex, filterID string) ([]UpdateResult, error) {
	var results []UpdateResult
	allEntries := append([]install.InstalledEntry{}, idx.Installed.Skills...)
	allEntries = append(allEntries, idx.Installed.Agents...)

	// Build a set of bundle IDs for lookup.
	bundlesByID := make(map[string]install.BundleEntry)
	for _, b := range idx.Installed.Bundles {
		bundlesByID[b.ID] = b
	}

	// Track which references we've already checked to avoid duplicate API calls.
	checkedRefs := make(map[string]*checkResult)

	// Process bundles first.
	for _, b := range idx.Installed.Bundles {
		if filterID != "" && b.ID != filterID {
			continue
		}

		cr := checkReference(checkedRefs, b.Reference, b.Version)
		results = appendBundleResults(results, b, allEntries, cr)
	}

	// Process standalone skills and agents (not part of a bundle).
	for _, e := range allEntries {
		if e.BundleID != "" {
			if r := checkBundleChild(e, bundlesByID, checkedRefs, filterID); r != nil {
				results = append(results, *r)
			}
			continue
		}

		if filterID != "" && e.ID != filterID {
			continue
		}

		results = appendStandaloneResult(results, e, checkReference(checkedRefs, e.Reference, e.Version))
	}

	return results, nil
}

// checkBundleChild checks whether a bundle-child entry should produce an
// UpdateResult when filtered by ID. Returns nil if the entry should be skipped.
func checkBundleChild(e install.InstalledEntry, bundlesByID map[string]install.BundleEntry, checkedRefs map[string]*checkResult, filterID string) *UpdateResult {
	if filterID == "" || filterID != e.ID {
		return nil
	}
	b, ok := bundlesByID[e.BundleID]
	if !ok {
		return nil
	}
	cr := checkReference(checkedRefs, b.Reference, b.Version)
	return &UpdateResult{
		ID:             e.ID,
		Name:           e.Name,
		SpecType:       e.SpecType,
		CurrentVersion: e.Version,
		LatestVersion:  cr.latestVersion,
		Reference:      b.Reference,
		HasUpdate:      cr.hasUpdate,
		IsBundleChild:  true,
		BundleID:       e.BundleID,
		Error:          cr.err,
	}
}

// checkReference performs the version check for a given reference+version,
// caching the result by normalized reference.
func checkReference(checkedRefs map[string]*checkResult, reference, currentVersion string) *checkResult {
	cacheKey := install.NormalizeReference(reference)
	if cached, ok := checkedRefs[cacheKey]; ok {
		return cached
	}

	result := &checkResult{}

	if install.IsLocalPath(reference) || reference == "" {
		result.latestVersion = currentVersion
		result.hasUpdate = false
	} else if github.IsGitHubSource(reference) {
		ref, err := github.Parse(reference)
		if err != nil {
			result.err = fmt.Errorf("parse reference: %w", err)
		} else {
			ghResult, err := CheckGitHub(ref.Owner, ref.Repo, currentVersion)
			if err != nil {
				result.err = err
			} else {
				result.latestVersion = ghResult.LatestVersion
				result.hasUpdate = ghResult.HasUpdate
			}
		}
	} else {
		ociResult, err := CheckOCI(reference, currentVersion)
		if err != nil {
			result.err = err
		} else {
			result.latestVersion = ociResult.LatestVersion
			result.hasUpdate = ociResult.HasUpdate
		}
	}

	checkedRefs[cacheKey] = result
	return result
}

func appendBundleResults(results []UpdateResult, b install.BundleEntry, entries []install.InstalledEntry, cr *checkResult) []UpdateResult {
	bundleResult := UpdateResult{
		ID:             b.ID,
		Name:           b.Name,
		SpecType:       "bundle",
		CurrentVersion: b.Version,
		LatestVersion:  cr.latestVersion,
		Reference:      b.Reference,
		HasUpdate:      cr.hasUpdate,
		IsLocal:        install.IsLocalPath(b.Reference) || b.Reference == "",
		Error:          cr.err,
	}
	if cr.hasUpdate {
		bundleResult.UpdateReference = buildUpdateReference(b.Reference, cr.latestVersion)
	}
	results = append(results, bundleResult)

	for _, e := range entries {
		if e.BundleID != b.ID {
			continue
		}

		results = append(results, UpdateResult{
			ID:             e.ID,
			Name:           e.Name,
			SpecType:       e.SpecType,
			CurrentVersion: e.Version,
			LatestVersion:  cr.latestVersion,
			Reference:      e.Reference,
			HasUpdate:      false,
			IsBundleChild:  true,
			BundleID:       b.ID,
		})
	}

	return results
}

func appendStandaloneResult(results []UpdateResult, e install.InstalledEntry, cr *checkResult) []UpdateResult {
	ur := UpdateResult{
		ID:             e.ID,
		Name:           e.Name,
		SpecType:       e.SpecType,
		CurrentVersion: e.Version,
		LatestVersion:  cr.latestVersion,
		Reference:      e.Reference,
		HasUpdate:      cr.hasUpdate,
		IsLocal:        install.IsLocalPath(e.Reference) || e.Reference == "",
		Error:          cr.err,
	}
	if cr.hasUpdate {
		ur.UpdateReference = buildUpdateReference(e.Reference, cr.latestVersion)
	}

	return append(results, ur)
}

// buildUpdateReference replaces the version in a reference string with
// the new version.
//
// Examples:
//
//	"ghcr.io/org/skill:v1.0.0" + "v2.0.0" → "ghcr.io/org/skill:v2.0.0"
//	"github.com/owner/repo@v1.0.0" + "v2.0.0" → "github.com/owner/repo@v2.0.0"
func buildUpdateReference(reference, newVersion string) string {
	if reference == "" || newVersion == "" {
		return reference
	}

	// GitHub-style: owner/repo@version
	if idx := strings.LastIndex(reference, "@"); idx != -1 {
		return reference[:idx] + "@" + newVersion
	}

	// OCI-style: registry/repo:tag
	internalRef := install.NormalizeReference(reference)
	if internalRef != reference {
		// Had a version tag — replace it.
		return internalRef + ":" + newVersion
	}

	// No version tag at all — append one.
	return reference + ":" + newVersion
}

// UpdatableResults returns only the results that have updates available
// (excluding bundle children, errors, and local installs).
func UpdatableResults(results []UpdateResult) []UpdateResult {
	var updatable []UpdateResult
	for _, r := range results {
		if r.HasUpdate && !r.IsBundleChild && r.Error == nil {
			updatable = append(updatable, r)
		}
	}
	return updatable
}
