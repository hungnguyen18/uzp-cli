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
