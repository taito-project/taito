package oci

import (
	"testing"

	"github.com/taito-project/taito/internal/config"
)

func TestNewInstaller(t *testing.T) {
	cfg := &config.Config{}
	inst := NewInstaller("ghcr.io/org/skill:v1.0.0", cfg)
	if inst == nil {
		t.Fatal("expected non-nil installer")
	}
	if inst.source != "ghcr.io/org/skill:v1.0.0" {
		t.Errorf("source = %q, want ghcr.io/org/skill:v1.0.0", inst.source)
	}
}

func TestResolveBundleItems_EmptyIncludes(t *testing.T) {
	items := resolveBundleItems("/nonexistent", nil)
	if items != nil {
		t.Errorf("expected nil items for empty includes, got %v", items)
	}
}

func TestResolveBundleItems_InvalidLayout(t *testing.T) {
	items := resolveBundleItems("/nonexistent", []string{"./skills/a/taito.spec"})
	if items != nil {
		t.Errorf("expected nil items for invalid layout, got %v", items)
	}
}
