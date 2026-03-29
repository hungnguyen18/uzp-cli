package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileCopy(t *testing.T) {
	// Create a temp source file
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	content := []byte("test vault content")
	if err := os.WriteFile(srcPath, content, 0600); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Test copy
	if err := fileCopy(srcPath, dstPath); err != nil {
		t.Fatalf("fileCopy failed: %v", err)
	}

	// Verify content
	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}

	// Verify permissions
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("failed to stat dest file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions mismatch: got %o, want 0600", info.Mode().Perm())
	}
}

func TestFileCopySourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	err := fileCopy(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest"))
	if err == nil {
		t.Error("expected error for nonexistent source, got nil")
	}
}

func TestFileValidateVault(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid vault file", func(t *testing.T) {
		vaultData := map[string]string{
			"salt": "dGVzdHNhbHQ=",
			"hash": "testhash",
			"data": "dGVzdGRhdGE=",
		}
		data, _ := json.Marshal(vaultData)
		path := filepath.Join(tmpDir, "valid.vault")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		if err := fileValidateVault(path); err != nil {
			t.Errorf("expected valid vault, got error: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.vault")
		if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		if err := fileValidateVault(path); err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		vaultData := map[string]string{
			"salt": "dGVzdHNhbHQ=",
		}
		data, _ := json.Marshal(vaultData)
		path := filepath.Join(tmpDir, "missing.vault")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		if err := fileValidateVault(path); err == nil {
			t.Error("expected error for missing fields, got nil")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		if err := fileValidateVault(filepath.Join(tmpDir, "nonexistent")); err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})
}
