package spec

import (
	"strings"
	"testing"
)

// --- Hard error tests ---

func TestValidateNilSpec(t *testing.T) {
	_, err := Validate(nil)
	if err == nil {
		t.Fatal("expected error for nil spec, got nil")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error = %q, want it to mention nil", err.Error())
	}
}

func TestValidateMissingType(t *testing.T) {
	s := &TaitoSpec{Name: "my-skill"}
	_, err := Validate(s)
	if err == nil {
		t.Fatal("expected error for missing type")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Errorf("error = %q, want it to mention type", err.Error())
	}
}

func TestValidateInvalidType(t *testing.T) {
	s := &TaitoSpec{Type: "workflow", Name: "my-skill"}
	_, err := Validate(s)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !strings.Contains(err.Error(), "workflow") {
		t.Errorf("error = %q, want it to mention the invalid value", err.Error())
	}
}

func TestValidateMissingName(t *testing.T) {
	s := &TaitoSpec{Type: TypeSkill}
	_, err := Validate(s)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error = %q, want it to mention name", err.Error())
	}
}

// --- Warning tests ---

func TestValidateValidSpecNoWarnings(t *testing.T) {
	s := &TaitoSpec{
		Type:         TypeSkill,
		Name:         "git-commit-helper",
		Version:      "1.0.0",
		TaitoVersion: "0.1.0",
		Description:  "A helpful skill",
		Keywords:     []string{"git", "commit"},
		Author:       &Author{Name: "Taito"},
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d:", len(warnings))
		for _, w := range warnings {
			t.Errorf("  %s: %s", w.Field, w.Message)
		}
	}
}

func TestValidateMinimalSpecNoWarnings(t *testing.T) {
	s := &TaitoSpec{Type: TypeAgent, Name: "my-agent"}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestValidateNamePatternWarning(t *testing.T) {
	tests := []struct {
		name    string
		wantWrn bool
	}{
		{"valid-name", false},
		{"a", false},
		{"0start", false},
		{"has_underscore", false},
		{"UPPERCASE", true},
		{"-leading-dash", true},
		{"has spaces", true},
		{"special!char", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &TaitoSpec{Type: TypeSkill, Name: tt.name}
			warnings, err := Validate(s)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			found := hasWarning(warnings, "name", "pattern")
			if found != tt.wantWrn {
				t.Errorf("name=%q: got pattern warning=%v, want %v", tt.name, found, tt.wantWrn)
			}
		})
	}
}

func TestValidateNameTooLong(t *testing.T) {
	longName := strings.Repeat("a", 129)
	s := &TaitoSpec{Type: TypeSkill, Name: longName}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasWarning(warnings, "name", "128") {
		t.Error("expected warning about name exceeding 128 characters")
	}
}

func TestValidateNameExactly128(t *testing.T) {
	name128 := strings.Repeat("a", 128)
	s := &TaitoSpec{Type: TypeSkill, Name: name128}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasWarning(warnings, "name", "128") {
		t.Error("name of exactly 128 characters should not trigger length warning")
	}
}

func TestValidateVersionSemverWarning(t *testing.T) {
	tests := []struct {
		version string
		wantWrn bool
	}{
		{"1.0.0", false},
		{"0.1.0-alpha", false},
		{"1.2.3+build.42", false},
		{"1.0.0-beta.1+sha.abc123", false},
		{"v1.0.0", true},
		{"1.0", true},
		{"not-a-version", true},
		{"", false}, // empty is fine (optional field)
	}

	for _, tt := range tests {
		t.Run("version="+tt.version, func(t *testing.T) {
			s := &TaitoSpec{Type: TypeSkill, Name: "test", Version: tt.version}
			warnings, err := Validate(s)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			found := hasWarning(warnings, "version", "semantic version")
			if found != tt.wantWrn {
				t.Errorf("version=%q: got semver warning=%v, want %v", tt.version, found, tt.wantWrn)
			}
		})
	}
}

func TestValidateTaitoVersionSemverWarning(t *testing.T) {
	s := &TaitoSpec{Type: TypeSkill, Name: "test", TaitoVersion: "bad"}
	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasWarning(warnings, "taitoVersion", "semantic version") {
		t.Error("expected semver warning for taitoVersion")
	}
}

func TestValidateTaitoVersionValidNoWarning(t *testing.T) {
	s := &TaitoSpec{Type: TypeSkill, Name: "test", TaitoVersion: "0.1.0"}
	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasWarning(warnings, "taitoVersion", "") {
		t.Error("valid taitoVersion should not trigger warning")
	}
}

func TestValidateDescriptionTooLong(t *testing.T) {
	s := &TaitoSpec{
		Type:        TypeSkill,
		Name:        "test",
		Description: strings.Repeat("x", 501),
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasWarning(warnings, "description", "500") {
		t.Error("expected warning about description exceeding 500 characters")
	}
}

func TestValidateDescriptionExactly500(t *testing.T) {
	s := &TaitoSpec{
		Type:        TypeSkill,
		Name:        "test",
		Description: strings.Repeat("x", 500),
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasWarning(warnings, "description", "500") {
		t.Error("description of exactly 500 characters should not trigger warning")
	}
}

func TestValidateKeywordsTooMany(t *testing.T) {
	keywords := make([]string, 21)
	for i := range keywords {
		keywords[i] = "kw"
	}
	s := &TaitoSpec{Type: TypeSkill, Name: "test", Keywords: keywords}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasWarning(warnings, "keywords", "20") {
		t.Error("expected warning about exceeding 20 keywords")
	}
}

func TestValidateKeywordBadPattern(t *testing.T) {
	s := &TaitoSpec{Type: TypeSkill, Name: "test", Keywords: []string{"valid", "BAD"}}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasWarning(warnings, "keywords[0]", "pattern") {
		t.Error("keywords[0]='valid' should not trigger warning")
	}
	if !hasWarning(warnings, "keywords[1]", "pattern") {
		t.Error("keywords[1]='BAD' should trigger pattern warning")
	}
}

func TestValidateKeywordTooLong(t *testing.T) {
	longKw := strings.Repeat("a", 65)
	s := &TaitoSpec{Type: TypeSkill, Name: "test", Keywords: []string{longKw}}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasWarning(warnings, "keywords[0]", "64") {
		t.Error("expected warning about keyword exceeding 64 characters")
	}
}

func TestValidateIncludesOnNonBundle(t *testing.T) {
	for _, typ := range []string{TypeSkill, TypeAgent} {
		t.Run(typ, func(t *testing.T) {
			s := &TaitoSpec{
				Type:     typ,
				Name:     "test",
				Includes: []string{"./skills/foo/taito.spec"},
			}
			warnings, err := Validate(s)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !hasWarning(warnings, "includes", "only valid") {
				t.Errorf("expected warning that includes is only valid for bundles, type=%s", typ)
			}
		})
	}
}

func TestValidateIncludesOnBundleNoWarning(t *testing.T) {
	s := &TaitoSpec{
		Type:     TypeBundle,
		Name:     "my-bundle",
		Includes: []string{"./skills/foo/taito.spec"},
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasWarning(warnings, "includes", "") {
		t.Error("includes on a bundle type should not trigger warning")
	}
}

func TestValidateIncludesAbsolutePath(t *testing.T) {
	s := &TaitoSpec{
		Type:     TypeBundle,
		Name:     "my-bundle",
		Includes: []string{"/etc/taito.spec"},
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasWarning(warnings, "includes[0]", "absolute") {
		t.Error("expected warning about absolute path in includes")
	}
}

func TestValidateIncludesParentTraversal(t *testing.T) {
	s := &TaitoSpec{
		Type:     TypeBundle,
		Name:     "my-bundle",
		Includes: []string{"../../outside/taito.spec"},
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasWarning(warnings, "includes[0]", "traverse") {
		t.Error("expected warning about parent traversal in includes path")
	}
}

func TestValidateIncludesValidPaths(t *testing.T) {
	s := &TaitoSpec{
		Type: TypeBundle,
		Name: "my-bundle",
		Includes: []string{
			"./skills/git-helper/taito.spec",
			"agents/devops-agent/taito.spec",
		},
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, w := range warnings {
		if strings.HasPrefix(w.Field, "includes[") {
			t.Errorf("unexpected includes warning: %s: %s", w.Field, w.Message)
		}
	}
}

func TestValidateAuthorWithoutName(t *testing.T) {
	s := &TaitoSpec{
		Type:   TypeSkill,
		Name:   "test",
		Author: &Author{Email: "test@example.com"},
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasWarning(warnings, "author.name", "name") {
		t.Error("expected warning about author missing name")
	}
}

func TestValidateAuthorWithNameNoWarning(t *testing.T) {
	s := &TaitoSpec{
		Type:   TypeSkill,
		Name:   "test",
		Author: &Author{Name: "Taito"},
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasWarning(warnings, "author", "") {
		t.Error("valid author should not trigger warning")
	}
}

func TestValidateNoAuthorNoWarning(t *testing.T) {
	s := &TaitoSpec{Type: TypeSkill, Name: "test"}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasWarning(warnings, "author", "") {
		t.Error("nil author should not trigger warning")
	}
}

func TestValidateAllThreeTypes(t *testing.T) {
	for _, typ := range []string{TypeSkill, TypeAgent, TypeBundle} {
		t.Run(typ, func(t *testing.T) {
			s := &TaitoSpec{Type: typ, Name: "test"}
			_, err := Validate(s)
			if err != nil {
				t.Errorf("type %q should be valid, got error: %v", typ, err)
			}
		})
	}
}

func TestValidateMultipleWarnings(t *testing.T) {
	// A spec with several issues at once should report all of them.
	s := &TaitoSpec{
		Type:         TypeSkill,
		Name:         "INVALID NAME!",
		Version:      "bad",
		TaitoVersion: "also-bad",
		Description:  strings.Repeat("x", 501),
		Keywords:     []string{"UPPER"},
		Includes:     []string{"./should-not-be-here/taito.spec"},
		Author:       &Author{Email: "no-name@example.com"},
	}

	warnings, err := Validate(s)
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}

	// We expect at least warnings for: name pattern, version semver,
	// taitoVersion semver, description length, keywords[0] pattern,
	// includes on non-bundle, author.name.
	if len(warnings) < 7 {
		t.Errorf("expected at least 7 warnings, got %d:", len(warnings))
		for _, w := range warnings {
			t.Errorf("  %s: %s", w.Field, w.Message)
		}
	}
}

// hasWarning checks if any warning matches the given field and contains the
// given substring in its message. If substr is empty, it matches any message
// for that field.
func hasWarning(warnings []Warning, field, substr string) bool {
	for _, w := range warnings {
		if w.Field == field {
			if substr == "" || strings.Contains(w.Message, substr) {
				return true
			}
		}
	}
	return false
}
