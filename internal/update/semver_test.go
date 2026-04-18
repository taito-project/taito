package update

import (
	"fmt"
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		ok    bool
		major int
		minor int
		patch int
		pre   string
	}{
		{"1.2.3", true, 1, 2, 3, ""},
		{"v1.2.3", true, 1, 2, 3, ""},
		{"0.0.0", true, 0, 0, 0, ""},
		{"v10.20.30", true, 10, 20, 30, ""},
		{"1.2.3-alpha.1", true, 1, 2, 3, "alpha.1"},
		{"v1.0.0-beta", true, 1, 0, 0, "beta"},
		{"1.2.3+build.123", true, 1, 2, 3, ""},
		{"1.2.3-rc.1+build", true, 1, 2, 3, "rc.1+build"},
		// Invalid cases
		{"", false, 0, 0, 0, ""},
		{"not-a-version", false, 0, 0, 0, ""},
		{"1.2", false, 0, 0, 0, ""},
		{"1.2.3.4", false, 0, 0, 0, ""},
		{"v1.2.x", false, 0, 0, 0, ""},
		{"abc123", false, 0, 0, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			sv, ok := ParseSemver(tt.input)
			if ok != tt.ok {
				t.Fatalf("ParseSemver(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if !ok {
				return
			}
			if sv.Major != tt.major || sv.Minor != tt.minor || sv.Patch != tt.patch {
				t.Errorf("ParseSemver(%q) = %d.%d.%d, want %d.%d.%d",
					tt.input, sv.Major, sv.Minor, sv.Patch, tt.major, tt.minor, tt.patch)
			}
			if sv.Prerelease != tt.pre {
				t.Errorf("ParseSemver(%q) prerelease = %q, want %q", tt.input, sv.Prerelease, tt.pre)
			}
		})
	}
}

func TestSemverCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},
		// Prerelease has lower precedence than stable
		{"1.0.0", "1.0.0-alpha", 1},
		{"1.0.0-alpha", "1.0.0", -1},
		// Prerelease ordering (lexicographic)
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-beta", "1.0.0-alpha", 1},
		{"1.0.0-alpha", "1.0.0-alpha", 0},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			a, _ := ParseSemver(tt.a)
			b, _ := ParseSemver(tt.b)
			got := a.Compare(b)
			if got != tt.want {
				t.Errorf("(%s).Compare(%s) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsNewerThan(t *testing.T) {
	a, _ := ParseSemver("2.0.0")
	b, _ := ParseSemver("1.0.0")
	if !a.IsNewerThan(b) {
		t.Error("2.0.0 should be newer than 1.0.0")
	}
	if b.IsNewerThan(a) {
		t.Error("1.0.0 should not be newer than 2.0.0")
	}
}

func TestFilterSemverTags(t *testing.T) {
	tags := []string{"v2.0.0", "not-semver", "v1.0.0", "v3.0.0-alpha", "v1.5.0", "garbage"}
	result := FilterSemverTags(tags)

	if len(result) != 4 {
		t.Fatalf("expected 4 valid tags, got %d", len(result))
	}

	// Should be sorted descending: v2.0.0, v1.5.0, v1.0.0, v3.0.0-alpha
	// v3.0.0-alpha < v3.0.0 (stable) but 3.0.0 > 2.0.0, however
	// alpha has lower precedence. Let's just check first is v3.0.0-alpha
	// (major 3 is highest) and ordering makes sense.
	// 3.0.0-alpha has major=3, so it sorts highest by major
	// even though it's prerelease. The Compare function compares
	// major first.
	if result[0].Original != "v3.0.0-alpha" {
		t.Errorf("expected first element to be v3.0.0-alpha, got %s", result[0].Original)
	}
	// Just verify descending order
	for i := 1; i < len(result); i++ {
		if result[i].Compare(result[i-1]) > 0 {
			t.Errorf("not sorted descending: %s > %s", result[i].Original, result[i-1].Original)
		}
	}
}

func TestLatestSemverTag(t *testing.T) {
	tags := []string{"v1.0.0", "v2.0.0", "v1.5.0"}
	latest, found := LatestSemverTag(tags)
	if !found {
		t.Fatal("expected to find a semver tag")
	}
	if latest.Original != "v2.0.0" {
		t.Errorf("expected v2.0.0, got %s", latest.Original)
	}

	_, found = LatestSemverTag([]string{"not-semver", "garbage"})
	if found {
		t.Error("expected no semver tags found")
	}
}

func TestIsSemver(t *testing.T) {
	if !IsSemver("v1.0.0") {
		t.Error("v1.0.0 should be semver")
	}
	if IsSemver("not-semver") {
		t.Error("not-semver should not be semver")
	}
}

func TestBuildUpdateReference(t *testing.T) {
	tests := []struct {
		name       string
		reference  string
		newVersion string
		want       string
	}{
		{"OCI with tag", "ghcr.io/org/skill:v1.0.0", "v2.0.0", "ghcr.io/org/skill:v2.0.0"},
		{"GitHub style", "github.com/owner/repo@v1.0.0", "v2.0.0", "github.com/owner/repo@v2.0.0"},
		{"OCI no tag", "ghcr.io/org/skill", "v1.0.0", "ghcr.io/org/skill:v1.0.0"},
		{"empty ref", "", "v1.0.0", ""},
		{"empty version", "ghcr.io/org/skill:v1.0.0", "", "ghcr.io/org/skill:v1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildUpdateReference(tt.reference, tt.newVersion)
			if got != tt.want {
				t.Errorf("buildUpdateReference(%q, %q) = %q, want %q", tt.reference, tt.newVersion, got, tt.want)
			}
		})
	}
}

func TestUpdatableResults(t *testing.T) {
	results := []UpdateResult{
		{Name: "a", HasUpdate: true},
		{Name: "b", HasUpdate: false},
		{Name: "c", HasUpdate: true, IsBundleChild: true},
		{Name: "d", HasUpdate: true, Error: fmt.Errorf("err")},
		{Name: "e", HasUpdate: true, IsLocal: false},
	}

	updatable := UpdatableResults(results)
	if len(updatable) != 2 {
		t.Fatalf("expected 2 updatable, got %d", len(updatable))
	}
	if updatable[0].Name != "a" || updatable[1].Name != "e" {
		t.Errorf("unexpected updatable results: %v", updatable)
	}
}
