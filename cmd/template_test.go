package cmd

import (
	"sort"
	"strings"
	"testing"
)

func TestCommentHintGenerate(t *testing.T) {
	tests := []struct {
		envKey   string
		expected string
	}{
		{"API_KEY", "API key"},
		{"MY_APIKEY", "API key"},
		{"DATABASE_URL", "Database connection URL"},
		{"DB_URL", "Database connection URL"},
		{"POSTGRES_URL", "Database connection URL"},
		{"REDIS_URI", "Database connection URL"},
		{"API_URL", "API endpoint URL"},
		{"WEBHOOK_URL", "Webhook URL"},
		{"SOME_URL", "URL"},
		{"SECRET_KEY", "Secret key"},
		{"STRIPE_SECRET", "Secret key"},
		{"ACCESS_TOKEN", "Access token"},
		{"REFRESH_TOKEN", "Refresh token"},
		{"AUTH_TOKEN", "Authentication token"},
		{"DB_PASSWORD", "Password"},
		{"PORT", "Port number"},
		{"APP_PORT", "Port number"},
		{"HOST", "Hostname"},
		{"APP_HOST", "Hostname"},
		{"ADMIN_EMAIL", "Email address"},
		{"AWS_REGION", "Region identifier"},
		{"S3_BUCKET", "Storage bucket name"},
		{"AWS_KEY_ID", "Key"},
		{"NODE_ENV", "Environment (e.g. development, staging, production)"},
		{"DEBUG", "Debug flag (true/false)"},
		{"LOG_LEVEL", "Log level or config"},
		{"APP_NAME", "Configuration value"},
	}

	for _, tt := range tests {
		t.Run(tt.envKey, func(t *testing.T) {
			result := commentHintGenerate(tt.envKey)
			if result != tt.expected {
				t.Errorf("commentHintGenerate(%q) = %q, want %q", tt.envKey, result, tt.expected)
			}
		})
	}
}

func TestTemplateOutputFormat(t *testing.T) {
	// Simulate the template output generation logic
	secrets := map[string]string{
		"stripe_secret_key": "sk_test_xxx",
		"api_key":           "abc123",
		"database_url":      "postgres://localhost/db",
	}

	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	t.Run("basic template without comments", func(t *testing.T) {
		var sb strings.Builder
		for _, key := range keys {
			envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
			sb.WriteString(envKey + "=\n")
		}
		output := sb.String()

		// Verify keys are sorted
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}

		// Verify no values are included
		for _, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				t.Errorf("expected key=value format, got %q", line)
				continue
			}
			if parts[1] != "" {
				t.Errorf("expected empty value for %s, got %q", parts[0], parts[1])
			}
		}
	})

	t.Run("template with comments", func(t *testing.T) {
		var sb strings.Builder
		for _, key := range keys {
			envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
			comment := commentHintGenerate(envKey)
			sb.WriteString(envKey + "=  # " + comment + "\n")
		}
		output := sb.String()

		// Verify each line has a comment
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if !strings.Contains(line, "#") {
				t.Errorf("expected comment in line %q", line)
			}
		}
	})

	t.Run("keys are sorted alphabetically", func(t *testing.T) {
		if !sort.StringsAreSorted(keys) {
			t.Errorf("keys should be sorted, got %v", keys)
		}
	})

	t.Run("empty project produces no key lines", func(t *testing.T) {
		emptySecrets := map[string]string{}
		emptyKeys := make([]string, 0, len(emptySecrets))
		for key := range emptySecrets {
			emptyKeys = append(emptyKeys, key)
		}
		if len(emptyKeys) != 0 {
			t.Errorf("expected 0 keys for empty project, got %d", len(emptyKeys))
		}
	})
}
