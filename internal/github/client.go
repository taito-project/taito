package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/taito-project/taito/internal/httputil"
)

const (
	// APIBase is the base URL for the GitHub REST API.
	APIBase = "https://api.github.com"

	// AcceptHeader is the Accept header value for the GitHub API.
	AcceptHeader = "application/vnd.github+json"
)

// repoResponse is the minimal subset of the GitHub repo API response we need.
type repoResponse struct {
	DefaultBranch string `json:"default_branch"`
}

// ResolveRef determines the git reference to download.
//
// If version is non-empty, it is returned as-is (0 API requests).
// If version is empty, the GitHub repo API is queried to find the
// default branch (1 API request).
func ResolveRef(owner, repo, version string) (string, error) {
	if version != "" {
		return version, nil
	}

	url := fmt.Sprintf("%s/repos/%s/%s", APIBase, owner, repo)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", AcceptHeader)

	resp, err := httputil.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch repo info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("repository %s/%s not found", owner, repo)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d for %s/%s", resp.StatusCode, owner, repo)
	}

	var result repoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse repo response: %w", err)
	}
	if result.DefaultBranch == "" {
		return "", fmt.Errorf("no default branch found for %s/%s", owner, repo)
	}

	return result.DefaultBranch, nil
}

// DownloadTarball downloads a repository tarball from the GitHub API and
// extracts it into a workspace directory. It returns the workspace path
// containing both the downloaded tarball and the extracted contents.
//
// The workspace layout:
//
//	<workspace>/repo.tar.gz                       — the downloaded tarball
//	<workspace>/<owner>-<repo>-<commitsha>/       — the extracted contents (archive root preserved)
//
// The caller is responsible for cleaning up the workspace directory.
func DownloadTarball(owner, repo, ref string) (workspace string, err error) {
	url := fmt.Sprintf("%s/repos/%s/%s/tarball/%s", APIBase, owner, repo, ref)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", AcceptHeader)

	resp, err := httputil.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download tarball: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("tarball not found for %s/%s at ref %q", owner, repo, ref)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d when downloading tarball for %s/%s", resp.StatusCode, owner, repo)
	}

	// Create workspace directory.
	workspace, err = os.MkdirTemp("", "taito-github-*")
	if err != nil {
		return "", fmt.Errorf("create workspace: %w", err)
	}

	// On error, clean up the workspace.
	defer func() {
		if err != nil {
			_ = os.RemoveAll(workspace)
			workspace = ""
		}
	}()

	// Save tarball to workspace.
	tarballPath := filepath.Join(workspace, "repo.tar.gz")
	f, fErr := os.Create(tarballPath)
	if fErr != nil {
		return "", fmt.Errorf("create tarball file: %w", fErr)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close tarball file: %w", cerr)
		}
	}()
	if _, copyErr := io.Copy(f, resp.Body); copyErr != nil {
		return "", fmt.Errorf("save tarball: %w", copyErr)
	}

	// Extract tarball into workspace (preserves archive root directory).
	if err := extractTarball(tarballPath, workspace); err != nil {
		return "", fmt.Errorf("extract tarball: %w", err)
	}

	return workspace, nil
}

// TarballPath returns the path to the downloaded tarball within a workspace.
func TarballPath(workspace string) string {
	return filepath.Join(workspace, "repo.tar.gz")
}

// SourceDir returns the path to the extracted source within a workspace.
// It discovers the extracted directory dynamically — GitHub tarballs always
// extract into a single root folder named "<owner>-<repo>-<commitsha>".
func SourceDir(workspace string) string {
	entries, err := os.ReadDir(workspace)
	if err != nil {
		return workspace
	}
	for _, e := range entries {
		if e.IsDir() {
			return filepath.Join(workspace, e.Name())
		}
	}
	return workspace
}

// ExtractVersion parses the commit SHA from the extracted tarball root directory name.
// GitHub tarballs always extract into a root folder named "<owner>-<repo>-<commitsha>".
// The returned value is truncated to 7 characters (standard short SHA length).
func ExtractVersion(workspace, owner, repo string) string {
	sourceDir := SourceDir(workspace)
	dirName := filepath.Base(sourceDir)
	prefix := fmt.Sprintf("%s-%s-", owner, repo)

	if strings.HasPrefix(dirName, prefix) {
		v := strings.TrimPrefix(dirName, prefix)
		if len(v) > 7 {
			v = v[:7]
		}
		return v
	}
	return ""
}
