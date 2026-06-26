package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpServer wraps an embedded MCP HTTP server that exposes eino tools
// to Claude Code CLI. The server is started on a random localhost port
// and the CLI is configured to connect to it via --mcp-config.
type mcpServer struct {
	listener net.Listener
	port     int
}

// newMCPServer creates an MCP server with the given eino tools registered.
// The server listens on 127.0.0.1:0 (random port). Caller must call close()
// when done to release the port.
func newMCPServer(tools []tool.InvokableTool) (*mcpServer, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "eino-tools",
		Version: "v0.1.0",
	}, nil)

	ctx := context.Background()

	for _, t := range tools {
		if err := registerEinoTool(server, ctx, t); err != nil {
			return nil, fmt.Errorf("register tool: %w", err)
		}
	}

	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		},
	)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start MCP listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	go http.Serve(listener, handler) //nolint:errcheck

	return &mcpServer{
		listener: listener,
		port:     port,
	}, nil
}

// mcpConfigJSON returns the --mcp-config JSON for this server.
func (m *mcpServer) mcpConfigJSON() string {
	config := map[string]any{
		"mcpServers": map[string]any{
			"eino-tools": map[string]any{
				"type": "http",
				"url":  fmt.Sprintf("http://127.0.0.1:%d/mcp", m.port),
			},
		},
	}
	b, _ := json.Marshal(config)
	return string(b)
}

// close shuts down the MCP HTTP server.
func (m *mcpServer) close() {
	m.listener.Close()
}

// registerEinoTool registers a single eino tool with the MCP server.
func registerEinoTool(server *mcp.Server, ctx context.Context, t tool.InvokableTool) error {
	info, err := t.Info(ctx)
	if err != nil {
		return fmt.Errorf("get tool info: %w", err)
	}

	schema, err := einoParamsToJSONSchema(info)
	if err != nil {
		return fmt.Errorf("convert schema for tool %q: %w", info.Name, err)
	}
	mcpTool := &mcp.Tool{
		Name:        info.Name,
		Description: info.Desc,
		InputSchema: schema,
	}

	// Capture t for the closure.
	einoTool := t
	server.AddTool(mcpTool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsJSON := string(req.Params.Arguments)
		if argsJSON == "" || argsJSON == "null" {
			argsJSON = "{}"
		}

		result, err := einoTool.InvokableRun(ctx, argsJSON)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil
	})

	return nil
}

// einoParamsToJSONSchema converts eino ParamsOneOf to a map[string]any
// compatible with MCP tool inputSchema requirements.
// Returns an error if the schema conversion fails.
func einoParamsToJSONSchema(info *schema.ToolInfo) (map[string]any, error) {
	// No params at all → empty object schema is a valid default.
	if info.ParamsOneOf == nil {
		return map[string]any{"type": "object"}, nil
	}

	// Use eino's built-in ToJSONSchema, then marshal → unmarshal to map.
	js, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil {
		return nil, fmt.Errorf("eino ToJSONSchema: %w", err)
	}
	if js == nil {
		return nil, fmt.Errorf("eino ToJSONSchema returned nil")
	}

	b, err := json.Marshal(js)
	if err != nil {
		return nil, fmt.Errorf("marshal JSON schema: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("unmarshal JSON schema: %w", err)
	}

	return result, nil
}
