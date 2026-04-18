// Package update provides version checking logic for the "taito update" command.
package update

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Semver represents a parsed semantic version.
type Semver struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string // e.g. "alpha.1" (empty if stable)
	Original   string // the original string, preserved for display
}

// ParseSemver parses a version string into a Semver. It accepts an optional
// "v" prefix (e.g. "v1.2.3") and optional prerelease suffix (e.g. "1.2.3-beta.1").
// Returns ok=false if the string is not a valid semver.
func ParseSemver(s string) (Semver, bool) {
	original := s
	s = strings.TrimPrefix(s, "v")

	// Split off prerelease (everything after the first hyphen).
	var prerelease string
	if idx := strings.IndexByte(s, '-'); idx != -1 {
		prerelease = s[idx+1:]
		s = s[:idx]
	}

	// Split off build metadata (everything after +). We discard it for comparison.
	if idx := strings.IndexByte(s, '+'); idx != -1 {
		s = s[:idx]
	}

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Semver{}, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return Semver{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 {
		return Semver{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil || patch < 0 {
		return Semver{}, false
	}

	return Semver{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
		Original:   original,
	}, true
}

// IsSemver returns true if s can be parsed as a semantic version.
func IsSemver(s string) bool {
	_, ok := ParseSemver(s)
	return ok
}

// String returns the original version string.
func (v Semver) String() string {
	if v.Original != "" {
		return v.Original
	}
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		return base + "-" + v.Prerelease
	}
	return base
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other.
// Follows semver spec: prerelease versions have lower precedence than the
// associated normal version.
func (v Semver) Compare(other Semver) int {
	if v.Major != other.Major {
		return intCmp(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return intCmp(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return intCmp(v.Patch, other.Patch)
	}

	// Both stable → equal.
	if v.Prerelease == "" && other.Prerelease == "" {
		return 0
	}
	// Stable > prerelease.
	if v.Prerelease == "" {
		return 1
	}
	if other.Prerelease == "" {
		return -1
	}
	// Both prerelease → lexicographic (good enough for our needs).
	if v.Prerelease < other.Prerelease {
		return -1
	}
	if v.Prerelease > other.Prerelease {
		return 1
	}
	return 0
}

// IsNewerThan returns true if v is a higher version than other.
func (v Semver) IsNewerThan(other Semver) bool {
	return v.Compare(other) > 0
}

// FilterSemverTags filters a list of strings to only valid semver tags and
// returns them sorted descending (newest first).
func FilterSemverTags(tags []string) []Semver {
	var versions []Semver
	for _, tag := range tags {
		if sv, ok := ParseSemver(tag); ok {
			versions = append(versions, sv)
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Compare(versions[j]) > 0
	})
	return versions
}

// LatestSemverTag finds the highest semver tag from a list of tags.
// Returns the version and true if found, or an empty Semver and false if no
// valid semver tags exist.
func LatestSemverTag(tags []string) (Semver, bool) {
	filtered := FilterSemverTags(tags)
	if len(filtered) == 0 {
		return Semver{}, false
	}
	return filtered[0], true
}

func intCmp(a, b int) int {
	if a < b {
		return -1
	}
	return 1
}
