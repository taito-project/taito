// Package github provides GitHub repository URL parsing, reference resolution,
// and tarball downloading for "taito install github.com/..." sources.
package github

import (
	"fmt"
	"strings"
)

const gitHubDomain = "github.com/"

// Ref represents a parsed GitHub repository reference.
type Ref struct {
	Owner   string // e.g. "anthropics"
	Repo    string // e.g. "skills"
	Subdir  string // optional subdirectory, e.g. "agents/devops" (empty if none)
	Version string // optional version from @, e.g. "0.0.1" (empty if none)
}

// IsGitHubSource returns true if the source string looks like a GitHub
// repository reference (starts with "github.com/" or "https://github.com/").
func IsGitHubSource(source string) bool {
	s := normalize(source)
	if !strings.HasPrefix(s, gitHubDomain) {
		return false
	}
	// Must have something after "github.com/" (at least an owner).
	rest := strings.TrimPrefix(s, gitHubDomain)
	return rest != ""
}

// Parse parses a GitHub source string into its components.
//
// Accepted formats:
//
//	github.com/owner/repo
//	github.com/owner/repo@version
//	github.com/owner/repo/subdir
//	github.com/owner/repo/subdir@version
//	https://github.com/owner/repo
//	https://github.com/owner/repo@version
//	https://github.com/owner/repo/subdir@version
func Parse(source string) (*Ref, error) {
	s := normalize(source)

	if !strings.HasPrefix(s, gitHubDomain) {
		return nil, fmt.Errorf("not a GitHub reference: %q", source)
	}

	// Strip the "github.com/" prefix.
	rest := strings.TrimPrefix(s, gitHubDomain)
	if rest == "" {
		return nil, fmt.Errorf("missing owner/repo in %q", source)
	}

	// Split off the @version or :version suffix if present.
	var version string
	if idx := strings.LastIndexAny(rest, "@:"); idx != -1 {
		version = rest[idx+1:]
		rest = rest[:idx]
		if version == "" {
			return nil, fmt.Errorf("empty version after @ or : in %q", source)
		}
	}

	// Split the remaining path into components.
	parts := strings.Split(rest, "/")

	// We need at least owner and repo.
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("missing owner/repo in %q", source)
	}

	ref := &Ref{
		Owner:   parts[0],
		Repo:    parts[1],
		Version: version,
	}

	// Anything after owner/repo is the subdirectory.
	if len(parts) > 2 {
		ref.Subdir = strings.Join(parts[2:], "/")
	}

	return ref, nil
}

// Normalized returns the canonical form of the source string for use as a
// cache key. This strips "https://" so that "https://github.com/owner/repo"
// and "github.com/owner/repo" produce the same hash.
func Normalized(source string) string {
	return normalize(source)
}

// normalize strips the https:// prefix if present.
func normalize(source string) string {
	s := strings.TrimPrefix(source, "https://")
	return s
}
