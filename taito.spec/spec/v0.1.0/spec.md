# taito.spec -- Specification v0.1.0

**Status:** Implemented in taito-cli v0.1.0 
**Version:** 0.1.0
**Date:** 2025-03-29

## Table of Contents

1. [Introduction](#1-introduction)
2. [Terminology](#2-terminology)
3. [File Structure](#3-file-structure)
4. [Manifest Format](#4-manifest-format)
5. [Skill Type](#5-skill-type)
6. [Agent Type](#6-agent-type)
7. [Bundle Type](#7-bundle-type)
8. [Versioning](#8-versioning)

---

## 1. Introduction

taito.spec defines a standard manifest format (`taito.spec`) for describing AI agent skills and agents. The specification is:

- **Agent-runtime agnostic** -- It does not prescribe how skills are executed, only how they are described.
- **Minimal** -- It defines the smallest useful set of fields needed for discovery, validation, and composition.
- **Composable** -- Agents are skills that reference other skills, enabling modular agent architectures.
- **Machine-readable** -- The manifest is JSON, validated by a JSON Schema.

The goal is to provide a common language for the AI agent ecosystem so that skills can be shared, validated, and composed regardless of the underlying framework.

### 1.1 Scope

This specification covers:

- The structure and contents of a `taito.spec` manifest file
- The convention for prompt/instruction files (`SKILL.md`)
- How bundles group skills and agents through the `includes` field

This specification does NOT cover:

- How skills are packaged into archives or containers (deferred to future tooling)
- How skills are distributed or installed (registry protocols, package managers)
- How skills are executed at runtime (framework-specific)

### 1.2 Conformance

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

A conforming skill, agent, or bundle MUST include a valid `taito.spec` file that passes validation against the JSON Schema defined in this specification.

---

## 2. Terminology

| Term | Definition |
|------|------------|
| **Skill** | A self-contained unit of AI agent capability, described by a `taito.spec` manifest and an accompanying prompt file. A skill provides instructions, context, or behavior that an agent can use. |
| **Agent** | A composable skill (type `"agent"`) that references other skills as dependencies. An agent combines multiple skills with its own prompt to form a complete agent persona. |
| **Bundle** | A root-level manifest (type `"bundle"`) that groups multiple skills and/or agents within a single repository or directory tree. A bundle does not define behavior itself -- it uses `includes` to reference its members. |
| **Manifest** | The `taito.spec` file that describes a skill, agent, or bundle. |

---

## 3. File Structure

### 3.1 Required Files

A conforming skill MUST contain:

- **`taito.spec`** -- The manifest file (see [Section 4](#4-manifest-format))

### 3.2 Recommended Files

A conforming skill SHOULD contain:

- **`SKILL.md`** -- The prompt/instruction file containing the behavior of the skill.

### 3.3 Directory Layout

A skill or agent is a directory. The `taito.spec` file MUST be at the root of the skill directory. All paths in the manifest are relative to the directory containing `taito.spec`.

```
my-skill/
  taito.spec          # REQUIRED
  SKILL.md            # RECOMMENDED
  ... other files ... # OPTIONAL
```

### 3.4 Bundle Directory Layout

A bundle is a directory that contains a root `taito.spec` with `"type": "bundle"` and one or more member directories, each containing their own `taito.spec`. See [Section 7](#7-bundle-type) for full details.

```
my-repo/
  taito.spec                # Bundle manifest (type: "bundle")
  skills/
    git-helper/
      taito.spec            # Skill manifest (type: "skill")
      SKILL.md
    code-reviewer/
      taito.spec            # Agent manifest (type: "agent")
      SKILL.md
```

### 3.5 Prompt File Convention

The prompt file MUST be a markdown file (`.md` extension). The RECOMMENDED and standard filename is `SKILL.md`. Implementations SHOULD look for this file by default.

The prompt file contains the instructions, system prompt, behavioral guidelines, or any other textual content that defines what the skill does and how it should behave. The exact interpretation of the prompt file is left to the consuming runtime.

---

## 4. Manifest Format

The manifest is a file named `taito.spec`. Despite the extension, the file MUST contain valid JSON and MUST conform to the JSON Schema defined by this specification for the declared `taitoVersion`.

### 4.1 Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | `string` | The type of this package. MUST be `"skill"`, `"agent"`, or `"bundle"`. |
| `name` | `string` | The name of the skill or agent. MUST be a lowercase string containing only alphanumeric characters, hyphens, and underscores. MUST be between 1 and 128 characters. |

### 4.2 Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | A short, human-readable description of what this skill or agent does. MUST be between 1 and 500 characters. |
| `taitoVersion` | `string` | The version of the taito.spec this manifest conforms to. MUST be a valid semantic version string. |
| `source` | `string` | The source repository or location (e.g., `"github.com/larszi/skill"`). |
| `author` | `object` | The author of the skill. See [Section 4.3](#43-author-object). |
| `license` | `string` | An SPDX license identifier (e.g., `"Apache-2.0"`, `"MIT"`). |
| `keywords` | `array` | An array of strings for categorization and discovery. Each keyword MUST be lowercase and between 1 and 64 characters. Maximum 20 keywords. |
| `includes` | `array` | An array of relative paths to `taito.spec` files within this repository. Only valid when `type` is `"bundle"`. See [Section 7](#7-bundle-type). |


### 4.3 Author Object

The `author` object has the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | REQUIRED | The author's name or organization name. |
| `url` | `string` | OPTIONAL | A URL associated with the author (website, GitHub profile, etc.). |
| `email` | `string` | OPTIONAL | The author's contact email address. |

### 4.4 Full Example

```json
{
  "taitoVersion": "0.1.0",
  "type": "skill",
  "name": "git-commit-helper",
  "version": "1.2.0",
  "description": "Helps write conventional commit messages by analyzing staged changes",
  "author": {
    "name": "Taito",
    "url": "https://github.com/taito-project"
  },
  "license": "Apache-2.0",
  "keywords": ["git", "commit", "conventional-commits", "developer-tools"]
}
```

---

## 5. Skill Type

A skill (`"type": "skill"`) is a self-contained unit of capability. It represents a single area of expertise, tool, or behavior that can be used by an agent or loaded directly by a runtime.


---

## 6. Agent Type

An agent (`"type": "agent"`) is a composable skill that references other skills. An agent combines multiple skills with its own prompt/persona to form a more capable entity.


### 6.2 Version Range Syntax

The following version range syntaxes MUST be supported:

| Syntax | Description | Example |
|--------|-------------|---------|
| `1.0.0` | Exact version | Matches only `1.0.0` |
| `*` | Any version | Matches any version |

Version format MUST follow Semantic Versioning 2.0.0. and only supports major.minor.patch (pre-release and build metadata are not supported).


---

## 7. Bundle Type

A bundle (`"type": "bundle"`) is a root-level manifest that groups multiple skills and/or agents within a single repository or directory tree. Bundles are useful when a repository contains multiple related skills that should be discovered and managed together.

### 7.1 Purpose

Bundles solve the monorepo problem: when a single repository contains multiple skills and/or agents, a bundle manifest at the root provides a single entry point for tooling to discover all the included packages. Without a bundle, tools would need to recursively scan the directory tree to find individual `taito.spec` files.

### 7.2 Includes

The `includes` field is an array of relative paths pointing to `taito.spec` files within the repository. Each path MUST point to a valid `taito.spec` file of type `"skill"` or `"agent"`.

### 7.3 Bundle Metadata

A bundle still carries its own `name`, `version`, `description`, and optional `author`, `license`, `keywords` fields. These describe the collection as a whole, not any individual member.

### 7.4 Nested Bundles

Bundles MUST NOT reference other bundles. 

### 7.5 Full Example

```
devtools/
  taito.spec                        # Bundle manifest
  skills/
    git-commit-helper/
      taito.spec                    # type: "skill"
      SKILL.md
    security-scanner/
      taito.spec                    # type: "skill"
      SKILL.md
  agents/
    code-review-agent/
      taito.spec                    # type: "agent"
      SKILL.md
```

**Root `taito.spec`:**

```json
{
  "taitoVersion": "0.1.0",
  "type": "bundle",
  "name": "devtools",
  "description": "A collection of developer productivity skills and agents",
  "author": {
    "name": "Taito",
    "url": "https://github.com/taito-project"
  },
  "license": "Apache-2.0",
  "keywords": ["developer-tools", "collection"],
  "includes": [
    "./skills/git-commit-helper/taito.spec",
    "./skills/security-scanner/taito.spec",
    "./agents/code-review-agent/taito.spec"
  ]
}
```

---


## Appendix A: JSON Schema Reference

The machine-readable JSON Schema for this specification version is available at:

- `spec/v0.1.0/taito.schema.json`

This schema can be used with any JSON Schema Draft 2020-12 compliant validator.
