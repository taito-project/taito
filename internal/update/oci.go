package update

import (
	"context"
	"fmt"

	"github.com/taito-project/taito/internal/registry"
	"oras.land/oras-go/v2/registry/remote"
)

// OCICheckResult holds the outcome of checking an OCI registry for updates.
type OCICheckResult struct {
	LatestVersion string
	HasUpdate     bool
	AllTags       []string
}

// newOCIResult constructs an OCICheckResult.
func newOCIResult(latestVersion string, hasUpdate bool, allTags []string) *OCICheckResult {
	return &OCICheckResult{
		LatestVersion: latestVersion,
		HasUpdate:     hasUpdate,
		AllTags:       allTags,
	}
}

// semverFallback tries to find the latest semver tag as a fallback.
// Returns (result, true) if a semver tag was found, or (nil, false) otherwise.
func semverFallback(allTags []string) (*OCICheckResult, bool) {
	latest, found := LatestSemverTag(allTags)
	if !found {
		return nil, false
	}
	return newOCIResult(latest.Original, true, allTags), true
}

// CheckOCI queries an OCI registry for available tags and determines whether
// a newer version exists compared to currentVersion.
//
// Strategy:
//   - If currentVersion is a valid semver, list all tags, parse them as semver,
//     and check if a newer one exists.
//   - If currentVersion is NOT a valid semver (e.g. "latest"), resolve the
//     "latest" tag digest and report an update if the reference differs.
func CheckOCI(reference, currentVersion string) (*OCICheckResult, error) {
	repo, err := registry.NewRepository(reference)
	if err != nil {
		return nil, registry.WrapAuthError(reference, fmt.Errorf("connect to registry: %w", err))
	}

	ctx := context.Background()

	// Collect all tags from the repository.
	var allTags []string
	err = repo.Tags(ctx, "", func(tags []string) error {
		allTags = append(allTags, tags...)
		return nil
	})
	if err != nil {
		return nil, registry.WrapAuthError(reference, fmt.Errorf("list tags: %w", err))
	}

	if len(allTags) == 0 {
		return newOCIResult(currentVersion, false, allTags), nil
	}

	// If the current version is semver, do semver comparison.
	if currentSV, ok := ParseSemver(currentVersion); ok {
		latest, found := LatestSemverTag(allTags)
		if !found {
			return newOCIResult(currentVersion, false, allTags), nil
		}
		return newOCIResult(latest.Original, latest.IsNewerThan(currentSV), allTags), nil
	}

	// Non-semver current version — check digests.
	return checkOCIDigest(ctx, repo, reference, currentVersion, allTags)
}

func checkOCIDigest(ctx context.Context, repo *remote.Repository, reference, currentVersion string, allTags []string) (*OCICheckResult, error) {
	currentTag := registry.TagFromReference(reference)
	currentDesc, err := repo.Resolve(ctx, currentTag)
	if err != nil {
		if r, ok := semverFallback(allTags); ok {
			return r, nil
		}
		return nil, registry.WrapAuthError(reference, fmt.Errorf("resolve current tag %q: %w", currentTag, err))
	}

	latestDesc, err := repo.Resolve(ctx, "latest")
	if err != nil {
		err = registry.WrapAuthError(reference, err)
		if registry.IsAuthError(err) {
			return nil, err
		}
		if r, ok := semverFallback(allTags); ok {
			return r, nil
		}
		return newOCIResult(currentVersion, false, allTags), nil
	}

	return newOCIResult("latest", currentDesc.Digest != latestDesc.Digest, allTags), nil
}
