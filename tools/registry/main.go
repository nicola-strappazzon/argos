package registry

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var Tools = map[string]Property{}

type Property struct {
	Name        string
	Description string
	InputSchema map[string]any
	Function    func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error)
}

func Add(p Property) {
	Tools[p.Name] = p
}
