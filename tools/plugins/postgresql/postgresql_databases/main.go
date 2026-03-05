package postgresql_databases

import (
	"context"
	"fmt"

	psqldriver "github.com/nicola-strappazzon/argos/internal/drivers/postgresql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Database struct {
	Name            string  `json:"name"`
	Owner           string  `json:"owner"`
	Encoding        string  `json:"encoding"`
	Collation       string  `json:"collation"`
	ConnectionLimit int64   `json:"connection_limit"`
	SizeMB          float64 `json:"size_mb"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "postgresql_databases",
		Description: "List databases on a PostgreSQL instance with their size (MB), encoding, collation, owner and connection limit.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.pgpass using the instance identifier to match the hostname.",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)

			db, err := psqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			query := `
				SELECT
					d.datname,
					r.rolname                                                   AS owner,
					pg_encoding_to_char(d.encoding)                             AS encoding,
					d.datcollate                                                AS collation,
					d.datconnlimit                                              AS connection_limit,
					ROUND(pg_database_size(d.datname) / 1024.0 / 1024.0, 2)   AS size_mb
				FROM pg_database d
				JOIN pg_roles r ON r.oid = d.datdba
				WHERE d.datistemplate = false
				ORDER BY size_mb DESC`

			rows, err := db.QueryContext(ctx, query)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			databases := make([]Database, 0)
			for rows.Next() {
				var d Database
				if err := rows.Scan(&d.Name, &d.Owner, &d.Encoding, &d.Collation, &d.ConnectionLimit, &d.SizeMB); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning row: %w", err)
				}
				databases = append(databases, d)
			}

			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading rows: %w", err)
			}

			var totalMB float64
			for _, d := range databases {
				totalMB += d.SizeMB
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance":      instanceID,
				"databases":     databases,
				"total":         len(databases),
				"total_size_mb": totalMB,
			}, nil
		},
	})
}
