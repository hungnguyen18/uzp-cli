package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Finding severity levels
const (
	severityWarn = "WARN"
	severityInfo = "INFO"
)

// Finding represents a single audit finding
type Finding struct {
	Severity string
	Path     string
	Message  string
}

var auditProject string

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit vault secrets for potential issues",
	Long: `Audit Secrets

Perform a health check on secrets stored in the vault.

CHECKS:
  - Empty values (WARN)
  - Weak values: length < 16 chars (WARN)
  - Duplicate values across project/key pairs (INFO)
  - Short key names: < 3 chars (INFO)

USAGE:
  uzp audit                Audit entire vault
  uzp audit -p myapp       Audit single project

EXIT CODES:
  0  No warnings found
  1  One or more warnings found`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if vault is unlocked, prompt for password if needed
		if err := ensureVaultUnlocked(); err != nil {
			return err
		}

		// Collect all secrets to audit
		secretMap, err := secretCollect(auditProject)
		if err != nil {
			return err
		}

		if len(secretMap) == 0 {
			fmt.Println("No secrets found.")
			return nil
		}

		// Run all checks
		findings := secretAudit(secretMap)

		// Print report
		secretAuditPrint(findings, len(secretMap))

		// Exit code 1 if any warnings found
		for _, f := range findings {
			if f.Severity == severityWarn {
				os.Exit(1)
			}
		}

		return nil
	},
}

func init() {
	auditCmd.Flags().StringVarP(&auditProject, "project", "p", "", "Audit only this project")
}

// secretCollect gathers secrets to audit, optionally filtering by project
func secretCollect(project string) (map[string]string, error) {
	result := make(map[string]string)

	if project != "" {
		secrets, err := vault.GetProjectSecrets(project)
		if err != nil {
			return nil, fmt.Errorf("project not found: %s", project)
		}
		for key, value := range secrets {
			result[project+"/"+key] = value
		}
		return result, nil
	}

	// All projects
	projects, err := vault.List()
	if err != nil {
		return nil, err
	}

	for proj, keys := range projects {
		for _, key := range keys {
			value, err := vault.Get(proj, key)
			if err != nil {
				continue
			}
			result[proj+"/"+key] = value
		}
	}

	return result, nil
}

// secretAudit runs all checks and returns sorted findings (WARN before INFO)
func secretAudit(secrets map[string]string) []Finding {
	var findings []Finding

	// Build reverse map for duplicate detection: value -> list of paths
	valueMap := make(map[string][]string)
	for path, value := range secrets {
		if value != "" {
			valueMap[value] = append(valueMap[value], path)
		}
	}

	// Sort paths for deterministic output
	listPath := make([]string, 0, len(secrets))
	for path := range secrets {
		listPath = append(listPath, path)
	}
	sort.Strings(listPath)

	for _, path := range listPath {
		value := secrets[path]

		// Extract key name (after the last /)
		parts := strings.SplitN(path, "/", 2)
		keyName := ""
		if len(parts) == 2 {
			keyName = parts[1]
		}

		// Check: empty value
		if value == "" {
			findings = append(findings, Finding{
				Severity: severityWarn,
				Path:     path,
				Message:  "empty value",
			})
			continue // skip weak check for empty values
		}

		// Check: weak value (< 16 chars)
		if len(value) < 16 {
			findings = append(findings, Finding{
				Severity: severityWarn,
				Path:     path,
				Message:  "value length < 16 chars (weak secret)",
			})
		}

		// Check: duplicate values
		if duplicates, ok := valueMap[value]; ok && len(duplicates) > 1 {
			for _, dup := range duplicates {
				if dup != path && dup > path {
					findings = append(findings, Finding{
						Severity: severityInfo,
						Path:     path,
						Message:  fmt.Sprintf("duplicate value found in %s", dup),
					})
				}
			}
		}

		// Check: short key name (< 3 chars)
		if len(keyName) < 3 {
			findings = append(findings, Finding{
				Severity: severityInfo,
				Path:     path,
				Message:  "key name very short (< 3 chars)",
			})
		}
	}

	// Sort findings: WARN before INFO, then by path
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity == severityWarn
		}
		return findings[i].Path < findings[j].Path
	})

	return findings
}

// secretAuditPrint prints the audit report to stdout
func secretAuditPrint(findings []Finding, totalSecret int) {
	fmt.Println("Audit Report")
	fmt.Println("============")
	fmt.Println()

	if len(findings) == 0 {
		fmt.Printf("All %d secrets passed health checks!\n", totalSecret)
		return
	}

	countWarn := 0
	countInfo := 0

	for _, f := range findings {
		fmt.Printf("  [%s] %s - %s\n", f.Severity, f.Path, f.Message)
		switch f.Severity {
		case severityWarn:
			countWarn++
		case severityInfo:
			countInfo++
		}
	}

	countPass := totalSecret - countWarn - countInfo
	if countPass < 0 {
		countPass = 0
	}

	fmt.Println()
	fmt.Printf("Summary: %d secrets checked, %d warnings, %d info, %d passed\n",
		totalSecret, countWarn, countInfo, countPass)
}
