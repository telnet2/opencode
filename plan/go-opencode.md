# Go-OpenCode vs TypeScript OpenCode: Configuration & CLI Compatibility Plan

## Executive Summary

The Go implementation (`go-opencode`) and TypeScript implementation (`packages/opencode`) have significant differences in configuration format, CLI interface, and feature support. This document outlines the gaps and provides a plan for achieving compatibility.

---

## 1. Configuration File Comparison

### 1.1 File Format & Location

| Aspect | TypeScript | Go | Status |
|--------|------------|-----|--------|
| File format | JSONC (JSON with comments) | JSONC | Compatible |
| Config filename | `opencode.json`, `opencode.jsonc` | `opencode.json`, `opencode.jsonc` | Compatible |
| Global config | `~/.opencode/config.json` | `~/.config/opencode/opencode.json` | **Different** |
| Project config | `.opencode/opencode.json` | `.opencode/opencode.json` | Compatible |
| Interpolation | `{env:VAR}`, `{file:path}` | None | **Missing in Go** |
| Schema reference | `$schema` field | None | **Missing in Go** |

### 1.2 Root Configuration Fields

| Field | TypeScript | Go | Notes |
|-------|------------|-----|-------|
| `model` | `string` | `string` | Compatible |
| `small_model` | `string` | `string` | Compatible (Go uses `SmallModel`) |
| `username` | `string` | N/A | **Missing in Go** |
| `theme` | `string` | N/A | **Missing in Go** (TUI only) |
| `share` | `"manual" \| "auto" \| "disabled"` | N/A | **Missing in Go** |
| `autoupdate` | `boolean \| "notify"` | N/A | **Missing in Go** |
| `plugin` | `string[]` | N/A | **Missing in Go** |
| `tools` | `Record<string, boolean>` | N/A | **Missing in Go** (only in agent) |
| `keybinds` | `KeybindsConfig` | N/A | **Missing in Go** (TUI only) |
| `tui` | `TUIConfig` | N/A | **Missing in Go** (TUI only) |
| `watcher` | `{ ignore: string[] }` | `WatcherConfig` | Compatible |
| `snapshot` | `boolean` | N/A | **Missing in Go** |
| `promptVariables` | `Record<string, string>` | N/A | **Missing in Go** |
| `instructions` | `string[]` | N/A | **Missing in Go** |
| `provider` | `Record<string, ProviderConfig>` | `map[string]ProviderConfig` | Partial |
| `mcp` | `Record<string, MCPConfig>` | N/A | **Missing in Go** |
| `formatter` | `boolean \| Record<string, FormatterConfig>` | N/A | **Missing in Go** |
| `lsp` | `boolean \| Record<string, LSPConfig>` | `*LSPConfig` | Different structure |
| `agent` | `Record<string, AgentConfig>` | `map[string]AgentConfig` | Partial |
| `command` | `Record<string, CommandConfig>` | N/A | **Missing in Go** |
| `permission` | `PermissionConfig` | N/A | **Missing in Go** (only in agent) |
| `enterprise` | `{ url: string }` | N/A | **Missing in Go** |
| `experimental` | `ExperimentalConfig` | `*ExperimentalConfig` | Different |

### 1.3 Provider Configuration

| Field | TypeScript | Go | Notes |
|-------|------------|-----|-------|
| `options.apiKey` | `string` | `apiKey` | Different casing |
| `options.baseURL` | `string` | `baseUrl` | Different casing |
| `options.timeout` | `number \| false` | N/A | **Missing in Go** |
| `whitelist` | `string[]` | N/A | **Missing in Go** |
| `blacklist` | `string[]` | N/A | **Missing in Go** |
| `models` | `Record<string, ModelInfo>` | N/A | **Missing in Go** |
| `disable` | N/A | `disable` | **Go only** |

### 1.4 Agent Configuration

| Field | TypeScript | Go | Notes |
|-------|------------|-----|-------|
| `model` | `string` | `*ModelRef` | Different type |
| `temperature` | `number` | `float64` | Compatible |
| `top_p` | `number` | `float64` | Field name: Go uses `TopP` |
| `prompt` | `string` | `string` | Compatible |
| `tools` | `Record<string, boolean>` | `map[string]bool` | Compatible |
| `disable` | `boolean` | N/A | **Missing in Go** |
| `description` | `string` | `string` | Compatible |
| `mode` | `"subagent" \| "primary" \| "all"` | `string` | Compatible |
| `color` | `string` | `string` | Compatible |
| `permission` | `PermissionConfig` | `AgentPermission` | Different structure |

### 1.5 Permission Configuration

| Field | TypeScript | Go | Notes |
|-------|------------|-----|-------|
| `edit` | `Permission` | `string` | Compatible |
| `bash` | `Permission \| Record<string, Permission>` | `interface{}` | TypeScript supports per-command |
| `webfetch` | `Permission` | `string` | Compatible |
| `doom_loop` | `Permission` | `string` | Compatible |
| `external_directory` | `Permission` | `string` | Compatible |

---

## 2. CLI Comparison

### 2.1 Entry Point & Architecture

| Aspect | TypeScript | Go | Notes |
|--------|------------|-----|-------|
| Binary name | `opencode` | `opencode-server` | **Different** |
| CLI framework | Yargs | flag (stdlib) | Different |
| Command structure | Subcommands | Flags only | **Go lacks subcommands** |

### 2.2 CLI Flags

**TypeScript Global Flags:**
```bash
--print-logs              # Print logs to stderr
--log-level               # DEBUG|INFO|WARN|ERROR
--help, -h
--version, -v
```

**Go Global Flags:**
```bash
-port                     # Server port (default: 8080)
-directory                # Working directory
-version                  # Print version
```

### 2.3 Commands Comparison

| TypeScript Command | Go Equivalent | Status |
|--------------------|---------------|--------|
| `opencode run [message]` | N/A | **Missing** |
| `opencode spawn [project]` | N/A | **Missing** |
| `opencode attach <url>` | N/A | **Missing** |
| `opencode serve` | `opencode-server` | **Partial** (Go is serve-only) |
| `opencode web` | N/A | **Missing** |
| `opencode acp` | N/A | **Missing** |
| `opencode models [provider]` | N/A | **Missing** |
| `opencode auth` | N/A | **Missing** |
| `opencode agent` | N/A | **Missing** |
| `opencode upgrade` | N/A | **Missing** |
| `opencode prompts` | N/A | **Missing** |
| `opencode export` | N/A | **Missing** |
| `opencode import` | N/A | **Missing** |
| `opencode stats` | N/A | **Missing** |
| `opencode mcp` | N/A | **Missing** |
| `opencode pr` | N/A | **Missing** |
| `opencode github` | N/A | **Missing** |
| `opencode debug` | N/A | **Missing** |
| `opencode generate` | N/A | **Missing** |

### 2.4 Run Command Options (Critical Gap)

TypeScript `opencode run` options that Go lacks:
```bash
--command, -c            # Command to run
--continue, -c           # Continue last session
--session, -s            # Session ID to continue
--share                  # Share session
--model, -m              # Model override
--agent                  # Agent to use
--format                 # Output format (default|json)
--file, -f               # Attach files
--title                  # Session title
--attach                 # Attach to server URL
--port                   # Port for local server
--prompt                 # Custom prompt
--prompt-file            # Prompt from file
--prompt-inline          # Inline prompt
```

---

## 3. Environment Variables

### 3.1 Comparison Table

| TypeScript Variable | Go Equivalent | Status |
|--------------------|---------------|--------|
| `OPENCODE_AUTO_SHARE` | N/A | **Missing** |
| `OPENCODE_CONFIG` | N/A | **Missing** |
| `OPENCODE_CONFIG_DIR` | N/A | **Missing** |
| `OPENCODE_CONFIG_CONTENT` | N/A | **Missing** |
| `OPENCODE_DISABLE_AUTOUPDATE` | N/A | **Missing** |
| `OPENCODE_PERMISSION` | N/A | **Missing** |
| `OPENCODE_DISABLE_LSP_DOWNLOAD` | N/A | **Missing** |
| `OPENCODE_ENABLE_EXPERIMENTAL_MODELS` | N/A | **Missing** |
| `OPENCODE_EXPERIMENTAL` | N/A | **Missing** |
| `OPENCODE_MODEL` | `OPENCODE_MODEL` | **Compatible** |
| `OPENCODE_SMALL_MODEL` | `OPENCODE_SMALL_MODEL` | **Compatible** |
| `ANTHROPIC_API_KEY` | `ANTHROPIC_API_KEY` | **Compatible** |
| `OPENAI_API_KEY` | `OPENAI_API_KEY` | **Compatible** |
| `GOOGLE_API_KEY` | `GOOGLE_API_KEY` | **Compatible** |
| `XDG_*` | `XDG_*` | **Compatible** |

---

## 4. Compatibility Plan

### Phase 1: Configuration Compatibility (High Priority)

#### 1.1 Align Config Field Names
- [ ] Rename Go `SmallModel` to `small_model` in JSON tags
- [ ] Rename Go `TopP` to `top_p` in JSON tags
- [ ] Update Go provider config to use `apiKey` and `baseURL` (camelCase for JSON)

#### 1.2 Support Global Config Location
- [ ] Add support for `~/.opencode/config.json` as primary location
- [ ] Keep `~/.config/opencode/` as fallback (XDG compliance)
- [ ] Add `OPENCODE_CONFIG` and `OPENCODE_CONFIG_DIR` env var support

#### 1.3 Add Missing Config Fields
- [ ] Add `username` field
- [ ] Add `instructions` array field
- [ ] Add `promptVariables` map field
- [ ] Add global `tools` enable/disable map
- [ ] Add global `permission` config (not just per-agent)
- [ ] Add provider `whitelist`/`blacklist` support
- [ ] Add provider `timeout` support

#### 1.4 Add Interpolation Support
- [ ] Implement `{env:VAR_NAME}` interpolation in config values
- [ ] Implement `{file:path}` interpolation for file includes

### Phase 2: CLI Compatibility (High Priority)

#### 2.1 Restructure CLI with Subcommands
- [ ] Replace flag-based CLI with Cobra for subcommand support
- [ ] Rename binary from `opencode-server` to `opencode`
- [ ] Add `serve` as default subcommand for backwards compatibility

#### 2.2 Implement Core Commands
- [ ] `opencode run` - Start interactive session
  - [ ] `--model, -m` - Model override
  - [ ] `--agent` - Agent selection
  - [ ] `--continue, -c` - Continue last session
  - [ ] `--session, -s` - Continue specific session
  - [ ] `--file, -f` - Attach files
  - [ ] `--prompt` - Custom prompt
  - [ ] `--format` - Output format
- [ ] `opencode serve` - Start headless server (current functionality)
  - [ ] `--port, -p` - Port
  - [ ] `--hostname` - Hostname
- [ ] `opencode models` - List available models
  - [ ] `--verbose` - Show pricing
  - [ ] `--refresh` - Refresh cache
- [ ] `opencode auth` - Auth management
  - [ ] `login` - Login to provider
  - [ ] `logout` - Logout from provider
  - [ ] `list` - List providers

#### 2.3 Implement Secondary Commands
- [ ] `opencode agent` - Agent management (list, create, delete)
- [ ] `opencode export` - Export session
- [ ] `opencode import` - Import session
- [ ] `opencode stats` - Usage statistics
- [ ] `opencode debug` - Debug utilities

### Phase 3: Feature Parity (Medium Priority)

#### 3.1 MCP Support
- [ ] Add MCP config parsing
- [ ] Implement MCP local server support
- [ ] Implement MCP remote server support

#### 3.2 Custom Commands
- [ ] Add command config parsing
- [ ] Implement `.opencode/command/` directory scanning
- [ ] Support markdown-based command definitions

#### 3.3 Formatter Integration
- [ ] Add formatter config support
- [ ] Implement file-edited hooks

#### 3.4 Sharing
- [ ] Add `share` config option
- [ ] Implement session sharing API

### Phase 4: Advanced Features (Lower Priority)

#### 4.1 GitHub Integration
- [ ] `opencode pr` command
- [ ] `opencode github` command

#### 4.2 Update System
- [ ] `opencode upgrade` command
- [ ] Auto-update notification

#### 4.3 Prompt Management
- [ ] `opencode prompts` command
- [ ] Custom prompt template support

---

## 5. Migration Guide for Users

### Config File Migration

**TypeScript format:**
```json
{
  "model": "anthropic/claude-sonnet-4-20250514",
  "small_model": "anthropic/claude-3-5-haiku-20241022",
  "provider": {
    "anthropic": {
      "options": {
        "apiKey": "sk-ant-..."
      }
    }
  },
  "agent": {
    "coder": {
      "tools": { "bash": true },
      "permission": { "edit": "allow" }
    }
  }
}
```

**Current Go format (needs alignment):**
```json
{
  "model": "anthropic/claude-sonnet-4-20250514",
  "small_model": "anthropic/claude-3-5-haiku-20241022",
  "provider": {
    "anthropic": {
      "apiKey": "sk-ant-..."
    }
  },
  "agent": {
    "coder": {
      "tools": { "bash": true },
      "permission": { "edit": "allow" }
    }
  }
}
```

**Key differences to address:**
1. Provider options nesting (`options.apiKey` vs `apiKey`)
2. Config file location (`~/.opencode/` vs `~/.config/opencode/`)
3. Missing interpolation support

---

## 6. Implementation Priorities

### Must Have (v1.0)
1. Binary rename to `opencode`
2. `opencode run` command with basic flags
3. `opencode serve` command
4. `opencode models` command
5. Config field name alignment
6. Support for both config locations

### Should Have (v1.1)
1. `opencode auth` command
2. `opencode agent` command
3. Environment variable interpolation
4. `OPENCODE_CONFIG` env var
5. Provider whitelist/blacklist

### Nice to Have (v1.2+)
1. `opencode export/import`
2. `opencode stats`
3. MCP support
4. Custom commands
5. File interpolation

---

## 7. Testing Strategy

### Config Compatibility Tests
- [ ] Load TypeScript config files in Go
- [ ] Verify all fields parsed correctly
- [ ] Test config merge precedence
- [ ] Test environment variable overrides

### CLI Compatibility Tests
- [ ] Verify command parsing matches TypeScript
- [ ] Test flag aliases (-m, --model)
- [ ] Test subcommand routing
- [ ] Test help output format

### Integration Tests
- [ ] Run same session with both implementations
- [ ] Verify API responses match
- [ ] Test model switching
- [ ] Test agent configuration

---

## 8. Files to Modify

### Go Files
- `cmd/opencode-server/main.go` -> `cmd/opencode/main.go`
- `pkg/types/config.go` - Add missing fields, fix JSON tags
- `internal/config/config.go` - Add interpolation, new locations
- New: `cmd/opencode/commands/*.go` - Subcommand implementations

### New Dependencies (Go)
- `github.com/spf13/cobra` - CLI framework with subcommands
- `github.com/spf13/viper` (optional) - Config management

---

## 9. Timeline Estimate

| Phase | Scope | Estimate |
|-------|-------|----------|
| Phase 1 | Config Compatibility | 2-3 days |
| Phase 2 | CLI Compatibility | 3-5 days |
| Phase 3 | Feature Parity | 5-7 days |
| Phase 4 | Advanced Features | Ongoing |

**Total for v1.0 compatibility: ~1-2 weeks**
