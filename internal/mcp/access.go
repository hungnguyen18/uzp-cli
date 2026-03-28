package mcp

import (
	"encoding/json"
	"os"
	"path"
)

// AccessRule defines access level for a project pattern.
type AccessRule struct {
	Project string `json:"project"`
	Access  string `json:"access"` // "allow", "prompt", or "deny"
}

// AccessConfig holds the full access control configuration.
type AccessConfig struct {
	Default string       `json:"default"`
	Rules   []AccessRule `json:"rules"`
}

// LoadAccessConfig loads access rules from the given path.
// Returns a safe default (prompt for all) if the file does not exist.
func LoadAccessConfig(filePath string) (*AccessConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &AccessConfig{Default: "prompt"}, nil
		}
		return nil, err
	}

	var config AccessConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Safe default if not specified
	if config.Default == "" {
		config.Default = "prompt"
	}

	return &config, nil
}

// Check returns the access level for a given project name.
// Evaluates rules in order; first match wins. Falls back to Default.
func (c *AccessConfig) Check(project string) string {
	for _, rule := range c.Rules {
		matched, err := path.Match(rule.Project, project)
		if err != nil {
			continue
		}
		if matched {
			return rule.Access
		}
	}

	if c.Default == "" {
		return "prompt"
	}
	return c.Default
}
