package postgresql_tables

import (
	"context"
	"database/sql"
	"fmt"

	psqldriver "github.com/nicola-strappazzon/argos/internal/drivers/postgresql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Table struct {
	Schema       string  `json:"schema"`
	Name         string  `json:"name"`
	Owner        string  `json:"owner"`
	AccessMethod string  `json:"access_method"`
	Rows         int64   `json:"row_count"`
	DeadTuples   int64   `json:"dead_tuples"`
	DataMB       float64 `json:"data_mb"`
	IndexMB      float64 `json:"index_mb"`
	TotalMB      float64 `json:"total_mb"`
	Comment      string  `json:"comment,omitempty"`
	LastVacuum   string  `json:"last_vacuum,omitempty"`
	LastAnalyze  string  `json:"last_analyze,omitempty"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "postgresql_tables",
		Description: "List tables within a PostgreSQL database with detailed info: schema, owner, access method, estimated row count, dead tuples, size (data/index/total), comment and last vacuum/analyze timestamps.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.pgpass using the instance identifier to match the hostname.",
				},
				"database": map[string]any{
					"type":        "string",
					"description": "The database name to inspect.",
				},
			},
			"required": []string{"db_instance_identifier", "database"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			database, _ := args["database"].(string)

			db, err := psqldriver.ConnectDB(instanceID, database)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			query := `
				SELECT
					n.nspname                                                                           AS schema,
					c.relname                                                                           AS name,
					r.rolname                                                                           AS owner,
					COALESCE(am.amname, '')                                                             AS access_method,
					GREATEST(c.reltuples::bigint, 0)                                                   AS row_count,
					COALESCE(s.n_dead_tup, 0)                                                          AS dead_tuples,
					ROUND(pg_relation_size(c.oid)       / 1024.0 / 1024.0, 2)                          AS data_mb,
					ROUND(pg_indexes_size(c.oid)        / 1024.0 / 1024.0, 2)                          AS index_mb,
					ROUND(pg_total_relation_size(c.oid) / 1024.0 / 1024.0, 2)                          AS total_mb,
					COALESCE(obj_description(c.oid, 'pg_class'), '')                                   AS comment,
					COALESCE(TO_CHAR(GREATEST(s.last_vacuum, s.last_autovacuum),   'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '') AS last_vacuum,
					COALESCE(TO_CHAR(GREATEST(s.last_analyze, s.last_autoanalyze), 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '') AS last_analyze
				FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				JOIN pg_roles r ON r.oid = c.relowner
				LEFT JOIN pg_am am ON am.oid = c.relam
				LEFT JOIN pg_stat_user_tables s ON s.relid = c.oid
				WHERE c.relkind = 'r'
					AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
				ORDER BY total_mb DESC`

			rows, err := db.QueryContext(ctx, query)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			tables := make([]Table, 0)
			var totalMB float64

			for rows.Next() {
				var t Table
				var comment, lastVacuum, lastAnalyze sql.NullString
				if err := rows.Scan(
					&t.Schema, &t.Name, &t.Owner, &t.AccessMethod,
					&t.Rows, &t.DeadTuples,
					&t.DataMB, &t.IndexMB, &t.TotalMB,
					&comment, &lastVacuum, &lastAnalyze,
				); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning row: %w", err)
				}
				t.Comment = comment.String
				t.LastVacuum = lastVacuum.String
				t.LastAnalyze = lastAnalyze.String
				totalMB += t.TotalMB
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
			}, nil
		},
	})
}
