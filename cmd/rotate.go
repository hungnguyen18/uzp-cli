package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hungnguyen18/uzp-cli/internal/rotation"
	"github.com/spf13/cobra"
)

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Manage secret rotation policies and status",
	Long: `Rotation Management

Track when secrets were last updated and set rotation policies
to ensure credentials are refreshed on schedule.

SUBCOMMANDS:
  list    List all secrets with last-updated timestamps
  check   Check which secrets need rotation
  set     Set a rotation policy for a secret
  unset   Remove a rotation policy

EXAMPLES:
  uzp rotate list
  uzp rotate check
  uzp rotate set myapp/api_key 90d
  uzp rotate unset myapp/api_key`,
}

var rotateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets with rotation status",
	Long: `List Rotation Status

Display all secrets with their last-updated timestamps and rotation policies.

EXAMPLES:
  uzp rotate list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureVaultUnlocked(); err != nil {
			return err
		}

		projects, err := vault.List()
		if err != nil {
			return err
		}

		if len(projects) == 0 {
			fmt.Println("No secrets found.")
			return nil
		}

		store, err := rotation.NewStore()
		if err != nil {
			return fmt.Errorf("failed to initialize rotation store: %w", err)
		}
		if err := store.Load(); err != nil {
			return fmt.Errorf("failed to load rotation data: %w", err)
		}

		// Collect all secret paths sorted
		listPath := secretPathList(projects)

		fmt.Println("Secret Rotation Status")
		fmt.Println("======================")
		fmt.Println()

		for i := 0; i < len(listPath); i++ {
			secretPath := listPath[i]

			// Build last-updated info
			updatedStr := "last updated: unknown"
			if ts, ok := store.GetLastUpdated(secretPath); ok {
				days := rotation.DaysAgo(ts)
				updatedStr = fmt.Sprintf("last updated: %s (%d days ago)", ts.Format("2006-01-02"), days)
			}

			// Build policy info
			policyStr := "policy: none"
			if p := store.GetPolicy(secretPath); p != nil {
				policyStr = fmt.Sprintf("policy: %dd", p.PeriodDays)
			}

			fmt.Printf("  %-24s %s  %s\n", secretPath, updatedStr, policyStr)
		}

		return nil
	},
}

var rotateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check which secrets need rotation",
	Long: `Rotation Check

Check all secrets with rotation policies and report which are overdue,
due soon, or have no policy set.

EXAMPLES:
  uzp rotate check`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureVaultUnlocked(); err != nil {
			return err
		}

		projects, err := vault.List()
		if err != nil {
			return err
		}

		if len(projects) == 0 {
			fmt.Println("No secrets found.")
			return nil
		}

		store, err := rotation.NewStore()
		if err != nil {
			return fmt.Errorf("failed to initialize rotation store: %w", err)
		}
		if err := store.Load(); err != nil {
			return fmt.Errorf("failed to load rotation data: %w", err)
		}

		listPath := secretPathList(projects)

		fmt.Println("Rotation Check")
		fmt.Println("==============")
		fmt.Println()

		countOverdue := 0
		countDueSoon := 0
		countNoPolicy := 0

		for i := 0; i < len(listPath); i++ {
			secretPath := listPath[i]

			p := store.GetPolicy(secretPath)
			if p == nil {
				countNoPolicy++
				continue
			}

			ts, hasTimestamp := store.GetLastUpdated(secretPath)
			if !hasTimestamp {
				// Has policy but no timestamp — treat as overdue
				fmt.Printf("  [OVERDUE] %s - last updated: unknown (policy: %dd)\n", secretPath, p.PeriodDays)
				countOverdue++
				continue
			}

			daysAge := rotation.DaysAgo(ts)
			daysRemaining := p.PeriodDays - daysAge

			if daysRemaining <= 0 {
				fmt.Printf("  [OVERDUE] %s - last updated %d days ago (policy: %dd) - %d days overdue!\n",
					secretPath, daysAge, p.PeriodDays, -daysRemaining)
				countOverdue++
			} else {
				fmt.Printf("  [DUE]     %s - last updated %d days ago (policy: %dd) - due in %d days\n",
					secretPath, daysAge, p.PeriodDays, daysRemaining)
				countDueSoon++
			}
		}

		if countNoPolicy > 0 {
			fmt.Printf("  [OK]      %d secrets without rotation policy (use `uzp rotate set` to add)\n", countNoPolicy)
		}

		fmt.Printf("\nSummary: %d overdue, %d due soon, %d no policy\n", countOverdue, countDueSoon, countNoPolicy)

		return nil
	},
}

var rotateSetCmd = &cobra.Command{
	Use:   "set <project/key> <duration>",
	Short: "Set rotation policy for a secret",
	Long: `Set Rotation Policy

Set a rotation policy for a secret. Duration is specified in days (e.g. 90d).

FORMAT:
  uzp rotate set <project/key> <duration>

EXAMPLES:
  uzp rotate set myapp/api_key 90d
  uzp rotate set myapp/stripe_key 30d`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse project/key
		parts := strings.Split(args[0], "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid format. use: project/key")
		}
		project := parts[0]
		key := parts[1]
		secretPath := args[0]

		// Parse duration
		days, err := rotation.ParseDuration(args[1])
		if err != nil {
			return err
		}

		// Verify the secret exists in the vault
		if err := ensureVaultUnlocked(); err != nil {
			return err
		}

		if _, err := vault.Get(project, key); err != nil {
			return fmt.Errorf("secret not found: %s", secretPath)
		}

		// Load rotation store
		store, err := rotation.NewStore()
		if err != nil {
			return fmt.Errorf("failed to initialize rotation store: %w", err)
		}
		if err := store.Load(); err != nil {
			return fmt.Errorf("failed to load rotation data: %w", err)
		}

		store.SetPolicy(secretPath, days)

		if err := store.Save(); err != nil {
			return fmt.Errorf("failed to save rotation data: %w", err)
		}

		fmt.Printf("Set rotation policy: %s = %dd\n", secretPath, days)

		return nil
	},
}

var rotateUnsetCmd = &cobra.Command{
	Use:   "unset <project/key>",
	Short: "Remove rotation policy for a secret",
	Long: `Remove Rotation Policy

Remove the rotation policy for a secret. The last-updated timestamp
is preserved for historical reference.

FORMAT:
  uzp rotate unset <project/key>

EXAMPLES:
  uzp rotate unset myapp/api_key`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse project/key
		parts := strings.Split(args[0], "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid format. use: project/key")
		}
		secretPath := args[0]

		// Load rotation store
		store, err := rotation.NewStore()
		if err != nil {
			return fmt.Errorf("failed to initialize rotation store: %w", err)
		}
		if err := store.Load(); err != nil {
			return fmt.Errorf("failed to load rotation data: %w", err)
		}

		if store.GetPolicy(secretPath) == nil {
			fmt.Printf("No rotation policy found for %s\n", secretPath)
			return nil
		}

		store.UnsetPolicy(secretPath)

		if err := store.Save(); err != nil {
			return fmt.Errorf("failed to save rotation data: %w", err)
		}

		fmt.Printf("Removed rotation policy for %s\n", secretPath)

		return nil
	},
}

// secretPathList collects all project/key paths from the vault and returns them sorted.
func secretPathList(projects map[string][]string) []string {
	var listPath []string
	for project, keys := range projects {
		for i := 0; i < len(keys); i++ {
			listPath = append(listPath, project+"/"+keys[i])
		}
	}
	sort.Strings(listPath)
	return listPath
}

func init() {
	rotateCmd.AddCommand(rotateListCmd)
	rotateCmd.AddCommand(rotateCheckCmd)
	rotateCmd.AddCommand(rotateSetCmd)
	rotateCmd.AddCommand(rotateUnsetCmd)
}
