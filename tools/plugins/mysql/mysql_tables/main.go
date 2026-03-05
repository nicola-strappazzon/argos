package mysql_tables

import (
	"context"
	"fmt"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Table struct {
	Name          string  `json:"name"`
	Engine        string  `json:"engine"`
	RowFormat     string  `json:"row_format"`
	Charset       string  `json:"charset"`
	Collation     string  `json:"collation"`
	Rows          int64   `json:"row_count"`
	DataMB        float64 `json:"data_mb"`
	IndexMB       float64 `json:"index_mb"`
	TotalMB       float64 `json:"total_mb"`
	FreeMB        float64 `json:"free_mb"`
	FragPct       float64 `json:"frag_pct"`
	AutoIncrement int64   `json:"auto_increment,omitempty"`
	Comment       string  `json:"comment,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_tables",
		Description: "List tables within a MySQL database with detailed info: engine, size, charset, collation, row format, fragmentation, auto_increment, comment and timestamps.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.my.cnf using this as the section name.",
				},
				"database": map[string]any{
					"type":        "string",
					"description": "The database (schema) name to inspect.",
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
					t.table_name,
					COALESCE(t.engine, '')                                                    AS engine,
					COALESCE(t.row_format, '')                                                AS row_format,
					COALESCE(c.character_set_name, '')                                        AS charset,
					COALESCE(t.table_collation, '')                                           AS collation,
					COALESCE(t.table_rows, 0)                                                 AS row_count,
					ROUND(COALESCE(t.data_length, 0) / 1024 / 1024, 2)                       AS data_mb,
					ROUND(COALESCE(t.index_length, 0) / 1024 / 1024, 2)                      AS index_mb,
					ROUND(COALESCE(t.data_length + t.index_length, 0) / 1024 / 1024, 2)      AS total_mb,
					ROUND(COALESCE(t.data_free, 0) / 1024 / 1024, 2)                         AS free_mb,
					ROUND(COALESCE(t.data_free, 0) / NULLIF(t.data_length + t.index_length + t.data_free, 0) * 100, 1) AS frag_pct,
					COALESCE(t.auto_increment, 0)                                             AS auto_increment,
					COALESCE(t.table_comment, '')                                             AS comment,
					COALESCE(DATE_FORMAT(t.create_time, '%Y-%m-%dT%H:%i:%SZ'), '')            AS created_at,
					COALESCE(DATE_FORMAT(t.update_time, '%Y-%m-%dT%H:%i:%SZ'), '')            AS updated_at
				FROM information_schema.tables t
				LEFT JOIN information_schema.collation_character_set_applicability c
					ON c.collation_name = t.table_collation
				WHERE t.table_schema = ?
					AND t.table_type = 'BASE TABLE'
				ORDER BY total_mb DESC`

			rows, err := db.QueryContext(ctx, query, database)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			tables := make([]Table, 0)
			var totalMB, totalFreeMB float64

			for rows.Next() {
				var t Table
				if err := rows.Scan(
					&t.Name, &t.Engine, &t.RowFormat, &t.Charset, &t.Collation,
					&t.Rows, &t.DataMB, &t.IndexMB, &t.TotalMB, &t.FreeMB, &t.FragPct,
					&t.AutoIncrement, &t.Comment, &t.CreatedAt, &t.UpdatedAt,
				); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning row: %w", err)
				}
				totalMB += t.TotalMB
				totalFreeMB += t.FreeMB
				tables = append(tables, t)
			}

			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading rows: %w", err)
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance":      instanceID,
				"database":      database,
				"tables":        tables,
				"total":         len(tables),
				"total_size_mb": totalMB,
				"total_free_mb": totalFreeMB,
			}, nil
		},
	})
}
