# Taito CLI Developer Guide

Welcome to the Taito CLI development guide! This document provides an overview of the architecture, technical details, and workflows for developers and maintainers looking to contribute to the `taito` application.

## 1. Introduction & Technology Stack

**Taito** is a command-line interface and terminal UI application for packaging, distributing, and managing AI skills, agents, and bundles via OCI registries and GitHub repositories.

### Core Dependencies
- **[Go](https://golang.org/):** See `go.mod` for the required version.
- **[Cobra](https://github.com/spf13/cobra):** Standard framework for creating powerful modern CLI applications.
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea) & [Lipgloss](https://github.com/charmbracelet/lipgloss):** Used for rendering the interactive terminal UI (TUI) and providing terminal styling.
- **[ORAS Go v2](https://github.com/oras-project/oras-go):** Handles interactions with OCI registries and artifacts.

## 2. Architecture Overview

The project is structured modularly to separate the standard CLI concerns from the interactive UI and the core domain logic:

- **`cmd/` (CLI Layer)**
  - Defines the entry points and CLI commands (see [Command Reference](#3-command-reference) below).
  - Uses `spf13/cobra` to parse flags, arguments, and delegate commands.
  - `root.go` is the main entry point. It loads user configuration via `PersistentPreRun` (stored in the cobra command context) and falls back to launching the TUI via `tea.NewProgram(ui.InitialModel())` if no subcommands are passed.

- **`ui/` (Terminal UI Layer)**
  - Built with `charmbracelet/bubbletea` and styled using `lipgloss`.
  - Contains state machines, thematic styling (`theme.go`), and specific interactive views for each command.

- **`internal/` (Core Business Logic)**
  - Pure functions, parsers, and business logic should reside here to avoid coupling with `cmd/` or `ui/`.
  - **`internal/spec`:** Parsing, loading, and validating `taito.spec` manifests against the v0.1.0 schema.
  - **`internal/archive`:** OCI artifact creation using `oras-go/v2` and local tarball creation. Also defines shared OCI constants (media types, annotation keys).
  - **`internal/registry`:** Remote OCI registry operations (push, pull, validation, credential management).
  - **`internal/install`:** Installation tracking (installed index), uninstall logic, and tool display names.
  - **`internal/oci`:** OCI-based installer for installing artifacts into configured AI coding tools.
  - **`internal/github`:** GitHub repository URL parsing, reference resolution, tarball downloading, and GitHub-based installer.
  - **`internal/update`:** Update checking (semver comparison, GitHub tag/commit comparison) and update installation.
  - **`internal/config`:** User configuration loading, saving, and platform-appropriate file paths.
  - **`internal/cache`:** Cache index management, package path hashing, and cache scanning/pruning.
  - **`internal/tarutil`:** Tar.gz extraction utilities.
  - **`internal/httputil`:** Shared HTTP client with timeouts.
  - **`internal/fsutil`:** Shared filesystem utilities (recursive directory copy).
  - **`internal/ociref`:** Dependency-free OCI reference tag parsing (avoids import cycles).

## 3. Command Reference

| Command | Aliases | Description |
|---------|---------|-------------|
| `taito` | — | Launches the interactive Bubble Tea TUI when no subcommand is given. |
| `taito init` | — | Interactive wizard to generate a new `taito.spec` manifest in the current directory. |
| `taito check [path]` | — | Validates a `taito.spec` manifest file against the v0.1.0 schema. Accepts `--path` flag (default `.`). |
| `taito package [reference]` | — | Packages a skill/agent/bundle into an OCI artifact or tar.gz archive. Flags: `--spec`, `--format` (oci/tar.gz), `--output`. |
| `taito push <reference>` | — | Pushes a local OCI layout to a remote OCI registry. Flag: `--source`. Requires `taito login`. |
| `taito pull <reference>` | — | Pulls an artifact from a remote OCI registry into a local OCI layout. Flag: `--output`. Requires `taito login`. |
| `taito install <reference\|path>` | — | Installs a taito artifact from an OCI registry, local OCI layout, or GitHub repository into all configured AI coding tools. Requires `taito setup`. |
| `taito uninstall <id>` | `rm` | Removes a package by ID from all configured tools. Bundle children are also removed. |
| `taito update [id]` | `up` | Checks for and applies updates to installed packages. Without arguments, checks all. |
| `taito list` | `ls` | Lists all installed packages in a styled table showing ID, name, type, tool, and version. |
| `taito setup` | — | Interactive wizard to configure AI coding tools (Copilot, Claude Code, OpenCode). |
| `taito login <registry>` | — | Authenticates to an OCI registry. Supports `--username`, `--password`, `--password-stdin` for non-interactive use. |
| `taito logout <registry>` | — | Removes stored credentials for a given OCI registry. |
| `taito prune` | — | Removes all cached artifacts. Flag: `--dry-run`. |
| `taito version` | — | Prints the taito version string. |

## 4. Key Execution Workflows

- **Interactive Mode (`taito`)**:
  1. `main.go` → `cmd/root.go` triggers execution.
  2. `PersistentPreRun` loads user config into the command context.
  3. No subcommand is detected, invoking the fallback logic.
  4. Launches `tea.NewProgram(ui.InitialModel())`.

- **Validation (`taito check`)**:
  1. `cmd/check.go` parses the path argument or `--path` flag.
  2. Delegates to `internal/spec.Load()` to load the manifest.
  3. Validates against the v0.1.0 schema using `internal/spec.Validate()`.
  4. Reports hard errors (exit code 1) and warnings via Bubble Tea UI.

- **Packaging (`taito package`)**:
  1. `cmd/package.go` reads positional args and flags (`--format`, `--spec`, `--output`).
  2. Validates the `taito.spec` file in the context directory.
  3. Invokes `internal/archive.CreateOCILayout()` or `internal/archive.CreateTarGz()` depending on format.
  4. On success, updates the cache index.

- **Pull & Push (`taito pull`, `taito push`)**:
  1. Authenticates via `internal/registry.NewRepository()` using stored credentials.
  2. Pull: copies from remote → temp dir → validates → moves to target (cache or `--output`).
  3. Push: validates local OCI layout → copies to remote registry.

- **Install (`taito install`)**:
  1. Determines source type: local OCI layout, OCI registry reference, or GitHub repository.
  2. For OCI: uses `internal/oci.NewInstaller`. For GitHub: uses `internal/github.NewInstaller`.
  3. Extracts artifacts and copies to each configured tool's directory.
  4. Bundles are expanded so each child skill/agent gets its own directory.
  5. Updates the installed index (`internal/install`).

- **Update (`taito update`)**:
  1. Loads the installed index and checks each entry for newer versions.
  2. OCI packages: lists remote tags and compares via semver.
  3. GitHub packages: fetches tags/commits from the GitHub API.
  4. Displays a summary and asks for confirmation before applying.

## 5. Development Setup & Best Practices

- **Testing:** We emphasize testing core business logic and command execution.
  - To run all tests: `go test ./...`
  - Unit tests are located alongside their respective implementations (e.g., `cmd/check_test.go`, `internal/spec/validate_test.go`).
- **Code Style:** Keep business logic out of `cmd/` and `ui/` as much as possible. `cmd/` should only handle argument parsing and basic I/O wiring, while `ui/` should strictly manage the view state and terminal interaction.
- **Configuration:** User config is loaded once via `PersistentPreRun` in `root.go` and passed through the cobra command context. Use `cfgFromCmd(cmd)` to access it in command handlers. Avoid global mutable state.
- **Constants:** OCI annotation keys are defined in `internal/archive` (`AnnotationSpecType`, `AnnotationSpecName`, `AnnotationSpecLicense`). GitHub API constants are in `internal/github` (`APIBase`, `AcceptHeader`). Use these instead of string literals.

## 6. Adding a New Command

To add a new feature or command to the CLI, follow these steps:

1. **Create the CLI Command:** Create a new file in `cmd/` (e.g., `cmd/newcmd.go`).
2. **Define the Cobra Command:**
   ```go
   var newCmd = &cobra.Command{
       Use:   "newcmd",
       Short: "A brief description",
       Run: func(cmd *cobra.Command, args []string) {
           cfg := cfgFromCmd(cmd)
           // Parse flags/args and call logic in internal/
       },
   }
   ```
3. **Register the Command:** In the `init()` block of your new file, register it with the root command:
   ```go
   func init() {
       rootCmd.AddCommand(newCmd)
   }
   ```
4. **Implement Core Logic:** Place the actual heavy lifting in an appropriate package under `internal/`.
5. **Add UI Integration (Optional):** If the command should also be accessible via the interactive Bubble Tea interface, update `ui/model.go` to handle the new state, and add corresponding views in the `ui/` package.
6. **Write Tests:** Add a `_test.go` file for both the Cobra command parser (`cmd/newcmd_test.go`) and the underlying logic (`internal/...`).
