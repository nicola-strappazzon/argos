package mysql_ping

import (
	"context"
	"time"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_ping",
		Description: "Test the connection to a MySQL instance. Returns success status and round-trip latency in milliseconds.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.my.cnf using this as the section name.",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)

			db, err := mysqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, map[string]any{
					"instance":   instanceID,
					"success":    false,
					"error":      err.Error(),
					"latency_ms": 0,
				}, nil
			}
			defer db.Close()

			start := time.Now()
			if err := db.PingContext(ctx); err != nil {
				return &mcp.CallToolResult{}, map[string]any{
					"instance":   instanceID,
					"success":    false,
					"error":      err.Error(),
					"latency_ms": 0,
				}, nil
			}
			latency := time.Since(start).Milliseconds()

			return &mcp.CallToolResult{}, map[string]any{
				"instance":   instanceID,
				"success":    true,
				"latency_ms": latency,
			}, nil
		},
	})
}
