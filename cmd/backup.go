package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var backupOutput string

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup the vault file",
	Long: `Backup Vault

Create a backup copy of the encrypted vault file.

EXAMPLES:
  uzp backup                           Backup to ~/.uzp/backups/
  uzp backup -o ~/safe/vault.bak       Backup to custom path

DEFAULT:
  Backups are stored in ~/.uzp/backups/ with timestamp filenames.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if vault exists
		if !vault.Exists() {
			return fmt.Errorf("vault does not exist, run 'uzp init' first")
		}

		// Determine backup path
		backupPath := backupOutput
		if backupPath == "" {
			backupDir := filepath.Join(filepath.Dir(vault.Path()), "backups")
			if err := os.MkdirAll(backupDir, 0700); err != nil {
				return fmt.Errorf("failed to create backup directory: %w", err)
			}
			timestamp := time.Now().Format("20060102_150405")
			backupPath = filepath.Join(backupDir, fmt.Sprintf("vault_%s.bak", timestamp))
		} else {
			// Create parent directory if needed for custom path
			dir := filepath.Dir(backupPath)
			if err := os.MkdirAll(dir, 0700); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		// Copy vault file to backup location
		if err := fileCopy(vault.Path(), backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		// Verify backup was written correctly by comparing file sizes
		srcInfo, err := os.Stat(vault.Path())
		if err != nil {
			return fmt.Errorf("failed to stat vault file: %w", err)
		}
		dstInfo, err := os.Stat(backupPath)
		if err != nil {
			return fmt.Errorf("failed to stat backup file: %w", err)
		}
		if srcInfo.Size() != dstInfo.Size() {
			// Clean up incomplete backup
			os.Remove(backupPath)
			return fmt.Errorf("backup verification failed: file sizes do not match")
		}

		fmt.Fprintf(os.Stderr, "Vault backed up to: %s\n", backupPath)
		return nil
	},
}

// fileCopy copies a file from src to dst with 0600 permissions using atomic write
func fileCopy(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Write to temp file first, then rename for atomicity
	dir := filepath.Dir(dst)
	tmpFile, err := os.CreateTemp(dir, "uzp.backup.*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to copy data: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename backup file: %w", err)
	}

	return nil
}

func init() {
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Custom output path for backup file")
}
