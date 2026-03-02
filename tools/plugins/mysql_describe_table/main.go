package mysql_describe_table

import (
	"context"
	"fmt"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Column struct {
	Position  int    `json:"position"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Nullable  bool   `json:"nullable"`
	Default   string `json:"default,omitempty"`
	Charset   string `json:"charset,omitempty"`
	Collation string `json:"collation,omitempty"`
	Key       string `json:"key,omitempty"`
	Extra     string `json:"extra,omitempty"`
	Comment   string `json:"comment,omitempty"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_describe_table",
		Description: "Describe the columns of a MySQL table: type, nullability, default, charset, collation, key type, extra and comment.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.my.cnf using this as the section name.",
				},
				"database": map[string]any{
					"type":        "string",
					"description": "The database (schema) name.",
				},
				"table": map[string]any{
					"type":        "string",
					"description": "The table name to describe.",
				},
			},
			"required": []string{"db_instance_identifier", "database", "table"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			database, _ := args["database"].(string)
			table, _ := args["table"].(string)

			db, err := mysqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			query := `
				SELECT
					c.ordinal_position,
					c.column_name,
					c.column_type,
					c.is_nullable,
					COALESCE(c.column_default, ''),
					COALESCE(c.character_set_name, ''),
					COALESCE(c.collation_name, ''),
					COALESCE(c.column_key, ''),
					COALESCE(c.extra, ''),
					COALESCE(c.column_comment, '')
				FROM information_schema.columns c
				WHERE c.table_schema = ?
					AND c.table_name = ?
				ORDER BY c.ordinal_position`

			rows, err := db.QueryContext(ctx, query, database, table)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			columns := make([]Column, 0)
			for rows.Next() {
				var col Column
				var nullable string
				if err := rows.Scan(
					&col.Position, &col.Name, &col.Type, &nullable,
					&col.Default, &col.Charset, &col.Collation,
					&col.Key, &col.Extra, &col.Comment,
				); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning row: %w", err)
				}
				col.Nullable = nullable == "YES"
				columns = append(columns, col)
			}

			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading rows: %w", err)
			}

			if len(columns) == 0 {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("table %s.%s not found", database, table)
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"database": database,
				"table":    table,
				"columns":  columns,
				"total":    len(columns),
			}, nil
		},
	})
}
