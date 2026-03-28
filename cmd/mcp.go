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
