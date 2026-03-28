package cmd

import (
	"fmt"
	"os"

	"github.com/hungnguyen18/uzp-cli/internal/storage"
	"github.com/spf13/cobra"
)

// Version information - injected at build time
var Version = "dev" // Default value, will be overridden by ldflags during build

var (
	vault       *storage.Vault
	showVersion bool
	rootCmd     = &cobra.Command{
		Use:   "uzp",
		Short: "Secure secrets manager",
		Long: `UZP - Secure Secrets Manager

A command-line tool for securely storing and managing sensitive information
such as API keys, access tokens, and service credentials.

SECURITY:
  AES-256-GCM encryption, master password protection, local storage

BASIC USAGE:
  uzp init                    Initialize new vault
  uzp add                     Add secret
  uzp list                    List all secrets
  uzp get project/key         Get secret value
  uzp update project/key      Update secret
  uzp inject -p project       Export as environment variables
  uzp run -p project -- cmd    Run command with secrets injected
  uzp import .env -p project   Import from .env file
  uzp mcp                       Start MCP server for AI agents

EXAMPLES:
  uzp inject -p myapp > .env  Export secrets to .env file
  uzp copy myapp/api_key      Copy secret to clipboard
  uzp search database         Search for secrets
  uzp run -p myapp -- npm start Run with injected secrets

STORAGE: ~/.uzp/uzp.vault (encrypted)`,
		Run: func(cmd *cobra.Command, args []string) {
			if showVersion {
				fmt.Printf("uzp version %s\n", Version)
				return
			}
			// If no subcommand is provided, show help by default
			_ = cmd.Help() // Explicitly ignore error as it's unlikely to fail for help display
		},
	}
)

func init() {
	// Add version flags
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version information")

	// Initialize vault instance
	var err error
	vault, err = storage.NewVault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Add all subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(copyCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(injectCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(importCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
