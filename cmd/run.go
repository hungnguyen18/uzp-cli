package cmd

import (
	"fmt"
	"os"

	"github.com/hungnguyen18/uzp-cli/internal/envutil"
	"github.com/spf13/cobra"
)

var runProjects []string

var runCmd = &cobra.Command{
	Use:   "run -p <project> [--] <command> [args...]",
	Short: "Run a command with secrets injected as environment variables",
	Long: `Run Command With Secrets

Inject vault secrets into a child process environment without writing
plaintext files to disk. Secrets exist only in the process environment.

EXAMPLES:
  uzp run -p myapp -- npm start
  uzp run -p shared -p myapp -- docker compose up
  uzp run -p backend -- go run .

Multiple -p flags merge secrets left-to-right (last wins on collision).

The -- separator is required between uzp flags and the child command.`,
	DisableFlagParsing: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		// args contains everything after "--"
		if len(args) == 0 {
			return fmt.Errorf("no command specified\n\nUsage: uzp run -p <project> -- <command> [args...]\n\nExample: uzp run -p myapp -- npm start")
		}

		if len(runProjects) == 0 {
			return fmt.Errorf("no project specified\n\nUsage: uzp run -p <project> -- <command> [args...]")
		}

		// Unlock vault
		if err := ensureVaultUnlocked(); err != nil {
			return err
		}

		// Load secrets from each project, in order
		var projectSecrets []map[string]string
		for _, project := range runProjects {
			secrets, err := vault.GetProjectSecrets(project)
			if err != nil {
				// Skip empty/missing projects with warning
				fmt.Fprintf(os.Stderr, "Warning: project '%s' not found, skipping\n", project)
				continue
			}
			projectSecrets = append(projectSecrets, secrets)
		}

		// Merge secrets (last-wins)
		merged := envutil.MergeSecrets(projectSecrets)

		// Build environment: current env + secrets overlay
		env := os.Environ()
		for key, value := range merged {
			env = append(env, key+"="+value)
		}

		// Execute the child command (platform-specific)
		return execCommand(args, env)
	},
}

func init() {
	runCmd.Flags().StringArrayVarP(&runProjects, "project", "p", nil, "Project name(s) to inject secrets from (repeatable)")
}
