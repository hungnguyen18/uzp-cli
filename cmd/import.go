package cmd

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/hungnguyen18/uzp-cli/internal/dotenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	importProject     string
	importOverwrite   bool
	importInteractive bool
)

var importCmd = &cobra.Command{
	Use:   "import <file> --project <name>",
	Short: "Import secrets from a .env file into the vault",
	Long: `Import Secrets

Import key-value pairs from a .env file into a vault project.

EXAMPLES:
  uzp import .env --project myapp
  uzp import .env.example --project myapp --interactive
  cat secrets.env | uzp import - --project backend

OPTIONS:
  --project, -p     Target project name (required)
  --overwrite       Overwrite existing keys (default: skip)
  --interactive     Prompt for each value instead of reading from file`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		if importProject == "" {
			return fmt.Errorf("missing project name\n\nUsage: uzp import <file> --project <name>")
		}

		if importInteractive && filePath == "-" {
			return fmt.Errorf("--interactive cannot be used with stdin (-)")
		}

		// Read file content
		var content []byte
		var err error
		if filePath == "-" {
			content, err = io.ReadAll(os.Stdin)
		} else {
			content, err = os.ReadFile(filePath)
		}
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Parse .env content
		entries, warnings := dotenv.Parse(string(content))

		// Print warnings to stderr
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
		}

		if len(entries) == 0 {
			fmt.Fprintln(os.Stderr, "No entries found in file.")
			return nil
		}

		// Unlock vault
		if err := ensureVaultUnlocked(); err != nil {
			return err
		}

		imported := 0
		skipped := 0

		// Deduplicate: last occurrence wins (matches .env behavior)
		seen := make(map[string]int)
		for i, e := range entries {
			seen[e.Key] = i
		}

		for i, entry := range entries {
			// Skip if not the last occurrence of this key
			if seen[entry.Key] != i {
				continue
			}

			value := entry.Value

			// Interactive mode: prompt for value
			if importInteractive {
				if value != "" {
					fmt.Fprintf(os.Stderr, "%s (default: %s): ", entry.Key, value)
				} else {
					fmt.Fprintf(os.Stderr, "%s: ", entry.Key)
				}
				input, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Fprintln(os.Stderr)
				if err != nil {
					return fmt.Errorf("failed to read input for %s: %w", entry.Key, err)
				}
				if len(input) > 0 {
					value = string(input)
				}
				// Clear input from memory
				for j := range input {
					input[j] = 0
				}
			}

			// Check if key exists
			if !importOverwrite {
				if _, err := vault.Get(importProject, entry.Key); err == nil {
					skipped++
					continue
				}
			}

			if err := vault.Add(importProject, entry.Key, value); err != nil {
				return fmt.Errorf("failed to add %s: %w", entry.Key, err)
			}
			imported++
		}

		fmt.Fprintf(os.Stderr, "Imported %d secrets into project '%s'", imported, importProject)
		if skipped > 0 {
			fmt.Fprintf(os.Stderr, " (%d skipped, use --overwrite to replace)", skipped)
		}
		fmt.Fprintln(os.Stderr)

		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&importProject, "project", "p", "", "Target project name (required)")
	importCmd.Flags().BoolVar(&importOverwrite, "overwrite", false, "Overwrite existing keys")
	importCmd.Flags().BoolVarP(&importInteractive, "interactive", "i", false, "Prompt for each value")
}
