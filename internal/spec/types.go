// Package spec provides data structures and utilities for reading and
// validating taito.spec manifest files (v0.1.0).
package spec

// Package type constants.
const (
	TypeSkill  = "skill"
	TypeAgent  = "agent"
	TypeBundle = "bundle"
)

// TaitoSpec represents the root manifest structure of a taito.spec file.
type TaitoSpec struct {
	TaitoVersion string   `json:"taitoVersion,omitempty"`
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Version      string   `json:"version,omitempty"`
	Description  string   `json:"description,omitempty"`
	Source       string   `json:"source,omitempty"`
	Author       *Author  `json:"author,omitempty"`
	License      string   `json:"license,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
	Includes     []string `json:"includes,omitempty"` // bundle only
}

// Author describes the author of a skill, agent, or bundle.
type Author struct {
	Name  string `json:"name"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// Warning represents a non-fatal validation issue found in a taito.spec file.
type Warning struct {
	Field   string // The field that caused the warning (e.g. "name", "includes[2]").
	Message string // A human-readable description of the issue.
}
