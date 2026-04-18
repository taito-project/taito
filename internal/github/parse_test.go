package github

import "testing"

func TestIsGitHubSource(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"github.com/owner/repo", true},
		{"github.com/owner/repo@v1", true},
		{"https://github.com/owner/repo", true},
		{"https://github.com/owner/repo@v1", true},
		{"github.com/owner/repo/subdir", true},
		{"ghcr.io/owner/repo:v1", false},
		{"./local-path", false},
		{"/absolute/path", false},
		{"registry.gitlab.com/org/skill:1.0.0", false},
		{"", false},
		{"github.com", false},
		{"github.com/", false},
	}

	for _, tc := range tests {
		t.Run(tc.source, func(t *testing.T) {
			got := IsGitHubSource(tc.source)
			if got != tc.want {
				t.Errorf("IsGitHubSource(%q) = %v, want %v", tc.source, got, tc.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    Ref
		wantErr bool
	}{
		{
			name:   "basic owner/repo",
			source: "github.com/anthropics/skills",
			want:   Ref{Owner: "anthropics", Repo: "skills"},
		},
		{
			name:   "with https prefix",
			source: "https://github.com/anthropics/skills",
			want:   Ref{Owner: "anthropics", Repo: "skills"},
		},
		{
			name:   "with version",
			source: "github.com/larszi/skill@0.0.1",
			want:   Ref{Owner: "larszi", Repo: "skill", Version: "0.0.1"},
		},
		{
			name:   "with version and https",
			source: "https://github.com/larszi/skill@0.0.1",
			want:   Ref{Owner: "larszi", Repo: "skill", Version: "0.0.1"},
		},
		{
			name:   "with version via colon",
			source: "github.com/larszi/skill:0.0.1",
			want:   Ref{Owner: "larszi", Repo: "skill", Version: "0.0.1"},
		},
		{
			name:   "with v-prefix version",
			source: "github.com/org/repo@v2.1.0",
			want:   Ref{Owner: "org", Repo: "repo", Version: "v2.1.0"},
		},
		{
			name:   "with branch as version",
			source: "github.com/org/repo@main",
			want:   Ref{Owner: "org", Repo: "repo", Version: "main"},
		},
		{
			name:   "with subdirectory",
			source: "github.com/taito-project/skills/agents/devops",
			want:   Ref{Owner: "taito-project", Repo: "skills", Subdir: "agents/devops"},
		},
		{
			name:   "with subdirectory and version",
			source: "github.com/taito-project/skills/agents/devops@v2",
			want:   Ref{Owner: "taito-project", Repo: "skills", Subdir: "agents/devops", Version: "v2"},
		},
		{
			name:   "deep subdirectory",
			source: "github.com/org/repo/a/b/c",
			want:   Ref{Owner: "org", Repo: "repo", Subdir: "a/b/c"},
		},
		{
			name:   "deep subdirectory with version",
			source: "github.com/org/repo/a/b/c@1.0",
			want:   Ref{Owner: "org", Repo: "repo", Subdir: "a/b/c", Version: "1.0"},
		},
		// Error cases.
		{
			name:    "not github",
			source:  "gitlab.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "missing repo",
			source:  "github.com/owner",
			wantErr: true,
		},
		{
			name:    "missing owner and repo",
			source:  "github.com/",
			wantErr: true,
		},
		{
			name:    "empty string",
			source:  "",
			wantErr: true,
		},
		{
			name:    "empty version after @",
			source:  "github.com/owner/repo@",
			wantErr: true,
		},
		{
			name:    "owner only with trailing slash",
			source:  "github.com/owner/",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.source)
			if tc.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) expected error, got %+v", tc.source, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tc.source, err)
			}
			if *got != tc.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tc.source, *got, tc.want)
			}
		})
	}
}

func TestNormalized(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"github.com/owner/repo", "github.com/owner/repo"},
		{"https://github.com/owner/repo", "github.com/owner/repo"},
		{"https://github.com/owner/repo@v1", "github.com/owner/repo@v1"},
		{"ghcr.io/owner/repo", "ghcr.io/owner/repo"},
	}

	for _, tc := range tests {
		t.Run(tc.source, func(t *testing.T) {
			got := Normalized(tc.source)
			if got != tc.want {
				t.Errorf("Normalized(%q) = %q, want %q", tc.source, got, tc.want)
			}
		})
	}
}

func TestNormalizedConsistency(t *testing.T) {
	// https:// and non-https:// versions should produce the same normalized form.
	a := Normalized("github.com/larszi/skill@0.0.1")
	b := Normalized("https://github.com/larszi/skill@0.0.1")
	if a != b {
		t.Errorf("normalized forms should match: %q != %q", a, b)
	}
}
