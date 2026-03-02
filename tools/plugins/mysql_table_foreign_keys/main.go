package mysql_table_foreign_keys

import (
	"context"
	"fmt"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ForeignKey struct {
	Constraint       string `json:"constraint"`
	Column           string `json:"column"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
	OnUpdate         string `json:"on_update"`
	OnDelete         string `json:"on_delete"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_table_foreign_keys",
		Description: "List foreign keys of a MySQL table: outgoing FKs (this table references others) and incoming FKs (other tables reference this table).",
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
					"description": "The table name to inspect foreign keys for.",
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

			// Outgoing: this table references other tables.
			outRows, err := db.QueryContext(ctx, `
				SELECT
					kcu.constraint_name,
					kcu.column_name,
					kcu.referenced_table_name,
					kcu.referenced_column_name,
					rc.update_rule,
					rc.delete_rule
				FROM information_schema.key_column_usage kcu
				INNER JOIN information_schema.referential_constraints rc
					ON rc.constraint_name = kcu.constraint_name
					AND rc.constraint_schema = kcu.table_schema
				WHERE kcu.table_schema = ?
					AND kcu.table_name = ?
					AND kcu.referenced_table_name IS NOT NULL
				ORDER BY kcu.constraint_name, kcu.ordinal_position`,
				database, table)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("querying outgoing FKs: %w", err)
			}
			defer outRows.Close()

			outgoing := make([]ForeignKey, 0)
			for outRows.Next() {
				var fk ForeignKey
				if err := outRows.Scan(&fk.Constraint, &fk.Column, &fk.ReferencedTable, &fk.ReferencedColumn, &fk.OnUpdate, &fk.OnDelete); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning outgoing FK: %w", err)
				}
				outgoing = append(outgoing, fk)
			}
			if err := outRows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			// Incoming: other tables reference this table.
			inRows, err := db.QueryContext(ctx, `
				SELECT
					kcu.constraint_name,
					kcu.column_name,
					kcu.table_name,
					kcu.referenced_column_name,
					rc.update_rule,
					rc.delete_rule
				FROM information_schema.key_column_usage kcu
				INNER JOIN information_schema.referential_constraints rc
					ON rc.constraint_name = kcu.constraint_name
					AND rc.constraint_schema = kcu.table_schema
				WHERE kcu.table_schema = ?
					AND kcu.referenced_table_name = ?
				ORDER BY kcu.table_name, kcu.constraint_name`,
				database, table)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("querying incoming FKs: %w", err)
			}
			defer inRows.Close()

			incoming := make([]ForeignKey, 0)
			for inRows.Next() {
				var fk ForeignKey
				if err := inRows.Scan(&fk.Constraint, &fk.Column, &fk.ReferencedTable, &fk.ReferencedColumn, &fk.OnUpdate, &fk.OnDelete); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning incoming FK: %w", err)
				}
				incoming = append(incoming, fk)
			}
			if err := inRows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance":              instanceID,
				"database":              database,
				"table":                 table,
				"outgoing_foreign_keys": outgoing,
				"incoming_foreign_keys": incoming,
				"total_outgoing":        len(outgoing),
				"total_incoming":        len(incoming),
			}, nil
		},
	})
}
