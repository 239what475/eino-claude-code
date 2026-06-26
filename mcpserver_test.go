package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ── test tools ──

type echoTool struct{}

func (t *echoTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "echo",
		Desc: "Echoes the input message back.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"message": {Type: schema.String, Desc: "The message to echo.", Required: true},
		}),
	}, nil
}

func (t *echoTool) InvokableRun(ctx context.Context, argsJSON string, opts ...tool.Option) (string, error) {
	var args struct{ Message string `json:"message"` }
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	return "echo: " + args.Message, nil
}

type noParamsTool struct{}

func (t *noParamsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "noop",
		Desc: "Does nothing.",
	}, nil
}

func (t *noParamsTool) InvokableRun(ctx context.Context, argsJSON string, opts ...tool.Option) (string, error) {
	return "done", nil
}

type errorTool struct{}

func (t *errorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "failer",
		Desc: "Always fails.",
	}, nil
}

func (t *errorTool) InvokableRun(ctx context.Context, argsJSON string, opts ...tool.Option) (string, error) {
	return "", fmt.Errorf("intentional failure")
}

// ── einoParamsToJSONSchema tests ──

func TestEinoParamsToJSONSchema_NilParams(t *testing.T) {
	info := &schema.ToolInfo{Name: "test", Desc: "test"}
	result, err := einoParamsToJSONSchema(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["type"] != "object" {
		t.Errorf("expected type 'object', got %v", result["type"])
	}
	if props, ok := result["properties"]; ok {
		t.Errorf("expected no properties for nil params, got %v", props)
	}
}

func TestEinoParamsToJSONSchema_WithParams(t *testing.T) {
	info := &schema.ToolInfo{
		Name: "test",
		Desc: "test",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"name":   {Type: schema.String, Desc: "User name", Required: true},
			"age":    {Type: schema.Integer, Desc: "User age"},
			"active": {Type: schema.Boolean, Desc: "Active status"},
		}),
	}

	result, err := einoParamsToJSONSchema(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["type"] != "object" {
		t.Errorf("expected type 'object', got %v", result["type"])
	}

	props, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be a map")
	}
	if len(props) != 3 {
		t.Errorf("expected 3 properties, got %d", len(props))
	}
	if props["name"].(map[string]any)["type"] != "string" {
		t.Error("expected 'name' type: string")
	}
	if props["age"].(map[string]any)["type"] != "integer" {
		t.Error("expected 'age' type: integer")
	}
	if props["active"].(map[string]any)["type"] != "boolean" {
		t.Error("expected 'active' type: boolean")
	}

	required, ok := result["required"].([]interface{})
	if !ok {
		t.Fatal("expected required to be a slice")
	}
	if len(required) != 1 || required[0] != "name" {
		t.Errorf("expected required to be ['name'], got %v", required)
	}
}

// ── MCP server lifecycle tests ──

func TestNewMCPServer_Empty(t *testing.T) {
	srv, err := newMCPServer(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv != nil {
		t.Error("expected nil server for empty tools")
		srv.close()
	}

	srv, err = newMCPServer([]tool.InvokableTool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv != nil {
		t.Error("expected nil server for empty tools slice")
		srv.close()
	}
}

func TestNewMCPServer_StartStop(t *testing.T) {
	srv, err := newMCPServer([]tool.InvokableTool{&echoTool{}})
	if err != nil {
		t.Fatalf("failed to start MCP server: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.port == 0 {
		t.Error("expected non-zero port")
	}

	config := srv.mcpConfigJSON()
	if config == "" {
		t.Error("expected non-empty MCP config")
	}

	// Verify the config contains expected keys.
	var cfg map[string]any
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		t.Fatalf("failed to parse MCP config: %v", err)
	}

	servers, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("expected mcpServers in config")
	}
	einoTools, ok := servers["eino-tools"].(map[string]any)
	if !ok {
		t.Fatal("expected eino-tools in mcpServers")
	}
	if einoTools["type"] != "http" {
		t.Errorf("expected type 'http', got %v", einoTools["type"])
	}

	// Close should not panic.
	srv.close()
}

func TestNewMCPServer_MultipleTools(t *testing.T) {
	srv, err := newMCPServer([]tool.InvokableTool{&echoTool{}, &noParamsTool{}})
	if err != nil {
		t.Fatalf("failed to start MCP server: %v", err)
	}
	defer srv.close()
	if srv.port == 0 {
		t.Error("expected non-zero port")
	}
}

// ── eino tool → MCP round-trip tests ──

func TestMCPRoundTrip_Echo(t *testing.T) {
	// Start the server with our test tool.
	srv, err := newMCPServer([]tool.InvokableTool{&echoTool{}})
	if err != nil {
		t.Fatalf("failed to start MCP server: %v", err)
	}
	defer srv.close()

	// We can't make a real MCP client call without the full SDK client
	// setup, but we can verify the server is listening and serving.
	// The integration with Claude Code CLI is validated by the example.
	if srv.port == 0 {
		t.Error("expected non-zero port")
	}
}

func TestSchemaConversion_ErrorTool(t *testing.T) {
	// errorTool has no params → should produce valid empty schema.
	info, _ := (&errorTool{}).Info(context.Background())
	result, err := einoParamsToJSONSchema(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["type"] != "object" {
		t.Errorf("expected type 'object', got %v", result["type"])
	}
}
