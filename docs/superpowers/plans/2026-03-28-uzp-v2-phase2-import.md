# Phase 2: `uzp import` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `uzp import <file> --project <name>` that imports secrets from `.env` files into the vault, reducing migration barrier to one command.

**Architecture:** New `.env` parser in `internal/dotenv/parser.go` handles three value formats (double-quoted, single-quoted, unquoted). New `cmd/import.go` reads the file, optionally prompts for values interactively, and calls `vault.Add()` for each entry.

**Tech Stack:** Go 1.24, cobra, golang.org/x/term (for interactive mode)

---

## File Structure

| Action | Path                             | Responsibility                                                 |
| ------ | -------------------------------- | -------------------------------------------------------------- |
| Create | `internal/dotenv/parser.go`      | Parse `.env` file content into key-value pairs                 |
| Create | `internal/dotenv/parser_test.go` | Tests for all value formats and edge cases                     |
| Create | `cmd/import.go`                  | Cobra command: file reading, interactive mode, vault insertion |
| Modify | `cmd/root.go`                    | Register `importCmd`                                           |

---

### Task 1: Implement `.env` parser

**Files:**

- Create: `internal/dotenv/parser.go`
- Create: `internal/dotenv/parser_test.go`

- [ ] **Step 1: Write parser tests**

Create `internal/dotenv/parser_test.go`:

```go
package dotenv

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Entry
	}{
		{
			name:  "simple unquoted",
			input: "KEY=value",
			expected: []Entry{{Key: "KEY", Value: "value"}},
		},
		{
			name:  "double quoted with spaces",
			input: `DB_URL="postgres://user:pass@host/db"`,
			expected: []Entry{{Key: "DB_URL", Value: "postgres://user:pass@host/db"}},
		},
		{
			name:  "single quoted literal",
			input: `SECRET='raw $value "here"'`,
			expected: []Entry{{Key: "SECRET", Value: `raw $value "here"`}},
		},
		{
			name:  "double quoted escape sequences",
			input: `MSG="line1\nline2\\end"`,
			expected: []Entry{{Key: "MSG", Value: "line1\nline2\\end"}},
		},
		{
			name:  "empty value allowed",
			input: "EMPTY=",
			expected: []Entry{{Key: "EMPTY", Value: ""}},
		},
		{
			name:     "comment line skipped",
			input:    "# this is a comment",
			expected: nil,
		},
		{
			name:     "blank line skipped",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "malformed line no equals",
			input:    "NOVALUE",
			expected: nil,
		},
		{
			name:  "inline comment after unquoted value",
			input: "KEY=value # comment",
			expected: []Entry{{Key: "KEY", Value: "value"}},
		},
		{
			name:  "trailing whitespace trimmed on unquoted",
			input: "KEY=value   ",
			expected: []Entry{{Key: "KEY", Value: "value"}},
		},
		{
			name:  "export prefix stripped",
			input: "export API_KEY=secret",
			expected: []Entry{{Key: "API_KEY", Value: "secret"}},
		},
		{
			name:  "multiline input",
			input: "A=1\n# comment\nB=2\n\nC=3",
			expected: []Entry{
				{Key: "A", Value: "1"},
				{Key: "B", Value: "2"},
				{Key: "C", Value: "3"},
			},
		},
		{
			name:  "duplicate keys last wins",
			input: "KEY=first\nKEY=second",
			expected: []Entry{
				{Key: "KEY", Value: "first"},
				{Key: "KEY", Value: "second"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, warnings := Parse(tt.input)
			if len(entries) != len(tt.expected) {
				t.Fatalf("expected %d entries, got %d (warnings: %v)", len(tt.expected), len(entries), warnings)
			}
			for i, e := range entries {
				if e.Key != tt.expected[i].Key || e.Value != tt.expected[i].Value {
					t.Errorf("entry %d: got {%q, %q}, want {%q, %q}", i, e.Key, e.Value, tt.expected[i].Key, tt.expected[i].Value)
				}
			}
		})
	}
}

func TestParseMalformedWarnings(t *testing.T) {
	input := "GOOD=value\nBADLINE\nALSO_GOOD=ok"
	entries, warnings := Parse(input)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/dotenv/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement parser**

Create `internal/dotenv/parser.go`:

```go
package dotenv

import (
	"fmt"
	"strings"
)

// Entry represents a single key-value pair from a .env file.
type Entry struct {
	Key   string
	Value string
}

// Parse parses .env file content into key-value entries.
// Returns parsed entries and any warnings for malformed lines.
// Handles double-quoted (with escape sequences), single-quoted (literal), and unquoted values.
func Parse(content string) ([]Entry, []string) {
	var entries []Entry
	var warnings []string

	lines := strings.Split(content, "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip optional "export " prefix
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
			line = strings.TrimSpace(line)
		}

		// Find the first = sign
		eqIdx := strings.Index(line, "=")
		if eqIdx < 0 {
			warnings = append(warnings, fmt.Sprintf("line %d: skipping malformed line (no '='): %s", lineNum+1, line))
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		rawValue := line[eqIdx+1:]

		value := parseValue(rawValue)
		entries = append(entries, Entry{Key: key, Value: value})
	}

	return entries, warnings
}

// parseValue handles the three .env value formats.
func parseValue(raw string) string {
	raw = strings.TrimSpace(raw)

	if len(raw) == 0 {
		return ""
	}

	// Double-quoted: unescape \n, \\, \"
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		inner := raw[1 : len(raw)-1]
		inner = strings.ReplaceAll(inner, `\n`, "\n")
		inner = strings.ReplaceAll(inner, `\\`, "\\")
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		return inner
	}

	// Single-quoted: literal, no escaping
	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		return raw[1 : len(raw)-1]
	}

	// Unquoted: strip inline comments and trailing whitespace
	if idx := strings.Index(raw, " #"); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.TrimRight(raw, " \t")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/dotenv/ -v`
Expected: PASS — all 14 test cases.

- [ ] **Step 5: Commit**

```bash
git add internal/dotenv/
git commit -S -m "feat: add .env file parser with quoted value support"
```

---

### Task 2: Implement `uzp import` command

**Files:**

- Create: `cmd/import.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create cmd/import.go**

Create `cmd/import.go`:

```go
package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
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
```

- [ ] **Step 2: Register importCmd in root.go**

In `cmd/root.go`, add inside `init()` after the `runCmd` line:

```go
	rootCmd.AddCommand(importCmd)
```

- [ ] **Step 3: Verify build**

Run: `go vet ./... && go build -o /dev/null .`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add cmd/import.go cmd/root.go
git commit -S -m "feat: add uzp import command for .env file migration"
```

---

### Task 3: Update help text

**Files:**

- Modify: `cmd/root.go`

- [ ] **Step 1: Add import to root help BASIC USAGE**

In `cmd/root.go`, add after the `uzp run` line in BASIC USAGE:

```
  uzp import .env -p project   Import from .env file
```

- [ ] **Step 2: Verify build and commit**

Run: `go vet ./... && go build -o /dev/null .`

```bash
git add cmd/root.go
git commit -S -m "docs: add uzp import to root help text"
```
