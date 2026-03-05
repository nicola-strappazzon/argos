package mysql_variables

import (
	"context"
	"fmt"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_variables",
		Description: "Show MySQL global variables (SHOW GLOBAL VARIABLES) for an RDS instance. Optionally filter by variable name prefix.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.my.cnf using this as the section name.",
				},
				"like": map[string]any{
					"type":        "string",
					"description": "Optional LIKE pattern to filter variables (e.g. 'innodb%', 'max%').",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			like, _ := args["like"].(string)

			db, err := mysqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			query := "SHOW GLOBAL VARIABLES"
			if like != "" {
				query = fmt.Sprintf("SHOW GLOBAL VARIABLES LIKE '%s'", like)
			}

			rows, err := db.QueryContext(ctx, query)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			variables := make(map[string]string)
			for rows.Next() {
				var name, value string
				if err := rows.Scan(&name, &value); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning row: %w", err)
				}
				variables[name] = value
			}

			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading rows: %w", err)
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance":  instanceID,
				"variables": variables,
				"total":     len(variables),
			}, nil
		},
	})
}
