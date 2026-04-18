package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/taito-project/taito/internal/github"
	"github.com/taito-project/taito/internal/httputil"
)

// gitHubTag represents a single tag from the GitHub Tags API.
type gitHubTag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

// GitHubCheckResult holds the outcome of checking GitHub for updates.
type GitHubCheckResult struct {
	LatestVersion string
	HasUpdate     bool
}

// extractTagNames returns the names from a slice of gitHubTag.
func extractTagNames(tags []gitHubTag) []string {
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return names
}

// shortenSHA truncates a SHA to 7 characters.
func shortenSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// CheckGitHub queries the GitHub Tags API to determine whether a newer
// version exists compared to currentVersion.
//
// Strategy:
//   - If currentVersion is a valid semver tag (e.g. "v4.1.0"): list all tags,
//     parse as semver, check if a higher version exists.
//   - If currentVersion looks like a commit SHA (short hex): fetch tags and
//     check if the latest tag's commit SHA differs. If no tags exist, fetch the
//     latest commit on the default branch.
func CheckGitHub(owner, repo, currentVersion string) (*GitHubCheckResult, error) {
	tags, err := fetchGitHubTags(owner, repo)
	if err != nil {
		return nil, err
	}

	tagNames := extractTagNames(tags)

	// If current version is semver, compare against tag semvers.
	if currentSV, ok := ParseSemver(currentVersion); ok {
		return checkGitHubSemver(currentSV, tagNames)
	}

	// Current version is a commit SHA.
	if len(tags) > 0 {
		return checkGitHubSHA(tags[0], tagNames, currentVersion)
	}

	// No tags at all — check latest commit on default branch.
	return checkGitHubNoTags(owner, repo, currentVersion)
}

func checkGitHubSemver(currentSV Semver, tagNames []string) (*GitHubCheckResult, error) {
	latest, found := LatestSemverTag(tagNames)
	if !found {
		return &GitHubCheckResult{
			LatestVersion: currentSV.Original,
			HasUpdate:     false,
		}, nil
	}
	return &GitHubCheckResult{
		LatestVersion: latest.Original,
		HasUpdate:     latest.IsNewerThan(currentSV),
	}, nil
}

func checkGitHubSHA(latestTag gitHubTag, tagNames []string, currentVersion string) (*GitHubCheckResult, error) {
	if latest, found := LatestSemverTag(tagNames); found {
		return &GitHubCheckResult{
			LatestVersion: latest.Original,
			HasUpdate:     true,
		}, nil
	}

	latestSHA := shortenSHA(latestTag.Commit.SHA)
	return &GitHubCheckResult{
		LatestVersion: latestSHA,
		HasUpdate:     latestSHA != currentVersion,
	}, nil
}

func checkGitHubNoTags(owner, repo, currentVersion string) (*GitHubCheckResult, error) {
	latestSHA, err := fetchLatestCommitSHA(owner, repo)
	if err != nil {
		return nil, fmt.Errorf("fetch latest commit: %w", err)
	}

	short := shortenSHA(latestSHA)
	return &GitHubCheckResult{
		LatestVersion: short,
		HasUpdate:     short != currentVersion,
	}, nil
}

// fetchGitHubTags fetches tags from the GitHub API. Returns up to 100 tags
// (first page), sorted by the GitHub API's default ordering.
func fetchGitHubTags(owner, repo string) ([]gitHubTag, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/tags?per_page=100", github.APIBase, owner, repo)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", github.AcceptHeader)

	resp, err := httputil.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch tags: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository %s/%s not found", owner, repo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d for %s/%s tags", resp.StatusCode, owner, repo)
	}

	var tags []gitHubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("parse tags response: %w", err)
	}
	return tags, nil
}

// fetchLatestCommitSHA fetches the latest commit SHA from the default branch.
func fetchLatestCommitSHA(owner, repo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/commits?per_page=1", github.APIBase, owner, repo)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", github.AcceptHeader)

	resp, err := httputil.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch commits: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d for %s/%s commits", resp.StatusCode, owner, repo)
	}

	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return "", fmt.Errorf("parse commits response: %w", err)
	}
	if len(commits) == 0 {
		return "", fmt.Errorf("no commits found for %s/%s", owner, repo)
	}
	return commits[0].SHA, nil
}
