package github

import (
	"github.com/taito-project/taito/internal/tarutil"
)

// extractTarball extracts a tar.gz archive to targetDir, preserving the
// archive's root directory (e.g. "owner-repo-commitsha/"). This keeps the
// original folder name so the commit SHA can be recovered later.
func extractTarball(tarGzPath, targetDir string) error {
	return tarutil.Extract(tarGzPath, targetDir, tarutil.ExtractOptions{})
}
