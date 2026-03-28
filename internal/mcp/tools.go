package mcp

import (
	"encoding/json"
	"fmt"
	"os"
)

func toolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "uzp_get",
			"description": "Get a single secret value from the vault",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project": map[string]interface{}{"type": "string", "description": "Project name"},
					"key":     map[string]interface{}{"type": "string", "description": "Secret key name"},
				},
				"required": []string{"project", "key"},
			},
		},
		{
			"name":        "uzp_list",
			"description": "List projects and their keys (values are not returned)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project": map[string]interface{}{"type": "string", "description": "Optional: filter by project name"},
				},
			},
		},
		{
			"name":        "uzp_search",
			"description": "Search for projects and keys matching a keyword",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keyword": map[string]interface{}{"type": "string", "description": "Search keyword"},
				},
				"required": []string{"keyword"},
			},
		},
	}
}

func (s *Server) executeTool(name string, argsRaw json.RawMessage) (string, error) {
	switch name {
	case "uzp_get":
		return s.toolGet(argsRaw)
	case "uzp_list":
		return s.toolList(argsRaw)
	case "uzp_search":
		return s.toolSearch(argsRaw)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *Server) toolGet(argsRaw json.RawMessage) (string, error) {
	var args struct {
		Project string `json:"project"`
		Key     string `json:"key"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Access control check
	access := s.access.Check(args.Project)
	switch access {
	case "deny":
		return "", fmt.Errorf("access denied for project '%s'", args.Project)
	case "prompt":
		if !promptUserApproval(args.Project, args.Key) {
			return "", fmt.Errorf("access denied by user for project '%s'", args.Project)
		}
	}

	value, err := s.vault.Get(args.Project, args.Key)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *Server) toolList(argsRaw json.RawMessage) (string, error) {
	var args struct {
		Project string `json:"project"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Project != "" {
		// Access control for specific project
		access := s.access.Check(args.Project)
		if access == "deny" {
			return "", fmt.Errorf("access denied for project '%s'", args.Project)
		}

		secrets, err := s.vault.GetProjectSecrets(args.Project)
		if err != nil {
			return "", err
		}
		keys := make([]string, 0, len(secrets))
		for k := range secrets {
			keys = append(keys, k)
		}
		data, _ := json.MarshalIndent(map[string]interface{}{args.Project: keys}, "", "  ")
		return string(data), nil
	}

	// List all projects (no access control on listing project names)
	projects, err := s.vault.List()
	if err != nil {
		return "", err
	}
	data, _ := json.MarshalIndent(projects, "", "  ")
	return string(data), nil
}

func (s *Server) toolSearch(argsRaw json.RawMessage) (string, error) {
	var args struct {
		Keyword string `json:"keyword"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	results, err := s.vault.Search(args.Keyword)
	if err != nil {
		return "", err
	}
	data, _ := json.MarshalIndent(results, "", "  ")
	return string(data), nil
}

// promptUserApproval asks the user on stderr whether to allow access.
func promptUserApproval(project, key string) bool {
	if key != "" {
		fmt.Fprintf(os.Stderr, "MCP agent requests access to '%s/%s'. Allow? [y/N]: ", project, key)
	} else {
		fmt.Fprintf(os.Stderr, "MCP agent requests access to project '%s'. Allow? [y/N]: ", project)
	}

	var response string
	fmt.Fscanln(os.Stderr, &response)

	return response == "y" || response == "Y" || response == "yes"
}
