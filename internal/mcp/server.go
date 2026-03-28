package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hungnguyen18/uzp-cli/internal/storage"
)

// JSONRPCRequest is a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError is a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// VaultUnlocker is called to unlock the vault before first tool use.
type VaultUnlocker func() error

// Server handles MCP protocol over stdio.
type Server struct {
	vault    *storage.Vault
	unlocker VaultUnlocker
	access   *AccessConfig
}

// NewServer creates a new MCP server.
func NewServer(vault *storage.Vault, unlocker VaultUnlocker, access *AccessConfig) *Server {
	return &Server{
		vault:    vault,
		unlocker: unlocker,
		access:   access,
	}
}

// Run starts the MCP server, reading from stdin and writing to stdout.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer for large messages
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp := s.handleRequest(line)
		if resp == nil {
			continue // notification, no response needed
		}

		data, err := json.Marshal(resp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
			continue
		}
		fmt.Fprintf(os.Stdout, "%s\n", data)
	}

	return scanner.Err()
}

// HandleSingle processes a single request from reader and writes response to writer.
// Used for testing.
func (s *Server) HandleSingle(reader io.Reader, writer io.Writer) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	if scanner.Scan() {
		resp := s.handleRequest(scanner.Bytes())
		if resp != nil {
			data, _ := json.Marshal(resp)
			fmt.Fprintf(writer, "%s\n", data)
		}
	}
}

func (s *Server) handleRequest(data []byte) *JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &JSONRPCError{Code: -32700, Message: "Parse error"},
		}
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.ID)
	case "notifications/initialized":
		return nil // notification, no response
	case "tools/list":
		return s.handleToolsList(req.ID)
	case "tools/call":
		return s.handleToolsCall(req.ID, req.Params)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONRPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}
	}
}

func (s *Server) handleInitialize(id interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "uzp",
				"version": "1.0.0",
			},
		},
	}
}

func (s *Server) handleToolsList(id interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"tools": toolDefinitions(),
		},
	}
}

func (s *Server) handleToolsCall(id interface{}, params json.RawMessage) *JSONRPCResponse {
	// Ensure vault is unlocked on first tool call
	if s.unlocker != nil && s.vault != nil && !s.vault.IsUnlocked() {
		if err := s.unlocker(); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONRPCError{Code: -32000, Message: "Vault unlock failed: " + err.Error()},
			}
		}
	}

	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &JSONRPCError{Code: -32602, Message: "Invalid params"},
		}
	}

	result, err := s.executeTool(call.Name, call.Arguments)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "Error: " + err.Error()},
				},
				"isError": true,
			},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": result},
			},
		},
	}
}
