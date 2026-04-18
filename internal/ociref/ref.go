// Package ociref provides shared OCI reference parsing utilities.
// This package is intentionally dependency-free to avoid import cycles
// between packages like archive and registry.
package ociref

import "strings"

// TagFromReference extracts the tag portion from an OCI reference string.
// For "ghcr.io/org/name:v1.0.0" it returns "v1.0.0".
// For "ghcr.io/org/name" (no tag) it returns "latest".
// For "ghcr.io/org/name@sha256:abc..." it returns the full digest reference.
// For a bare tag like "v1.0.0" (no slash, no colon) it returns the string as-is.
func TagFromReference(reference string) string {
	// Handle digest references.
	if idx := strings.LastIndex(reference, "@"); idx >= 0 {
		return reference[idx+1:]
	}
	// Handle tag references.
	// Find the tag portion after the last colon, but only if it's after the
	// last slash (to avoid matching the port in "localhost:5000/name").
	lastSlash := strings.LastIndex(reference, "/")
	lastColon := strings.LastIndex(reference, ":")
	if lastColon > lastSlash {
		return reference[lastColon+1:]
	}
	// No slash and no colon — treat the whole string as a bare tag.
	if lastSlash < 0 {
		return reference
	}
	return "latest"
}
