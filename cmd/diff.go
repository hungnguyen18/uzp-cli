package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/hungnguyen18/uzp-cli/internal/envutil"
	"github.com/spf13/cobra"
)

var diffKeysOnly bool

// DiffResult holds the result of comparing two projects' secrets
type DiffResult struct {
	LeftOnly  []string // Keys only in left project
	RightOnly []string // Keys only in right project
	Different []string // Keys in both with different values
	Same      []string // Keys in both with same values
}

// diffCompare compares two sets of secrets and returns a DiffResult.
// Keys are converted to env format for display via ConvertToEnvKey.
func diffCompare(leftSecrets, rightSecrets map[string]string, keysOnly bool) DiffResult {
	// Build env-key maps
	leftEnv := make(map[string]string, len(leftSecrets))
	for k, v := range leftSecrets {
		leftEnv[envutil.ConvertToEnvKey(k)] = v
	}
	rightEnv := make(map[string]string, len(rightSecrets))
	for k, v := range rightSecrets {
		rightEnv[envutil.ConvertToEnvKey(k)] = v
	}

	var result DiffResult

	// Collect all unique keys
	allKeys := make(map[string]struct{})
	for k := range leftEnv {
		allKeys[k] = struct{}{}
	}
	for k := range rightEnv {
		allKeys[k] = struct{}{}
	}

	for k := range allKeys {
		leftVal, inLeft := leftEnv[k]
		rightVal, inRight := rightEnv[k]

		switch {
		case inLeft && !inRight:
			result.LeftOnly = append(result.LeftOnly, k)
		case !inLeft && inRight:
			result.RightOnly = append(result.RightOnly, k)
		case keysOnly:
			// When keys-only mode, all shared keys go to Same (no value comparison)
			result.Same = append(result.Same, k)
		case leftVal != rightVal:
			result.Different = append(result.Different, k)
		default:
			result.Same = append(result.Same, k)
		}
	}

	// Sort all slices alphabetically
	sort.Strings(result.LeftOnly)
	sort.Strings(result.RightOnly)
	sort.Strings(result.Different)
	sort.Strings(result.Same)

	return result
}

// diffHasDifferences returns true if there are any differences between the two projects
func diffHasDifferences(result DiffResult, keysOnly bool) bool {
	if len(result.LeftOnly) > 0 || len(result.RightOnly) > 0 {
		return true
	}
	if !keysOnly && len(result.Different) > 0 {
		return true
	}
	return false
}

var diffCmd = &cobra.Command{
	Use:   "diff PROJECT_A PROJECT_B",
	Short: "Compare secrets between two projects",
	Long: `Compare Secrets

Compare secrets between two projects in the vault, showing which keys
exist in one or both projects, and whether shared keys have the same values.

USAGE:
  uzp diff PROJECT_A PROJECT_B [--keys]

EXAMPLES:
  uzp diff staging prod           Compare staging vs prod
  uzp diff myapp-dev myapp-prod   Compare dev vs prod environments
  uzp diff app1 app2 --keys       Compare key presence only (skip value diff)

OUTPUT SYMBOLS:
  + Key only in left project
  + Key only in right project
  ~ Key in both with different values
  = Key in both with same values

EXIT CODES:
  0  Projects are identical
  1  Differences found (useful for CI scripting)`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		leftProject := args[0]
		rightProject := args[1]

		// Check if vault is unlocked, prompt for password if needed
		if err := ensureVaultUnlocked(); err != nil {
			return err
		}

		// Get secrets for both projects
		leftSecrets, err := vault.GetProjectSecrets(leftProject)
		if err != nil {
			return fmt.Errorf("project not found: %s", leftProject)
		}

		rightSecrets, err := vault.GetProjectSecrets(rightProject)
		if err != nil {
			return fmt.Errorf("project not found: %s", rightProject)
		}

		// Compare
		result := diffCompare(leftSecrets, rightSecrets, diffKeysOnly)

		// Print header
		fmt.Fprintf(os.Stderr, "Comparing %s \u2194 %s\n\n", leftProject, rightProject)

		// Print sections
		if len(result.LeftOnly) > 0 {
			fmt.Printf("Keys only in %s:\n", leftProject)
			for _, k := range result.LeftOnly {
				fmt.Printf("  + %s\n", k)
			}
			fmt.Println()
		}

		if len(result.RightOnly) > 0 {
			fmt.Printf("Keys only in %s:\n", rightProject)
			for _, k := range result.RightOnly {
				fmt.Printf("  + %s\n", k)
			}
			fmt.Println()
		}

		if !diffKeysOnly {
			if len(result.Different) > 0 {
				fmt.Printf("Keys in both (different values):\n")
				for _, k := range result.Different {
					fmt.Printf("  ~ %s\n", k)
				}
				fmt.Println()
			}

			if len(result.Same) > 0 {
				fmt.Printf("Keys in both (same values):\n")
				for _, k := range result.Same {
					fmt.Printf("  = %s\n", k)
				}
				fmt.Println()
			}
		}

		// Print summary
		if diffKeysOnly {
			fmt.Printf("Summary: %d only-left, %d only-right, %d shared\n",
				len(result.LeftOnly), len(result.RightOnly), len(result.Same))
		} else {
			fmt.Printf("Summary: %d only-left, %d only-right, %d different, %d same\n",
				len(result.LeftOnly), len(result.RightOnly), len(result.Different), len(result.Same))
		}

		// Exit code 1 if differences found
		if diffHasDifferences(result, diffKeysOnly) {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	diffCmd.Flags().BoolVar(&diffKeysOnly, "keys", false, "Only compare key presence (skip value comparison)")
}
