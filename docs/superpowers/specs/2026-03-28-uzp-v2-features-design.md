# UZP CLI v2 Feature Design

## Overview

Three new features for UZP CLI to evolve from a local secrets store into a secrets infrastructure layer for both humans and AI agents.

**Approach**: CLI-first. Each phase ships independently with immediate value.

| Phase | Feature      | Purpose                                                         | Effort   |
| ----- | ------------ | --------------------------------------------------------------- | -------- |
| 1     | `uzp run`    | Inject secrets into process environment without plaintext files | 2-3 days |
| 2     | `uzp import` | Import secrets from `.env` files to reduce migration barrier    | 1-2 days |
| 3     | `uzp mcp`    | Read-only MCP server with scope-based access for AI agents      | 3-5 days |

## Dropped Feature: `uzp sync`

Encrypted team sharing via shared passphrase was considered and rejected. A weak or leaked passphrase exposes all secrets with no revocation path. If team sharing is needed, it belongs in a separate product with proper key exchange.

---

## Phase 1: `uzp run`

### Purpose

Inject vault secrets into a child process environment. Replaces the `uzp inject > .env && source .env && command` workflow with a single command. Secrets never touch disk.

### Usage

```bash
# Single project
uzp run -p myapp -- npm start

# Multiple projects (last-wins for duplicate keys)
uzp run -p shared -p myapp -- docker compose up

# AI agents use the same interface
# Claude Code: uzp run -p backend -- go run .
```

### Implementation

1. Parse flags and split on `--` to separate uzp args from child command.
2. Call `ensureVaultUnlocked()` to prompt for master password if needed.
3. Load secrets from specified projects. For multiple `-p` flags, merge left-to-right (later project overrides on key collision).
4. Convert key names via `convertToEnvKey()` (reuse existing logic from `inject.go`).
5. Build environment: current `os.Environ()` + secrets overlay.
6. On Unix: `syscall.Exec(binary, args, env)` — replaces the current process entirely. Parent process disappears, so secrets do not persist in parent `/proc/pid/environ`.
7. On Windows: `os/exec.Command` with `cmd.Env = env`, `cmd.Wait()`, then `os.Exit(cmd.ProcessState.ExitCode())`.

### New file

`cmd/run.go` — registered as `rootCmd.AddCommand(runCmd)`.

### Edge cases

- No `--` separator: print error with usage example.
- Key collision across projects: last `-p` flag wins (documented, deterministic).
- Child process fails: forward exit code to caller.
- Empty project: skip silently (no secrets injected from that project).
- Command not found: forward the exec error as-is.

### Security properties

- Secrets exist only in child process environment, never written to disk.
- `syscall.Exec` replaces the parent process, so no parent retains secrets in memory.
- Master password prompt goes to stderr, does not interfere with child process stdout/stdin.

---

## Phase 2: `uzp import`

### Purpose

Import secrets from `.env` files into the vault. Reduces migration barrier to a single command.

### Usage

```bash
# Import .env file into a project
uzp import .env --project myapp

# Interactive mode: read keys from file, prompt for each value
uzp import .env.example --project myapp --interactive

# Import from stdin
cat secrets.env | uzp import - --project backend
```

### Implementation

1. Read file (or stdin if path is `-`).
2. Parse line by line:
   - Skip empty lines and comments (`#`).
   - Extract `KEY=VALUE` pairs.
   - Handle double-quoted values (`KEY="value with spaces"`): unescape `\"`, `\\`, `\n`.
   - Handle single-quoted values (`KEY='literal $value'`): no escaping.
   - Handle unquoted values: trim trailing whitespace.
3. In `--interactive` mode: for each key found, prompt user to enter value via `term.ReadPassword()` (hidden input). Show the key name and default value (if any) for context.
4. Check for existing keys: skip by default, `--overwrite` flag to force update.
5. Call `vault.Add(project, key, value)` for each entry.
6. Report summary to stderr: `Imported N secrets into project 'myapp' (M skipped)`.

### New file

`cmd/import.go` — registered as `rootCmd.AddCommand(importCmd)`.

### .env parser

Implemented as `internal/dotenv/parser.go`. Handles the three value formats (double-quoted, single-quoted, unquoted). Does not support multiline heredoc syntax — only `\n` escape sequences within double-quoted values.

### Edge cases

- File not found: clear error message.
- Malformed lines (no `=`): skip with warning to stderr.
- Empty value: allowed (some env vars are legitimately empty).
- Key exceeds vault limits (255 bytes): reject with error.
- Duplicate keys in same file: last occurrence wins.
- `--interactive` with `-` (stdin): error, interactive requires a file.

### What we are NOT building

- Export to JSON/YAML/Docker formats — `uzp inject` already covers `.env` export.
- Import from cloud providers (AWS SSM, 1Password, etc.) — different scope.

---

## Phase 3: `uzp mcp` (MCP Server)

### Purpose

Expose vault secrets to AI agents (Claude Code, Codex, OpenCode) via the Model Context Protocol. Read-only with scope-based access control.

### Usage

```bash
# Start MCP server (stdio transport)
uzp mcp
```

AI agent configuration (Claude Code example):

```json
{
  "mcpServers": {
    "uzp": {
      "command": "uzp",
      "args": ["mcp"]
    }
  }
}
```

### MCP Tools

| Tool         | Description                        | Input                              | Output                                 |
| ------------ | ---------------------------------- | ---------------------------------- | -------------------------------------- |
| `uzp_get`    | Get a single secret value          | `{ project: string, key: string }` | `{ value: string }`                    |
| `uzp_list`   | List projects and keys (no values) | `{ project?: string }`             | `{ projects: { [name]: string[] } }`   |
| `uzp_search` | Search by keyword                  | `{ keyword: string }`              | `{ results: { [project]: string[] } }` |

Read-only only. No `add`, `update`, `delete`. Agent must instruct the user to run CLI commands for write operations.

### Scope-Based Access Control

Config file: `~/.uzp/access.json`

```json
{
  "default": "prompt",
  "rules": [
    { "project": "myapp", "access": "allow" },
    { "project": "production", "access": "prompt" },
    { "project": "infra/*", "access": "deny" }
  ]
}
```

Three access levels:

- **allow**: Agent reads freely, no user interaction.
- **prompt**: Each request triggers a confirmation prompt on stderr. User approves or denies.
- **deny**: Request rejected immediately. Agent receives an error message.

If `access.json` does not exist: default is `prompt` for all projects (safe default).

Glob patterns supported in project field (`*` matches any characters).

### Implementation

#### New files

- `cmd/mcp.go` — command entry point, starts MCP server.
- `internal/mcp/server.go` — MCP protocol handler (stdio JSON-RPC).
- `internal/mcp/tools.go` — tool implementations wrapping vault operations.
- `internal/mcp/access.go` — access control rule evaluation.

#### Protocol

- Transport: stdio (stdin/stdout JSON-RPC). Standard for Claude Code, Codex, and other CLI-based AI agents.
- MCP SDK: use `github.com/mark3labs/mcp-go` (Go MCP SDK) or implement minimal JSON-RPC handler directly (the protocol surface for tools-only is small).
- Server advertises tools via `tools/list`, handles `tools/call`.

#### Authentication

- MCP runs as a local subprocess — inherits the user's filesystem permissions.
- First tool call triggers `ensureVaultUnlocked()` on stderr.
- No network exposure, no API keys, no tokens.

### What we are NOT building (phase 1)

- Write operations — add after access control is battle-tested.
- HTTP/SSE transport — stdio is sufficient for all current AI CLI agents.
- Per-key access control — project-level granularity is enough.
- MCP Resources or Prompts — tools only.

---

## Shared Implementation Notes

### Testing strategy

- Unit tests for `.env` parser (various quoting, edge cases).
- Unit tests for access control rule matching (glob patterns).
- Integration test: `uzp import .env && uzp run -p test -- env` verifies round-trip.
- MCP tools: test via direct JSON-RPC stdin/stdout interaction.

### Version and release

- Each phase is a minor version bump (1.1.0, 1.2.0, 1.3.0).
- Each phase ships as independent PR with its own tests.
- CHANGELOG.md auto-updated via existing CI workflow.
