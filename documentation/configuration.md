# Configuration (config.json)

The Taito CLI uses a `config.json` file to manage user preferences, such as the target AI coding tools for installations and custom cache directories. 

You can easily generate and update this configuration interactively by running:
```bash
$ taito setup
```

## Configuration File Locations
Depending on your operating system, the `config.json` file is located at:
* **Linux**: `~/.config/taito/config.json`
* **macOS**: `~/.config/taito/config.json`

---

## Example

```json
{
  "storage": {
    "cacheDir": "/custom/path/to/cache"
  },
  "tools": [
    {
      "name": "claude-code",
      "path": "~/.claude"
    },
    {
      "name": "cursor"
    }
  ]
}
```

---

## Fields Reference

### storage
**Type:** Object | **Required:** No
Contains storage-related settings for the Taito CLI.

### storage.cacheDir
**Type:** String | **Required:** No
Overrides the default cache directory where pulled OCI artifacts are stored. If omitted, the platform's default cache directory is used (e.g., `~/.cache/taito` on Linux, `~/Library/Caches/taito` on macOS).

### tools
**Type:** Array of Objects | **Required:** No
A list of configured AI coding tools where skills and agents should be installed.

### tools[].name
**Type:** String | **Required:** Yes (if inside a tool object)
The identifier of the AI coding tool. Supported known tools include: `"cursor"`, `"windsurf"`, `"claude-code"`, `"opencode"`, and `"copilot"`.

### tools[].path
**Type:** String | **Required:** No
Overrides the default configuration directory path for the specified tool. Path expansion using `~/` is supported. If omitted, Taito resolves the standard platform-specific default path for that tool.
