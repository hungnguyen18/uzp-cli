# Phase 1: `uzp run` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `uzp run -p <project> -- <command>` that injects vault secrets into a child process environment without writing plaintext files to disk.

**Architecture:** New `cmd/run.go` command that loads secrets from one or more projects, merges them into the current environment, then replaces the process via `syscall.Exec` (Unix) or `os/exec` (Windows). Reuses `convertToEnvKey()` from `inject.go` (moved to shared location).

**Tech Stack:** Go 1.24, cobra, syscall (Unix), os/exec (Windows)

---

## File Structure

| Action | Path                               | Responsibility                                                           |
| ------ | ---------------------------------- | ------------------------------------------------------------------------ |
| Create | `cmd/run.go`                       | Cobra command definition, flag parsing, environment build, exec          |
| Create | `cmd/run_unix.go`                  | Unix-specific `execCommand()` using `syscall.Exec`                       |
| Create | `cmd/run_windows.go`               | Windows-specific `execCommand()` using `os/exec.Command`                 |
| Create | `internal/envutil/envutil.go`      | `ConvertToEnvKey()` and `MergeSecrets()` — shared between inject and run |
| Modify | `cmd/inject.go`                    | Replace local `convertToEnvKey` with `envutil.ConvertToEnvKey`           |
| Modify | `cmd/root.go`                      | Register `runCmd`                                                        |
| Create | `internal/envutil/envutil_test.go` | Tests for key conversion and secret merging                              |
| Create | `cmd/run_test.go`                  | Tests for flag parsing, environment building                             |

---

### Task 1: Extract `convertToEnvKey` into shared `envutil` package

**Files:**

- Create: `internal/envutil/envutil.go`
- Create: `internal/envutil/envutil_test.go`
- Modify: `cmd/inject.go`

- [ ] **Step 1: Write tests for ConvertToEnvKey**

Create `internal/envutil/envutil_test.go`:

```go
package envutil

import "testing"

func TestConvertToEnvKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"api_key", "API_KEY"},
		{"database-url", "DATABASE_URL"},
		{"myApp.secret", "MYAPP_SECRET"},
		{"ALREADY_UPPER", "ALREADY_UPPER"},
		{"key with spaces", "KEY_WITH_SPACES"},
		{"123numeric", "123NUMERIC"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ConvertToEnvKey(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToEnvKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/envutil/ -v`
Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Create envutil package with ConvertToEnvKey**

Create `internal/envutil/envutil.go`:

```go
package envutil

// ConvertToEnvKey converts a key name to UPPER_SNAKE_CASE environment variable format.
// Non-alphanumeric characters are replaced with underscores.
func ConvertToEnvKey(key string) string {
	result := make([]byte, 0, len(key))

	for i := 0; i < len(key); i++ {
		c := key[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			if c >= 'a' && c <= 'z' {
				c = c - 32
			}
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}

	return string(result)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/envutil/ -v`
Expected: PASS — all 7 test cases.

- [ ] **Step 5: Update inject.go to use envutil.ConvertToEnvKey**

In `cmd/inject.go`, replace the local `convertToEnvKey` function and update imports:

Replace import block:

```go
import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hungnguyen18/uzp-cli/internal/envutil"
	"github.com/spf13/cobra"
)
```

Replace the line inside the `for` loop:

```go
envKey := envutil.ConvertToEnvKey(key)
```

Delete the entire `convertToEnvKey` function (lines 92-111 of `cmd/inject.go`).

- [ ] **Step 6: Verify build and existing behavior**

Run: `go vet ./... && go build -o /dev/null .`
Expected: Build succeeds with no errors.

- [ ] **Step 7: Commit**

```bash
git add internal/envutil/ cmd/inject.go
git commit -S -m "refactor: extract ConvertToEnvKey into shared envutil package"
```

---

### Task 2: Add `MergeSecrets` to envutil

**Files:**

- Modify: `internal/envutil/envutil.go`
- Modify: `internal/envutil/envutil_test.go`

- [ ] **Step 1: Write tests for MergeSecrets**

Append to `internal/envutil/envutil_test.go`:

```go
func TestMergeSecrets(t *testing.T) {
	t.Run("single project", func(t *testing.T) {
		projects := []map[string]string{
			{"api_key": "secret1", "db_url": "postgres://..."},
		}
		result := MergeSecrets(projects)
		if result["API_KEY"] != "secret1" {
			t.Errorf("expected API_KEY=secret1, got %q", result["API_KEY"])
		}
		if result["DB_URL"] != "postgres://..." {
			t.Errorf("expected DB_URL=postgres://..., got %q", result["DB_URL"])
		}
	})

	t.Run("multiple projects last wins", func(t *testing.T) {
		projects := []map[string]string{
			{"api_key": "from-shared", "log_level": "info"},
			{"api_key": "from-myapp", "db_url": "postgres://..."},
		}
		result := MergeSecrets(projects)
		if result["API_KEY"] != "from-myapp" {
			t.Errorf("expected last-wins API_KEY=from-myapp, got %q", result["API_KEY"])
		}
		if result["LOG_LEVEL"] != "info" {
			t.Errorf("expected LOG_LEVEL=info, got %q", result["LOG_LEVEL"])
		}
		if result["DB_URL"] != "postgres://..." {
			t.Errorf("expected DB_URL=postgres://..., got %q", result["DB_URL"])
		}
	})

	t.Run("empty project skipped", func(t *testing.T) {
		projects := []map[string]string{
			{},
			{"key": "value"},
		}
		result := MergeSecrets(projects)
		if len(result) != 1 || result["KEY"] != "value" {
			t.Errorf("unexpected result: %v", result)
		}
	})

	t.Run("no projects", func(t *testing.T) {
		result := MergeSecrets(nil)
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/envutil/ -run TestMergeSecrets -v`
Expected: FAIL — `MergeSecrets` not defined.

- [ ] **Step 3: Implement MergeSecrets**

Append to `internal/envutil/envutil.go`:

```go
// MergeSecrets merges secrets from multiple projects into a flat map of ENV_KEY=value.
// Projects are applied left-to-right: later projects override earlier ones on key collision.
// Key names are converted to UPPER_SNAKE_CASE via ConvertToEnvKey.
func MergeSecrets(projects []map[string]string) map[string]string {
	result := make(map[string]string)
	for _, secrets := range projects {
		for key, value := range secrets {
			envKey := ConvertToEnvKey(key)
			result[envKey] = value
		}
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/envutil/ -v`
Expected: PASS — all ConvertToEnvKey and MergeSecrets tests.

- [ ] **Step 5: Commit**

```bash
git add internal/envutil/
git commit -S -m "feat: add MergeSecrets to envutil for multi-project environment building"
```

---

### Task 3: Implement `uzp run` command (Unix)

**Files:**

- Create: `cmd/run.go`
- Create: `cmd/run_unix.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create cmd/run.go with command definition and environment building**

Create `cmd/run.go`:

```go
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
```

- [ ] **Step 2: Create cmd/run_unix.go for Unix exec**

Create `cmd/run_unix.go`:

```go
//go:build !windows

package cmd

import (
	"fmt"
	"os/exec"
	"syscall"
)

// execCommand replaces the current process with the given command.
// On Unix, this uses syscall.Exec so no parent process retains secrets in memory.
func execCommand(args []string, env []string) error {
	binary, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", args[0])
	}

	return syscall.Exec(binary, args, env)
}
```

- [ ] **Step 3: Create cmd/run_windows.go for Windows exec**

Create `cmd/run_windows.go`:

```go
//go:build windows

package cmd

import (
	"fmt"
	"os"
	osexec "os/exec"
)

// execCommand runs the given command with the provided environment on Windows.
// Windows does not support syscall.Exec, so we use os/exec and forward the exit code.
func execCommand(args []string, env []string) error {
	cmd := osexec.Command(args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*osexec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Register runCmd in root.go**

In `cmd/root.go`, add inside the `init()` function, after the `resetCmd` line:

```go
	rootCmd.AddCommand(runCmd)
```

- [ ] **Step 5: Verify build**

Run: `go vet ./... && go build -o /dev/null .`
Expected: Build succeeds.

- [ ] **Step 6: Manual smoke test**

```bash
go build -o uzp .
./uzp run --help
```

Expected: Shows help text with usage, examples, and -p flag.

- [ ] **Step 7: Commit**

```bash
git add cmd/run.go cmd/run_unix.go cmd/run_windows.go cmd/root.go
git commit -S -m "feat: add uzp run command for secret injection into process environment"
```

---

### Task 4: Integration test for `uzp run`

**Files:**

- Create: `cmd/run_test.go`

- [ ] **Step 1: Write test for environment building logic**

Create `cmd/run_test.go`:

```go
package cmd

import (
	"testing"
)

func TestRunProjectFlagValidation(t *testing.T) {
	t.Run("no project flag returns error", func(t *testing.T) {
		runProjects = nil
		err := runCmd.RunE(runCmd, []string{"echo", "hello"})
		if err == nil {
			t.Fatal("expected error for missing project flag")
		}
		expected := "no project specified"
		if err.Error()[:len(expected)] != expected {
			t.Errorf("expected error starting with %q, got %q", expected, err.Error())
		}
	})

	t.Run("no command returns error", func(t *testing.T) {
		runProjects = []string{"myapp"}
		err := runCmd.RunE(runCmd, nil)
		if err == nil {
			t.Fatal("expected error for missing command")
		}
		expected := "no command specified"
		if err.Error()[:len(expected)] != expected {
			t.Errorf("expected error starting with %q, got %q", expected, err.Error())
		}
	})
}
```

- [ ] **Step 2: Run test**

Run: `go test ./cmd/ -run TestRunProjectFlagValidation -v`
Expected: PASS — both validation cases.

- [ ] **Step 3: Commit**

```bash
git add cmd/run_test.go
git commit -S -m "test: add validation tests for uzp run command"
```

---

### Task 5: Update root help text and bump version

**Files:**

- Modify: `cmd/root.go`
- Modify: `package.json`

- [ ] **Step 1: Update root command help to include run**

In `cmd/root.go`, update the `Long` string in `rootCmd` to add the `run` command:

Add after the `uzp inject -p project` line in BASIC USAGE:

```
  uzp run -p project -- cmd    Run command with secrets injected
```

Add after the `uzp search database` line in EXAMPLES:

```
  uzp run -p myapp -- npm start Run with injected secrets
```

- [ ] **Step 2: Verify build**

Run: `go vet ./... && go build -o /dev/null .`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go
git commit -S -m "docs: add uzp run to root help text"
```
