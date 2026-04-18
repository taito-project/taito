package spec

import (
	"fmt"
	"path/filepath"
	"regexp"
)

var (
	namePattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
	semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
		`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
		`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	keywordPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

	validTypes = map[string]bool{
		TypeSkill:  true,
		TypeAgent:  true,
		TypeBundle: true,
	}
)

// warnFunc is a callback that appends a validation warning.
type warnFunc func(field, msg string)

// Validate checks a TaitoSpec for conformance with the v0.1.0 specification.
//
// Hard errors (returned as error) are reserved for unrecoverable issues:
//   - missing or invalid type field
//   - missing name field
//
// Everything else is returned as a list of warnings.
func Validate(s *TaitoSpec) ([]Warning, error) {
	if s == nil {
		return nil, fmt.Errorf("spec is nil")
	}

	if err := validateRequired(s); err != nil {
		return nil, err
	}

	var warnings []Warning
	warn := func(field, msg string) {
		warnings = append(warnings, Warning{Field: field, Message: msg})
	}

	validateName(s, warn)
	validateVersion(s, warn)
	validateDescription(s, warn)
	validateKeywords(s, warn)
	validateIncludes(s, warn)
	validateAuthor(s, warn)

	return warnings, nil
}

// validateRequired checks the hard-error fields: type and name.
func validateRequired(s *TaitoSpec) error {
	if s.Type == "" {
		return fmt.Errorf("required field \"type\" is missing")
	}
	if !validTypes[s.Type] {
		return fmt.Errorf("field \"type\" has invalid value %q: must be one of \"skill\", \"agent\", \"bundle\"", s.Type)
	}
	if s.Name == "" {
		return fmt.Errorf("required field \"name\" is missing")
	}
	return nil
}

// validateName checks the name field pattern and length.
func validateName(s *TaitoSpec, warn warnFunc) {
	if !namePattern.MatchString(s.Name) {
		warn("name", fmt.Sprintf("value %q does not match required pattern ^[a-z0-9][a-z0-9_-]*$", s.Name))
	}
	if len(s.Name) > 128 {
		warn("name", fmt.Sprintf("value exceeds maximum length of 128 characters (got %d)", len(s.Name)))
	}
}

// validateVersion checks version and taitoVersion fields for semver conformance.
func validateVersion(s *TaitoSpec, warn warnFunc) {
	if s.Version != "" && !semverPattern.MatchString(s.Version) {
		warn("version", fmt.Sprintf("value %q is not a valid semantic version", s.Version))
	}
	if s.TaitoVersion != "" && !semverPattern.MatchString(s.TaitoVersion) {
		warn("taitoVersion", fmt.Sprintf("value %q is not a valid semantic version", s.TaitoVersion))
	}
}

// validateDescription checks the description length.
func validateDescription(s *TaitoSpec, warn warnFunc) {
	if len(s.Description) > 500 {
		warn("description", fmt.Sprintf("exceeds maximum length of 500 characters (got %d)", len(s.Description)))
	}
}

// validateKeywords checks keyword count, pattern, and length.
func validateKeywords(s *TaitoSpec, warn warnFunc) {
	if len(s.Keywords) > 20 {
		warn("keywords", fmt.Sprintf("exceeds maximum of 20 items (got %d)", len(s.Keywords)))
	}
	for i, kw := range s.Keywords {
		field := fmt.Sprintf("keywords[%d]", i)
		if !keywordPattern.MatchString(kw) {
			warn(field, fmt.Sprintf("value %q does not match required pattern ^[a-z0-9][a-z0-9_-]*$", kw))
		}
		if len(kw) > 64 {
			warn(field, fmt.Sprintf("value exceeds maximum length of 64 characters (got %d)", len(kw)))
		}
	}
}

// validateIncludes checks includes field type-correctness and path safety.
func validateIncludes(s *TaitoSpec, warn warnFunc) {
	if s.Type != TypeBundle && len(s.Includes) > 0 {
		warn("includes", fmt.Sprintf("field is only valid for type %q, but type is %q", TypeBundle, s.Type))
	}
	for i, inc := range s.Includes {
		field := fmt.Sprintf("includes[%d]", i)
		if filepath.IsAbs(inc) {
			warn(field, fmt.Sprintf("path %q must be relative, not absolute", inc))
		}
		cleaned := filepath.Clean(inc)
		if len(cleaned) >= 2 && cleaned[:2] == ".." {
			warn(field, fmt.Sprintf("path %q must not traverse above the bundle root", inc))
		}
	}
}

// validateAuthor checks that author.name is present when author is set.
func validateAuthor(s *TaitoSpec, warn warnFunc) {
	if s.Author != nil && s.Author.Name == "" {
		warn("author.name", "author object is present but required field \"name\" is missing")
	}
}
