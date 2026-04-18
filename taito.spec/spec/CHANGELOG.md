# taito.spec Changelog

All notable changes to the taito.spec specification will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the specification versioning adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-03-29

### Added

- Initial draft of the taito.spec specification
- `taito.spec` manifest format with required fields: `taitoVersion`, `type`, `name`, `version`, `description`
- Three types: `skill` (standalone), `agent` (composable, with dependencies), and `bundle` (groups multiple skills/agents in a repo)
- `author`, `license`, `keywords`, `prompt`, `repository`, `homepage` optional fields
- `dependencies` field for agents to reference other skills by name + semver range
- `members` field for bundles to reference skills/agents by relative directory path
- `config` field for declaring configuration requirements with types, defaults, secrets, and enums
- `compatibility` field for runtime and model hints
- `x-` prefixed custom extension fields for forward-compatible extensibility
- JSON Schema (Draft 2020-12) for machine validation
- Prompt file convention (`SKILL.md`)
- Four reference examples: simple-skill, agent-with-skills, skill-with-config, bundle-repo
