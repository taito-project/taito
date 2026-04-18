package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestResolveVersionWithTag(t *testing.T) {
	// When the user specifies a tag (e.g. ":v2.0.0"), the resolved
	// version should be the tag, NOT the commit SHA from the tarball
	// directory name.
	tarball := buildTestTarball(t, "larszi-somecoolSkill-abc1234", map[string]string{
		"taito.spec": `{"type":"bundle","name":"somecoolSkill"}`,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/larszi/somecoolSkill/tarball/v2.0.0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(tarball)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{base: server.URL, wrapped: origTransport}
	defer func() { http.DefaultTransport = origTransport }()

	result, err := Resolve("github.com/larszi/somecoolSkill:v2.0.0")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	defer func() { _ = os.RemoveAll(result.Workspace) }()

	if result.Version != "v2.0.0" {
		t.Errorf("Version = %q, want %q", result.Version, "v2.0.0")
	}
}

func TestResolveVersionWithoutTag(t *testing.T) {
	// When the user does NOT specify a tag, the resolved version should
	// be the short commit SHA extracted from the tarball directory name.
	tarball := buildTestTarball(t, "larszi-somecoolSkill-abc1234def5678", map[string]string{
		"taito.spec": `{"type":"bundle","name":"somecoolSkill"}`,
	})

	mux := http.NewServeMux()
	// ResolveRef will query the repo API to get the default branch.
	mux.HandleFunc("/repos/larszi/somecoolSkill", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"default_branch": "main",
		})
	})
	mux.HandleFunc("/repos/larszi/somecoolSkill/tarball/main", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(tarball)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{base: server.URL, wrapped: origTransport}
	defer func() { http.DefaultTransport = origTransport }()

	result, err := Resolve("github.com/larszi/somecoolSkill")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	defer func() { _ = os.RemoveAll(result.Workspace) }()

	// ExtractVersion truncates to 7 characters.
	want := "abc1234"
	if result.Version != want {
		t.Errorf("Version = %q, want %q", result.Version, want)
	}
}
