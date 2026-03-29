package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore the vault from a backup",
	Long: `Restore Vault

Restore the vault from a backup file.

EXAMPLES:
  uzp restore vault.bak
  uzp restore ~/safe/vault_20260329_153045.bak

CONFIRMATION:
  You will be prompted to type 'RESTORE' to confirm.

SAFETY:
  A backup of the current vault is created before restoring.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupFile := args[0]

		// Verify backup file exists and is readable
		info, err := os.Stat(backupFile)
		if err != nil {
			return fmt.Errorf("backup file not found: %w", err)
		}
		if info.IsDir() {
			return fmt.Errorf("backup path is a directory, not a file")
		}

		// Verify backup file is a valid vault format (can be parsed as JSON with expected fields)
		if err := fileValidateVault(backupFile); err != nil {
			return fmt.Errorf("backup file is not a valid vault: %w", err)
		}

		// Ask for confirmation
		fmt.Fprintln(os.Stderr, "WARNING: This will replace the current vault with the backup!")
		fmt.Fprintln(os.Stderr, "Any secrets added since the backup was created will be lost.")
		fmt.Fprint(os.Stderr, "\nType 'RESTORE' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "RESTORE" {
			fmt.Fprintln(os.Stderr, "Restore cancelled.")
			return nil
		}

		// Create a safety backup of the current vault before restoring
		if vault.Exists() {
			safetyDir := filepath.Join(filepath.Dir(vault.Path()), "backups")
			if err := os.MkdirAll(safetyDir, 0700); err != nil {
				return fmt.Errorf("failed to create backup directory: %w", err)
			}
			timestamp := time.Now().Format("20060102_150405")
			safetyPath := filepath.Join(safetyDir, fmt.Sprintf("vault_pre_restore_%s.bak", timestamp))

			if err := fileCopy(vault.Path(), safetyPath); err != nil {
				return fmt.Errorf("failed to create safety backup: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Current vault backed up to: %s\n", safetyPath)
		}

		// Restore: copy backup file to vault location using atomic write
		if err := fileCopy(backupFile, vault.Path()); err != nil {
			return fmt.Errorf("failed to restore vault: %w", err)
		}

		// Verify the restored vault can be loaded
		if err := fileValidateVault(vault.Path()); err != nil {
			return fmt.Errorf("restored vault validation failed: %w", err)
		}

		fmt.Fprintln(os.Stderr, "Vault restored successfully.")
		return nil
	},
}

// fileValidateVault checks that the file is valid encrypted vault JSON
func fileValidateVault(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var vaultFile struct {
		Salt string `json:"salt"`
		Hash string `json:"hash"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(data, &vaultFile); err != nil {
		return fmt.Errorf("failed to parse vault JSON: %w", err)
	}

	if vaultFile.Salt == "" || vaultFile.Hash == "" || vaultFile.Data == "" {
		return fmt.Errorf("missing required vault fields (salt, hash, data)")
	}

	return nil
}
