package mysql_documentation

import (
	"context"
	"fmt"
	"strings"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Unsigned bool   `json:"unsigned"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

type Table struct {
	Name    string   `json:"name"`
	Comment string   `json:"comment,omitempty"`
	Columns []Column `json:"columns"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_documentation",
		Description: "List all tables and their columns in a MySQL database grouped by table, including table and column comments, types, nullability and defaults.",
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
			},
			"required": []string{"db_instance_identifier", "database"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			database, _ := args["database"].(string)

			db, err := mysqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			query := `
				SELECT
					c.table_name,
					COALESCE(t.table_comment, ''),
					c.column_name,
					c.column_type,
					c.is_nullable,
					COALESCE(c.column_default, ''),
					COALESCE(c.column_comment, '')
				FROM information_schema.columns c
				INNER JOIN information_schema.tables t
					ON t.table_schema = c.table_schema
					AND t.table_name = c.table_name
					AND t.table_type = 'BASE TABLE'
				WHERE c.table_schema = ?
				ORDER BY c.table_name, c.ordinal_position`

			rows, err := db.QueryContext(ctx, query, database)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			tableMap := make(map[string]*Table)
			tableOrder := make([]string, 0)

			for rows.Next() {
				var tableName, tableComment, colName, colType, nullable, colDefault, colComment string
				if err := rows.Scan(&tableName, &tableComment, &colName, &colType, &nullable, &colDefault, &colComment); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning row: %w", err)
				}

				if _, exists := tableMap[tableName]; !exists {
					tableMap[tableName] = &Table{
						Name:    tableName,
						Comment: tableComment,
						Columns: make([]Column, 0),
					}
					tableOrder = append(tableOrder, tableName)
				}

				tableMap[tableName].Columns = append(tableMap[tableName].Columns, Column{
					Name:     colName,
					Type:     colType,
					Unsigned: strings.Contains(strings.ToLower(colType), "unsigned"),
					Nullable: nullable == "YES",
					Default:  colDefault,
					Comment:  colComment,
				})
			}

			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading rows: %w", err)
			}

			tables := make([]Table, 0, len(tableOrder))
			for _, name := range tableOrder {
				tables = append(tables, *tableMap[name])
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"database": database,
				"tables":   tables,
				"total":    len(tables),
			}, nil
		},
	})
}
