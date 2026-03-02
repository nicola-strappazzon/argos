package mysql_databases

import (
	"context"
	"database/sql"
	"fmt"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Database struct {
	Name      string  `json:"name"`
	Charset   string  `json:"charset"`
	Collation string  `json:"collation"`
	SizeMB    float64 `json:"size_mb"`
	Tables    int64   `json:"tables"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_databases",
		Description: "List databases on a MySQL instance with their size, character set, collation and table count.",
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
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			query := `
				SELECT
					s.schema_name,
					s.default_character_set_name,
					s.default_collation_name,
					ROUND(COALESCE(SUM(t.data_length + t.index_length), 0) / 1024 / 1024, 2) AS size_mb,
					COUNT(t.table_name) AS tables
				FROM information_schema.schemata s
				LEFT JOIN information_schema.tables t ON t.table_schema = s.schema_name
				GROUP BY s.schema_name, s.default_character_set_name, s.default_collation_name
				ORDER BY size_mb DESC`

			rows, err := db.QueryContext(ctx, query)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			databases := make([]Database, 0)
			for rows.Next() {
				var d Database
				if err := rows.Scan(&d.Name, &d.Charset, &d.Collation, &d.SizeMB, &d.Tables); err != nil {
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
				"instance":     instanceID,
				"databases":    databases,
				"total":        len(databases),
				"total_size_mb": sql.NullFloat64{Float64: totalMB, Valid: true}.Float64,
			}, nil
		},
	})
}
