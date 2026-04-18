package update

import (
	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/github"
	"github.com/taito-project/taito/internal/install"
	"github.com/taito-project/taito/internal/oci"
	"github.com/taito-project/taito/ui"
)

type InstallResult struct {
	Name string
	Err  error
}

func Install(ref string, cfg *config.Config) InstallResult {
	name := ref
	installer := newInstaller(ref, cfg)

	if cleaner, ok := installer.(interface{ Cleanup() }); ok {
		defer cleaner.Cleanup()
	}

	resolveMsg := installer.Resolve()
	if msg, ok := resolveMsg.(ui.InstallResolveMsg); ok && msg.Err != nil {
		return InstallResult{Name: name, Err: msg.Err}
	}

	extractMsg := installer.Extract()
	if msg, ok := extractMsg.(ui.InstallExtractMsg); ok {
		if msg.Err != nil {
			return InstallResult{Name: name, Err: msg.Err}
		}
		name = msg.Name
	}

	installMsg := installer.Install()
	if msg, ok := installMsg.(ui.InstallResultMsg); ok && msg.Err != nil {
		return InstallResult{Name: name, Err: msg.Err}
	}

	return InstallResult{Name: name}
}

func newInstaller(ref string, cfg *config.Config) ui.Installer {
	if install.IsLocalPath(ref) || !github.IsGitHubSource(ref) {
		return oci.NewInstaller(ref, cfg)
	}

	return github.NewInstaller(ref, cfg)
}
