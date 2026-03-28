package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/hungnguyen18/uzp-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	ttl int
)

var copyCmd = &cobra.Command{
	Use:   "copy <project/key>",
	Short: "Copy a secret value to clipboard",
	Long: `Copy Secret

Copy a secret value to clipboard with automatic clearing.

FORMAT:
  project/key

EXAMPLES:
  uzp copy myapp/api_key
  uzp copy backend/database_url

OPTIONS:
  --ttl  Seconds before clipboard is cleared (default: 15)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate TTL
		if ttl < 1 {
			return fmt.Errorf("TTL must be at least 1 second")
		}

		// Check if vault is unlocked, prompt for password if needed
		if err := ensureVaultUnlocked(); err != nil {
			return err
		}

		// Parse project/key
		parts := strings.Split(args[0], "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format. Use: project/key")
		}

		project := parts[0]
		key := parts[1]

		// Get value
		value, err := vault.Get(project, key)
		if err != nil {
			return err
		}

		// Print message before blocking — user sees this immediately
		fmt.Printf("Copied %s/%s to clipboard.\n", project, key)
		fmt.Printf("Clipboard will be cleared in %d seconds...\n", ttl)

		// Copy to clipboard and block until TTL expires and clipboard is cleared
		duration := time.Duration(ttl) * time.Second
		if err := utils.CopyToClipboard(value, duration); err != nil {
			return err
		}

		fmt.Println("Clipboard cleared.")

		return nil
	},
}

func init() {
	copyCmd.Flags().IntVarP(&ttl, "ttl", "t", 15, "Time to live in seconds before clipboard is cleared")
}
