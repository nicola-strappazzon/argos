package main

import (
	"context"
	"log"

	_ "github.com/nicola-strappazzon/mcp/tools/plugins"
	"github.com/nicola-strappazzon/mcp/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "My personal assistant"}, nil)

	for key := range registry.Tools {
		if tool, ok := registry.Tools[key]; ok {
			log.Printf("Load tool: %s", tool.Name)

			inputSchema := tool.InputSchema
			if inputSchema == nil {
				inputSchema = map[string]any{"type": "object"}
			}

			mcp.AddTool(server, &mcp.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: inputSchema,
			},
				tool.Function,
			)
		}
	}

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
