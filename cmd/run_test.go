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
