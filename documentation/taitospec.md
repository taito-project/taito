# taito.spec

The `taito.spec` file is the JSON manifest that defines a Taito package. Every skill, agent, or bundle must contain a valid `taito.spec` file in its root directory to be packaged, distributed, and installed via the Taito CLI.

You can easily generate a basic `taito.spec` file in your project by running:
```bash
$ taito init
```

---

## Examples

### Skill or Agent
```json
{
  "taitoVersion": "0.1.0",
  "type": "skill",
  "name": "git-commit-helper",
  "version": "1.0.0",
  "description": "Helps write conventional git commit messages.",
  "author": {
    "name": "Jane Doe",
    "email": "jane@example.com"
  },
  "license": "MIT",
  "keywords": ["git", "productivity"]
}
```

### Bundle
```json
{
  "type": "bundle",
  "name": "devops-toolkit",
  "version": "2.1.0",
  "description": "A collection of agents and skills for DevOps workflows.",
  "includes": [
    "./skills/docker-deploy",
    "./agents/infrastructure-agent"
  ]
}
```

---

## Fields Reference

### type
**Type:** String | **Required:** Yes
The package type. Must be exactly `"skill"`, `"agent"`, or `"bundle"`.

### name
**Type:** String | **Required:** Yes
The unique name of the package. Must match `^[a-z0-9][a-z0-9_-]*$` and be a maximum of 128 characters.

### version
**Type:** String | **Required:** No
The semantic version of the package (e.g., `"1.0.0"`). Must be valid semver.

### taitoVersion
**Type:** String | **Required:** No
The minimum required version of the Taito CLI to run this package (e.g., `"0.1.0"`). Must be valid semver.

### description
**Type:** String | **Required:** No
A short summary of what the package does. Maximum 500 characters.

### source
**Type:** String | **Required:** No
The URL to the source repository or homepage.

### author
**Type:** Object | **Required:** No
The author information. If provided, the `name` property is **required**. Can optionally include `email` and `url`.

### license
**Type:** String | **Required:** No
The license under which the package is distributed (e.g., `"MIT"`, `"Apache-2.0"`).

### keywords
**Type:** Array | **Required:** No
Array of strings for discoverability. Max 20 items. Each keyword must match `^[a-z0-9][a-z0-9_-]*$` and be a max of 64 characters.

### includes
**Type:** Array | **Required:** Bundles Only
Array of relative paths pointing to child skills/agents. See the [Bundles](#how-bundles-work) section below.

---

## How Bundles Work

A **bundle** is a special type of package designed to group multiple skills and agents together into a single distributable artifact.

Instead of containing executable logic itself, a bundle acts as a parent directory containing child packages. When a user installs a bundle, Taito reads the bundle's `taito.spec` file and automatically installs all the child packages referenced in the `includes` array.

### Rules for Bundles (`includes`)
1. The `includes` field is **only** valid when `"type": "bundle"`.
2. Every item in the `includes` array must be a **relative path** pointing to a subdirectory within the bundle. (e.g., `"./agents/my-agent"`).
3. Absolute paths (e.g., `"/home/user/agent"`) are strictly prohibited.
4. Path traversal above the bundle's root directory using `..` is strictly prohibited.
5. The target directory of an `includes` path **must** contain its own valid `taito.spec` file (which defines it as a `skill` or `agent`).