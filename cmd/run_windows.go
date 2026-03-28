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
