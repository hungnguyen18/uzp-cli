package cmd

import (
	"testing"
)

func TestDiffCompare(t *testing.T) {
	t.Run("identical projects", func(t *testing.T) {
		left := map[string]string{"api_key": "abc", "db_url": "postgres://"}
		right := map[string]string{"api_key": "abc", "db_url": "postgres://"}

		result := diffCompare(left, right, false)

		if len(result.LeftOnly) != 0 {
			t.Errorf("expected 0 left-only, got %d", len(result.LeftOnly))
		}
		if len(result.RightOnly) != 0 {
			t.Errorf("expected 0 right-only, got %d", len(result.RightOnly))
		}
		if len(result.Different) != 0 {
			t.Errorf("expected 0 different, got %d", len(result.Different))
		}
		if len(result.Same) != 2 {
			t.Errorf("expected 2 same, got %d", len(result.Same))
		}
	})

	t.Run("completely different projects", func(t *testing.T) {
		left := map[string]string{"debug_mode": "true", "test_key": "val"}
		right := map[string]string{"sentry_dsn": "https://", "cdn_url": "https://cdn"}

		result := diffCompare(left, right, false)

		if len(result.LeftOnly) != 2 {
			t.Errorf("expected 2 left-only, got %d", len(result.LeftOnly))
		}
		if len(result.RightOnly) != 2 {
			t.Errorf("expected 2 right-only, got %d", len(result.RightOnly))
		}
		if len(result.Different) != 0 {
			t.Errorf("expected 0 different, got %d", len(result.Different))
		}
		if len(result.Same) != 0 {
			t.Errorf("expected 0 same, got %d", len(result.Same))
		}
	})

	t.Run("mixed differences", func(t *testing.T) {
		left := map[string]string{
			"api_key":    "staging-key",
			"db_url":     "postgres://staging",
			"redis_url":  "redis://shared",
			"debug_mode": "true",
		}
		right := map[string]string{
			"api_key":    "prod-key",
			"db_url":     "postgres://prod",
			"redis_url":  "redis://shared",
			"sentry_dsn": "https://sentry",
		}

		result := diffCompare(left, right, false)

		if len(result.LeftOnly) != 1 {
			t.Errorf("expected 1 left-only, got %d: %v", len(result.LeftOnly), result.LeftOnly)
		}
		if len(result.RightOnly) != 1 {
			t.Errorf("expected 1 right-only, got %d: %v", len(result.RightOnly), result.RightOnly)
		}
		if len(result.Different) != 2 {
			t.Errorf("expected 2 different, got %d: %v", len(result.Different), result.Different)
		}
		if len(result.Same) != 1 {
			t.Errorf("expected 1 same, got %d: %v", len(result.Same), result.Same)
		}

		// Verify left-only key
		if result.LeftOnly[0] != "DEBUG_MODE" {
			t.Errorf("expected left-only key DEBUG_MODE, got %s", result.LeftOnly[0])
		}
		// Verify right-only key
		if result.RightOnly[0] != "SENTRY_DSN" {
			t.Errorf("expected right-only key SENTRY_DSN, got %s", result.RightOnly[0])
		}
	})

	t.Run("keys only mode skips value comparison", func(t *testing.T) {
		left := map[string]string{"api_key": "staging-key", "db_url": "staging-db"}
		right := map[string]string{"api_key": "prod-key", "cdn_url": "https://cdn"}

		result := diffCompare(left, right, true)

		if len(result.LeftOnly) != 1 {
			t.Errorf("expected 1 left-only, got %d", len(result.LeftOnly))
		}
		if len(result.RightOnly) != 1 {
			t.Errorf("expected 1 right-only, got %d", len(result.RightOnly))
		}
		// In keys-only mode, all shared keys go to Same regardless of value
		if len(result.Different) != 0 {
			t.Errorf("expected 0 different in keys-only mode, got %d", len(result.Different))
		}
		if len(result.Same) != 1 {
			t.Errorf("expected 1 same in keys-only mode, got %d", len(result.Same))
		}
	})

	t.Run("empty projects", func(t *testing.T) {
		left := map[string]string{}
		right := map[string]string{}

		result := diffCompare(left, right, false)

		if len(result.LeftOnly) != 0 || len(result.RightOnly) != 0 || len(result.Different) != 0 || len(result.Same) != 0 {
			t.Errorf("expected all empty slices for empty projects")
		}
	})

	t.Run("one empty project", func(t *testing.T) {
		left := map[string]string{"key_a": "val", "key_b": "val"}
		right := map[string]string{}

		result := diffCompare(left, right, false)

		if len(result.LeftOnly) != 2 {
			t.Errorf("expected 2 left-only, got %d", len(result.LeftOnly))
		}
		if len(result.RightOnly) != 0 {
			t.Errorf("expected 0 right-only, got %d", len(result.RightOnly))
		}
	})

	t.Run("keys are sorted alphabetically", func(t *testing.T) {
		left := map[string]string{"zebra": "1", "alpha": "2", "middle": "3"}
		right := map[string]string{}

		result := diffCompare(left, right, false)

		expected := []string{"ALPHA", "MIDDLE", "ZEBRA"}
		for i, k := range result.LeftOnly {
			if k != expected[i] {
				t.Errorf("expected key at index %d to be %s, got %s", i, expected[i], k)
			}
		}
	})

	t.Run("key conversion to env format", func(t *testing.T) {
		left := map[string]string{"my-api-key": "val", "some.config.value": "val"}
		right := map[string]string{}

		result := diffCompare(left, right, false)

		expected := []string{"MY_API_KEY", "SOME_CONFIG_VALUE"}
		for i, k := range result.LeftOnly {
			if k != expected[i] {
				t.Errorf("expected key %s, got %s", expected[i], k)
			}
		}
	})
}

func TestDiffHasDifferences(t *testing.T) {
	t.Run("no differences", func(t *testing.T) {
		result := DiffResult{Same: []string{"KEY_A"}}
		if diffHasDifferences(result, false) {
			t.Error("expected no differences")
		}
	})

	t.Run("left only is a difference", func(t *testing.T) {
		result := DiffResult{LeftOnly: []string{"KEY_A"}}
		if !diffHasDifferences(result, false) {
			t.Error("expected differences due to left-only keys")
		}
	})

	t.Run("right only is a difference", func(t *testing.T) {
		result := DiffResult{RightOnly: []string{"KEY_A"}}
		if !diffHasDifferences(result, false) {
			t.Error("expected differences due to right-only keys")
		}
	})

	t.Run("different values is a difference", func(t *testing.T) {
		result := DiffResult{Different: []string{"KEY_A"}}
		if !diffHasDifferences(result, false) {
			t.Error("expected differences due to different values")
		}
	})

	t.Run("keys-only mode ignores value differences", func(t *testing.T) {
		result := DiffResult{Different: []string{"KEY_A"}, Same: []string{"KEY_B"}}
		if diffHasDifferences(result, true) {
			t.Error("expected no differences in keys-only mode when only values differ")
		}
	})

	t.Run("keys-only mode still detects left-only", func(t *testing.T) {
		result := DiffResult{LeftOnly: []string{"KEY_A"}}
		if !diffHasDifferences(result, true) {
			t.Error("expected differences in keys-only mode for left-only keys")
		}
	})
}

func TestDiffCommandArgs(t *testing.T) {
	t.Run("requires exactly 2 args", func(t *testing.T) {
		// Cobra's ExactArgs(2) handles this - verify the command has the right validator
		err := diffCmd.Args(diffCmd, []string{"one"})
		if err == nil {
			t.Fatal("expected error for 1 arg")
		}

		err = diffCmd.Args(diffCmd, []string{})
		if err == nil {
			t.Fatal("expected error for 0 args")
		}

		err = diffCmd.Args(diffCmd, []string{"a", "b", "c"})
		if err == nil {
			t.Fatal("expected error for 3 args")
		}

		err = diffCmd.Args(diffCmd, []string{"a", "b"})
		if err != nil {
			t.Fatalf("expected no error for 2 args, got: %v", err)
		}
	})
}
