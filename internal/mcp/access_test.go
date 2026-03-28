package mcp

import "testing"

func TestAccessCheck(t *testing.T) {
	rules := &AccessConfig{
		Default: "deny",
		Rules: []AccessRule{
			{Project: "myapp", Access: "allow"},
			{Project: "production", Access: "prompt"},
			{Project: "infra/*", Access: "deny"},
		},
	}

	tests := []struct {
		project  string
		expected string
	}{
		{"myapp", "allow"},
		{"production", "prompt"},
		{"infra/secrets", "deny"},
		{"infra/keys", "deny"},
		{"unknown", "deny"},           // falls to default
		{"myapp-staging", "deny"},     // exact match only for non-glob
	}

	for _, tt := range tests {
		t.Run(tt.project, func(t *testing.T) {
			result := rules.Check(tt.project)
			if result != tt.expected {
				t.Errorf("Check(%q) = %q, want %q", tt.project, result, tt.expected)
			}
		})
	}
}

func TestAccessCheckDefaultPrompt(t *testing.T) {
	rules := &AccessConfig{
		Default: "",
		Rules:   nil,
	}
	result := rules.Check("anything")
	if result != "prompt" {
		t.Errorf("expected default 'prompt', got %q", result)
	}
}

func TestAccessConfigNoFile(t *testing.T) {
	config, err := LoadAccessConfig("/nonexistent/path/access.json")
	if err != nil {
		t.Fatal(err)
	}
	if config.Default != "prompt" {
		t.Errorf("expected default 'prompt' when no file, got %q", config.Default)
	}
}
