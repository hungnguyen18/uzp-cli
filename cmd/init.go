package cmd

import (
	"crypto/subtle"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new vault",
	Long: `Initialize Vault

Create a new vault with a master password.

EXAMPLES:
  uzp init

WORKFLOW:
  1. Enter master password (minimum 8 characters)
  2. Confirm password
  3. Vault is created and ready to use

STORAGE:
  Creates ~/.uzp/uzp.vault (encrypted)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if vault already exists
		if vault.Exists() {
			return fmt.Errorf("vault already exists")
		}

		// Prompt for master password
		fmt.Print("Enter master password: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println() // New line after password

		// Confirm password
		fmt.Print("Confirm master password: ")
		confirmPassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			// Clear first password before returning
			for i := range password {
				password[i] = 0
			}
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println() // New line after password

		// Check if passwords match (constant-time comparison)
		if subtle.ConstantTimeCompare(password, confirmPassword) != 1 {
			// Clear passwords from memory before returning
			for i := range password {
				password[i] = 0
			}
			for i := range confirmPassword {
				confirmPassword[i] = 0
			}
			return fmt.Errorf("passwords do not match")
		}

		// Clear confirmation password immediately — no longer needed
		for i := range confirmPassword {
			confirmPassword[i] = 0
		}

		// Check password strength
		if len(password) < 8 {
			for i := range password {
				password[i] = 0
			}
			return fmt.Errorf("password must be at least 8 characters long")
		}

		// Initialize vault — pass []byte directly, no string conversion
		if err := vault.Initialize(password); err != nil {
			for i := range password {
				password[i] = 0
			}
			return fmt.Errorf("failed to initialize vault: %w", err)
		}

		fmt.Println("Vault initialized successfully!")
		fmt.Println("Vault is ready to use.")

		// Clear password from memory
		for i := range password {
			password[i] = 0
		}

		return nil
	},
}
