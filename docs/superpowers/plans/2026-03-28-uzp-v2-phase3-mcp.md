# Phase 3: `uzp mcp` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `uzp mcp` that starts a read-only MCP server over stdio, allowing AI agents (Claude Code, Codex, OpenCode) to access vault secrets with scope-based access control.

**Architecture:** Minimal MCP JSON-RPC handler over stdio (no external SDK dependency). Three tools: `uzp_get`, `uzp_list`, `uzp_search`. Access control via `~/.uzp/access.json` with allow/prompt/deny rules per project (glob patterns). Prompt mode asks user on stderr.

**Tech Stack:** Go 1.24, cobra, JSON-RPC over stdio, `path.Match` for glob patterns

---

## File Structure

| Action | Path                          | Responsibility                                                          |
| ------ | ----------------------------- | ----------------------------------------------------------------------- |
| Create | `internal/mcp/access.go`      | Load and evaluate access control rules from `~/.uzp/access.json`        |
| Create | `internal/mcp/access_test.go` | Tests for rule matching, glob patterns, defaults                        |
| Create | `internal/mcp/server.go`      | MCP JSON-RPC protocol handler: read stdin, write stdout, route to tools |
| Create | `internal/mcp/tools.go`       | Tool implementations: get, list, search (wrap vault operations)         |
| Create | `internal/mcp/server_test.go` | Tests for JSON-RPC request/response handling                            |
| Create | `cmd/mcp.go`                  | Cobra command entry point, wire server to vault                         |
| Modify | `cmd/root.go`                 | Register `mcpCmd`                                                       |

---

### Task 1: Implement access control

**Files:**

- Create: `internal/mcp/access.go`
- Create: `internal/mcp/access_test.go`

- [ ] **Step 1: Write access control tests**

Create `internal/mcp/access_test.go`:

```go
package mcp

import "testing"

func TestAccessCheck(t *testing.T) {
	rules := &AccessConfig{
		Default: "deny",
		Rules: []AccessRule{
			{Project: "myapp", Access: "allow"},
			{Project: "production", Access: "prompt"},
			{Project: "infra/*", Access: "deny"},
		},
	}

	tests := []struct {
		project  string
		expected string
	}{
		{"myapp", "allow"},
		{"production", "prompt"},
		{"infra/secrets", "deny"},
		{"infra/keys", "deny"},
		{"unknown", "deny"},           // falls to default
		{"myapp-staging", "deny"},     // exact match only for non-glob
	}

	for _, tt := range tests {
		t.Run(tt.project, func(t *testing.T) {
			result := rules.Check(tt.project)
			if result != tt.expected {
				t.Errorf("Check(%q) = %q, want %q", tt.project, result, tt.expected)
			}
		})
	}
}

func TestAccessCheckDefaultPrompt(t *testing.T) {
	rules := &AccessConfig{
		Default: "",
		Rules:   nil,
	}
	result := rules.Check("anything")
	if result != "prompt" {
		t.Errorf("expected default 'prompt', got %q", result)
	}
}

func TestAccessConfigNoFile(t *testing.T) {
	config, err := LoadAccessConfig("/nonexistent/path/access.json")
	if err != nil {
		t.Fatal(err)
	}
	if config.Default != "prompt" {
		t.Errorf("expected default 'prompt' when no file, got %q", config.Default)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcp/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement access control**

Create `internal/mcp/access.go`:

```go
package mcp

import (
	"encoding/json"
	"os"
	"path"
)

// AccessRule defines access level for a project pattern.
type AccessRule struct {
	Project string `json:"project"`
	Access  string `json:"access"` // "allow", "prompt", or "deny"
}

// AccessConfig holds the full access control configuration.
type AccessConfig struct {
	Default string       `json:"default"`
	Rules   []AccessRule `json:"rules"`
}

// LoadAccessConfig loads access rules from the given path.
// Returns a safe default (prompt for all) if the file does not exist.
func LoadAccessConfig(filePath string) (*AccessConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &AccessConfig{Default: "prompt"}, nil
		}
		return nil, err
	}

	var config AccessConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Safe default if not specified
	if config.Default == "" {
		config.Default = "prompt"
	}

	return &config, nil
}

// Check returns the access level for a given project name.
// Evaluates rules in order; first match wins. Falls back to Default.
func (c *AccessConfig) Check(project string) string {
	for _, rule := range c.Rules {
		matched, err := path.Match(rule.Project, project)
		if err != nil {
			continue
		}
		if matched {
			return rule.Access
		}
	}

	if c.Default == "" {
		return "prompt"
	}
	return c.Default
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcp/ -v`
Expected: PASS — all access control tests.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/
git commit -S -m "feat: add MCP access control with glob pattern matching"
```

---

### Task 2: Implement MCP JSON-RPC server

**Files:**

- Create: `internal/mcp/server.go`
- Create: `internal/mcp/server_test.go`

- [ ] **Step 1: Write server protocol tests**

Create `internal/mcp/server_test.go`:

```go
package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestServerInitialize(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"

	stdin := strings.NewReader(input)
	var stdout bytes.Buffer

	server := NewServer(nil, nil, &AccessConfig{Default: "allow"})
	server.HandleSingle(stdin, &stdout)

	var resp JSONRPCResponse
	if err := json.NewDecoder(&stdout).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != 1 {
		t.Errorf("expected id=1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}
}

func TestServerToolsList(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n"

	stdin := strings.NewReader(input)
	var stdout bytes.Buffer

	server := NewServer(nil, nil, &AccessConfig{Default: "allow"})
	server.HandleSingle(stdin, &stdout)

	var resp JSONRPCResponse
	if err := json.NewDecoder(&stdout).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("expected tools array, got %T", result["tools"])
	}
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcp/ -run TestServer -v`
Expected: FAIL — `NewServer`, `JSONRPCResponse` not defined.

- [ ] **Step 3: Implement MCP server**

Create `internal/mcp/server.go`:

```go
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hungnguyen18/uzp-cli/internal/storage"
)

// JSONRPCRequest is a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError is a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// VaultUnlocker is called to unlock the vault before first tool use.
type VaultUnlocker func() error

// Server handles MCP protocol over stdio.
type Server struct {
	vault    *storage.Vault
	unlocker VaultUnlocker
	access   *AccessConfig
}

// NewServer creates a new MCP server.
func NewServer(vault *storage.Vault, unlocker VaultUnlocker, access *AccessConfig) *Server {
	return &Server{
		vault:    vault,
		unlocker: unlocker,
		access:   access,
	}
}

// Run starts the MCP server, reading from stdin and writing to stdout.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer for large messages
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp := s.handleRequest(line)
		if resp == nil {
			continue // notification, no response needed
		}

		data, err := json.Marshal(resp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
			continue
		}
		fmt.Fprintf(os.Stdout, "%s\n", data)
	}

	return scanner.Err()
}

// HandleSingle processes a single request from reader and writes response to writer.
// Used for testing.
func (s *Server) HandleSingle(reader io.Reader, writer io.Writer) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	if scanner.Scan() {
		resp := s.handleRequest(scanner.Bytes())
		if resp != nil {
			data, _ := json.Marshal(resp)
			fmt.Fprintf(writer, "%s\n", data)
		}
	}
}

func (s *Server) handleRequest(data []byte) *JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &JSONRPCError{Code: -32700, Message: "Parse error"},
		}
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.ID)
	case "notifications/initialized":
		return nil // notification, no response
	case "tools/list":
		return s.handleToolsList(req.ID)
	case "tools/call":
		return s.handleToolsCall(req.ID, req.Params)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONRPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}
	}
}

func (s *Server) handleInitialize(id interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "uzp",
				"version": "1.0.0",
			},
		},
	}
}

func (s *Server) handleToolsList(id interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"tools": toolDefinitions(),
		},
	}
}

func (s *Server) handleToolsCall(id interface{}, params json.RawMessage) *JSONRPCResponse {
	// Ensure vault is unlocked on first tool call
	if s.unlocker != nil && s.vault != nil && !s.vault.IsUnlocked() {
		if err := s.unlocker(); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONRPCError{Code: -32000, Message: "Vault unlock failed: " + err.Error()},
			}
		}
	}

	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &JSONRPCError{Code: -32602, Message: "Invalid params"},
		}
	}

	result, err := s.executeTool(call.Name, call.Arguments)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "Error: " + err.Error()},
				},
				"isError": true,
			},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": result},
			},
		},
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcp/ -run TestServer -v`
Expected: PASS — initialize and tools/list.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/server.go internal/mcp/server_test.go
git commit -S -m "feat: add MCP JSON-RPC server with stdio transport"
```

---

### Task 3: Implement MCP tool handlers

**Files:**

- Create: `internal/mcp/tools.go`

- [ ] **Step 1: Create tool definitions and handlers**

Create `internal/mcp/tools.go`:

```go
package mcp

import (
	"encoding/json"
	"fmt"
	"os"
)

func toolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "uzp_get",
			"description": "Get a single secret value from the vault",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project": map[string]interface{}{"type": "string", "description": "Project name"},
					"key":     map[string]interface{}{"type": "string", "description": "Secret key name"},
				},
				"required": []string{"project", "key"},
			},
		},
		{
			"name":        "uzp_list",
			"description": "List projects and their keys (values are not returned)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project": map[string]interface{}{"type": "string", "description": "Optional: filter by project name"},
				},
			},
		},
		{
			"name":        "uzp_search",
			"description": "Search for projects and keys matching a keyword",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keyword": map[string]interface{}{"type": "string", "description": "Search keyword"},
				},
				"required": []string{"keyword"},
			},
		},
	}
}

func (s *Server) executeTool(name string, argsRaw json.RawMessage) (string, error) {
	switch name {
	case "uzp_get":
		return s.toolGet(argsRaw)
	case "uzp_list":
		return s.toolList(argsRaw)
	case "uzp_search":
		return s.toolSearch(argsRaw)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *Server) toolGet(argsRaw json.RawMessage) (string, error) {
	var args struct {
		Project string `json:"project"`
		Key     string `json:"key"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Access control check
	access := s.access.Check(args.Project)
	switch access {
	case "deny":
		return "", fmt.Errorf("access denied for project '%s'", args.Project)
	case "prompt":
		if !promptUserApproval(args.Project, args.Key) {
			return "", fmt.Errorf("access denied by user for project '%s'", args.Project)
		}
	}

	value, err := s.vault.Get(args.Project, args.Key)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *Server) toolList(argsRaw json.RawMessage) (string, error) {
	var args struct {
		Project string `json:"project"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Project != "" {
		// Access control for specific project
		access := s.access.Check(args.Project)
		if access == "deny" {
			return "", fmt.Errorf("access denied for project '%s'", args.Project)
		}

		secrets, err := s.vault.GetProjectSecrets(args.Project)
		if err != nil {
			return "", err
		}
		keys := make([]string, 0, len(secrets))
		for k := range secrets {
			keys = append(keys, k)
		}
		data, _ := json.MarshalIndent(map[string]interface{}{args.Project: keys}, "", "  ")
		return string(data), nil
	}

	// List all projects (no access control on listing project names)
	projects, err := s.vault.List()
	if err != nil {
		return "", err
	}
	data, _ := json.MarshalIndent(projects, "", "  ")
	return string(data), nil
}

func (s *Server) toolSearch(argsRaw json.RawMessage) (string, error) {
	var args struct {
		Keyword string `json:"keyword"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	results, err := s.vault.Search(args.Keyword)
	if err != nil {
		return "", err
	}
	data, _ := json.MarshalIndent(results, "", "  ")
	return string(data), nil
}

// promptUserApproval asks the user on stderr whether to allow access.
func promptUserApproval(project, key string) bool {
	if key != "" {
		fmt.Fprintf(os.Stderr, "MCP agent requests access to '%s/%s'. Allow? [y/N]: ", project, key)
	} else {
		fmt.Fprintf(os.Stderr, "MCP agent requests access to project '%s'. Allow? [y/N]: ", project)
	}

	var response string
	fmt.Fscanln(os.Stderr, &response)

	return response == "y" || response == "Y" || response == "yes"
}
```

- [ ] **Step 2: Verify build**

Run: `go vet ./... && go build -o /dev/null .`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add internal/mcp/tools.go
git commit -S -m "feat: add MCP tool handlers (get, list, search) with access control"
```

---

### Task 4: Implement `uzp mcp` command

**Files:**

- Create: `cmd/mcp.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create cmd/mcp.go**

Create `cmd/mcp.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	mcpserver "github.com/hungnguyen18/uzp-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for AI agent integration",
	Long: `MCP Server

Start a Model Context Protocol server over stdio for AI agent integration.

AI agents (Claude Code, Codex, OpenCode) can connect to read vault secrets
with scope-based access control.

CONFIGURATION:
  Add to your AI agent config:
  {
    "mcpServers": {
      "uzp": { "command": "uzp", "args": ["mcp"] }
    }
  }

ACCESS CONTROL:
  Configure ~/.uzp/access.json to control which projects agents can access.
  Default: prompt for all projects (user confirms each request).

TOOLS PROVIDED:
  uzp_get      Get a single secret value
  uzp_list     List projects and keys (no values)
  uzp_search   Search by keyword`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load access config
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to determine home directory: %w", err)
		}
		accessPath := filepath.Join(homeDir, ".uzp", "access.json")
		accessConfig, err := mcpserver.LoadAccessConfig(accessPath)
		if err != nil {
			return fmt.Errorf("failed to load access config: %w", err)
		}

		// Create MCP server with vault and unlocker
		server := mcpserver.NewServer(vault, ensureVaultUnlocked, accessConfig)

		// Run MCP server (blocks until stdin closes)
		return server.Run()
	},
}
```

- [ ] **Step 2: Register mcpCmd in root.go**

In `cmd/root.go`, add inside `init()` after the `importCmd` line:

```go
	rootCmd.AddCommand(mcpCmd)
```

- [ ] **Step 3: Verify build**

Run: `go vet ./... && go build -o /dev/null .`
Expected: Build succeeds.

- [ ] **Step 4: Manual smoke test**

```bash
go build -o uzp .
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./uzp mcp
```

Expected: JSON response with serverInfo name="uzp".

- [ ] **Step 5: Commit**

```bash
git add cmd/mcp.go cmd/root.go
git commit -S -m "feat: add uzp mcp command for AI agent integration"
```

---

### Task 5: Update help text and finalize

**Files:**

- Modify: `cmd/root.go`

- [ ] **Step 1: Add mcp to root help**

In `cmd/root.go`, add to BASIC USAGE:

```
  uzp mcp                       Start MCP server for AI agents
```

- [ ] **Step 2: Verify full build and all tests**

Run: `go vet ./... && go test ./... -v && go build -o /dev/null .`
Expected: All tests pass, build succeeds.

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go
git commit -S -m "docs: add uzp mcp to root help text"
```
