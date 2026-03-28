package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestServerInitialize(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"

	stdin := strings.NewReader(input)
	var stdout bytes.Buffer

	server := NewServer(nil, nil, &AccessConfig{Default: "allow"})
	server.HandleSingle(stdin, &stdout)

	var resp JSONRPCResponse
	if err := json.NewDecoder(&stdout).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// JSON numbers decode as float64
	if resp.ID != float64(1) {
		t.Errorf("expected id=1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}
}

func TestServerToolsList(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n"

	stdin := strings.NewReader(input)
	var stdout bytes.Buffer

	server := NewServer(nil, nil, &AccessConfig{Default: "allow"})
	server.HandleSingle(stdin, &stdout)

	var resp JSONRPCResponse
	if err := json.NewDecoder(&stdout).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("expected tools array, got %T", result["tools"])
	}
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}
