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
