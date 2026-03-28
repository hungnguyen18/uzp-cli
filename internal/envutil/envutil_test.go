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
