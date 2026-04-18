# Commands

For looking up the available commands, you also can run:

```bash
$ taito --help
```

## taito setup

The `setup` command is used to configure the taito cli and launch an interactive setup wizard for configuring.
And can be reun at any time to update the configuration.

Prompts for AI coding tools (Copilot, Claude Code, OpenCode..)

```bash
$ taito setup
```

<details>
  <summary>Example output:</summary>

```
$ taito setup
    AI coding tools
    Used for global skill and agent installation
    Select the tools you use (space to toggle, enter to confirm):
      > [ ] Cursor  ~/.cursor
        [ ] Windsurf  ~/.windsurf                                                    
        [x] Claude Code  ~/.claude                                                 
        [x] OpenCode  ~/.config/opencode
        [ ] Copilot  ~/.copilot
    esc: quit  space: toggle  enter: confirm

After confirming your selection:

taito setup

    ✓ Configuration saved

    Tools:
      Claude Code    ~/.claude
        agents/      ~/.claude/agents
        skills/      ~/.claude/skills
      OpenCode       ~/.config/opencode
        agents/      ~/.config/opencode/agents
        skills/      ~/.config/opencode/skills

    To customize tool paths, edit ~/.config/taito/config.json
    Saved to /home/larszi/.config/taito/config.json
```
</details>

## taito install

The `install` command is used to install bundle containing multiple skills or agents as well as only single ones from:
-  GitHub repository (containg taito.spec file/s)
-  GitHub repository (without taito.spec file/s, using convention)
-  OCI local taito package (alocalpage:v1.0.0)
-  OCI remote registry (ghcr.io/owner/repo:taito-v1.0.0)

```bash
$ taito install <source>:<version/tag>
```

### From Github repositories

You can choose to install from GitHub repositories directly. However, if the repository does not contain a taito.spec file, taito will attempt to install all skills and agents found in the repository using `skills/` and `agents/` as convention-based directories.

If you find your favorite repository does not contain taito.spec files, consider contributing to the ecosystem by adding them and opening a pull request!

```bash
$ taito install github.com/lackeyjb/playwright-skill@v4.1.0

  ⚠ No taito.spec found in lackeyjb/playwright-skill — consider adding 
    taito.spec files and opening a pull request!

  Continuing on the assumption that the repo contains 'skills' or 'agents' 
  directories.

  Please select skills/agents to install from 
  'github.com/lackeyjb/playwright-skill@v4.1.0':

  > [x] playwright-skill - skill

  (Press Space to toggle, Enter to confirm)

```

### From OCI repositories

taito mainly supports installing from OCI packages, for which you need to package your skills/agents using `taito package` command and publish to an OCI registry (can be public or private). For this check the `taito init` and `taito package` commands below.

```bash
$ taito install registry.gitlab.com/skill-harbor/infrastructure/test-only-one:v1.0.0

  Please select skills/agents to install from 'test':

  > [x] doc-generator      - skill
    [x] git-commit-helper  - skill
    [x] devops-agent       - agent

  (Press Space to toggle, Enter to confirm)
```

This allows you to easily share your skills and agents with others by simply sharing the OCI package. Its fully compatible with any OCI registry. Making it easy to version, distribute and manage your skills and agents in a standardized way. This is also the preferred way in corporate environments.

## taito list

The `list` command is used to display all skills, agents, and bundles currently installed across your configured AI coding tools. Packages installed via a bundle are grouped under their parent bundle in a tree structure.

```bash
$ taito list
```

Example output:

```bash
$ taito list
ID              NAME                TYPE    VERSION  TOOL         SOURCE
ffe390a1        devops-bundle       bundle  v1.0.0   Claude Code  ghcr.io/org/devops-bundle:v1.0.0
 ├─ a1b2c3d4    git-helper          skill   v1.0.0   Claude Code  
 └─ e5f6g7h8    docker-agent        agent   v1.0.0   Claude Code  
9402ea07        playwright-skill    skill   v4.1.0   Cursor       github.com/lackeyjb/playwright-skill@v4.1.0
```

## taito uninstall

The `uninstall` command is used to remove a skill, agent, or bundle by its ID from all configured AI coding tools. If a bundle is removed, all of its child skills and agents are also uninstalled. You can also use the alias `rm`.

```bash
$ taito uninstall <id>
```

<details>
  <summary>Example output:</summary>

```bash
$ taito uninstall ffe390a1
    ✓ Uninstalled bundle 'devops-bundle' (ffe390a1)
    ✓ Uninstalled child skill 'git-helper' (a1b2c3d4)
    ✓ Uninstalled child agent 'docker-agent' (e5f6g7h8)
    Successfully removed from Claude Code.
```
</details>

## taito check

The `check` command is used to validate a `taito.spec` manifest file according to the specification. It flags hard errors (like missing or invalid types) as failures and reports other issues as warnings.

```bash
$ taito check [path]
```

<details>
  <summary>Example output:</summary>

```bash
$ taito check ./skills/git-helper
    ✓ Validating taito.spec...
    ⚠ Warning: missing 'description' field
    ✓ Spec is valid!
```
</details>

## taito init

The `init` command launches an interactive wizard to help you generate a valid `taito.spec` manifest file in the current directory for a skill, agent, or bundle.

```bash
$ taito init
```

<details>
  <summary>Example output:</summary>

```bash
$ taito init
Initialize a new taito.spec

Select package type:

  skill
> agent
  bundle

↑/↓: navigate  enter: select

Type: agent

Package name (e.g. my-skill):
my-cool-agent

enter: confirm

Type: agent
Name: my-cool-agent

Description (optional):
An agent that does cool things.

enter: finish

    ✓ Successfully created taito.spec for agent 'my-cool-agent'
```

If you select `bundle`, the generated `taito.spec` will look like this:

```json
{
  "type": "bundle",
  "name": "my-bundle",
  "includes": [
    "<path ref to your skills / agents>"
  ]
}
```
</details>

## taito package

The `package` command is used to package a skill, agent, or bundle into an OCI artifact for publication and distribution. It automatically loads and validates the `taito.spec` file and stores the artifact in the cache directory by default.
Note that you will need to create a `taito.spec` file first using `taito init` before.

```bash
$ taito package [reference]
```

<details>
  <summary>Example output:</summary>

```bash
$ taito package ghcr.io/org/my-skill:1.0.0 --spec=./my-skill
    ✓ Validating taito.spec...
    ✓ Building OCI layout...
    ✓ Artifact packaged successfully!
    Digest: sha256:a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2
    Saved to cache: ~/.cache/taito/packages/ghcr.io/org/my-skill:1.0.0
```
</details>

## taito login

The `login` command is used to authenticate to an OCI registry and safely store credentials using native credential helpers (like macOS Keychain or Windows Credential Manager) for pushing and pulling packages.

```bash
$ taito login <registry>
```

<details>
  <summary>Example output:</summary>

```bash
$ taito login ghcr.io
Username: myuser
Password: [hidden]
    ✓ Authenticated securely with ghcr.io
    Credentials saved to native keychain.
```
</details>

## taito logout

The `logout` command is used to remove stored credentials for an OCI registry from your native credential helper.

```bash
$ taito logout <registry>
```

<details>
  <summary>Example output:</summary>

```
$ taito logout ghcr.io
    ✓ Successfully logged out of ghcr.io
    Credentials removed from native keychain.
```
</details>

## taito push

The `push` command is used to push a local OCI layout (previously created by `taito package`) to a remote OCI registry. You must be logged in first.

```bash
$ taito push <reference>
```

<details>
  <summary>Example output:</summary>

```bash
$ taito push ghcr.io/org/my-skill:1.0.0
    ⠋ Pushing layer sha256:a1b2c3d4...
    ⠙ Pushing config sha256:e5f6g7h8...
    ✓ Pushed successfully!
    Reference: ghcr.io/org/my-skill:1.0.0
    Digest: sha256:9f8e7d6c5b4a3x2y1z...
```
</details>

## taito pull

The `pull` command is used to fetch a taito artifact from a remote OCI registry into a local OCI layout. It automatically validates the artifact before committing it to disk.

```bash
$ taito pull <reference>
```

<details>
  <summary>Example output:</summary>

```bash
$ taito pull ghcr.io/org/my-skill:1.0.0
    ⠋ Pulling layer sha256:a1b2c3d4...
    ✓ Validating artifact...
    ✓ Pulled successfully!
    Saved to: ~/.cache/taito/packages/ghcr.io/org/my-skill:1.0.0
```
</details>

## taito prune

The `prune` command is used to remove all cached artifacts from the taito packages cache directory. You can use `--dry-run` to see what would be removed without actually deleting anything.

```bash
$ taito prune
```

<details>
  <summary>Example output:</summary>

```bash
$ taito prune
    Deleted ~/.cache/taito/packages/ghcr.io/org/my-skill:1.0.0
    Deleted ~/.cache/taito/packages/ghcr.io/org/devops-bundle:v1.0.0
    ✓ Cache cleared successfully (Total reclaimed: 14.2 MB).
```
</details>
