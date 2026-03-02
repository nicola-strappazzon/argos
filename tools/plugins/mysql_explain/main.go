package mysql_explain

import (
	"context"
	"database/sql"
	"fmt"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ExplainRow struct {
	ID           int64   `json:"id"`
	SelectType   string  `json:"select_type"`
	Table        string  `json:"table,omitempty"`
	Partitions   string  `json:"partitions,omitempty"`
	AccessType   string  `json:"access_type"`
	PossibleKeys string  `json:"possible_keys,omitempty"`
	Key          string  `json:"key,omitempty"`
	KeyLen       string  `json:"key_len,omitempty"`
	Ref          string  `json:"ref,omitempty"`
	Rows         int64   `json:"rows"`
	Filtered     float64 `json:"filtered_pct"`
	Extra        string  `json:"extra,omitempty"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_explain",
		Description: "Run EXPLAIN on a MySQL query and return the execution plan. Optionally run EXPLAIN ANALYZE to include actual execution metrics (WARNING: EXPLAIN ANALYZE executes the query).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.my.cnf using this as the section name.",
				},
				"database": map[string]any{
					"type":        "string",
					"description": "The database (schema) to use for the query.",
				},
				"query": map[string]any{
					"type":        "string",
					"description": "The SELECT query to explain.",
				},
				"analyze": map[string]any{
					"type":        "boolean",
					"description": "Run EXPLAIN ANALYZE instead of EXPLAIN. This actually executes the query. Default: false.",
				},
			},
			"required": []string{"db_instance_identifier", "database", "query"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			database, _ := args["database"].(string)
			query, _ := args["query"].(string)
			analyze, _ := args["analyze"].(bool)

			db, err := mysqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			if _, err := db.ExecContext(ctx, "USE `"+database+"`"); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("selecting database: %w", err)
			}

			prefix := "EXPLAIN"
			if analyze {
				prefix = "EXPLAIN ANALYZE"
			}

			rows, err := db.QueryContext(ctx, prefix+" "+query)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing explain: %w", err)
			}
			defer rows.Close()

			// EXPLAIN ANALYZE returns a single text column (tree format).
			if analyze {
				var treeOutput string
				if rows.Next() {
					if err := rows.Scan(&treeOutput); err != nil {
						return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning analyze result: %w", err)
					}
				}
				return &mcp.CallToolResult{}, map[string]any{
					"instance": instanceID,
					"database": database,
					"query":    query,
					"analyze":  true,
					"plan":     treeOutput,
				}, nil
			}

			// Standard EXPLAIN returns tabular rows.
			plan := make([]ExplainRow, 0)
			for rows.Next() {
				var r ExplainRow
				var (
					partitions   sql.NullString
					table        sql.NullString
					possibleKeys sql.NullString
					key          sql.NullString
					keyLen       sql.NullString
					ref          sql.NullString
					extra        sql.NullString
				)
				if err := rows.Scan(
					&r.ID, &r.SelectType, &table, &partitions,
					&r.AccessType, &possibleKeys, &key, &keyLen,
					&ref, &r.Rows, &r.Filtered, &extra,
				); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning explain row: %w", err)
				}
				r.Table = table.String
				r.Partitions = partitions.String
				r.PossibleKeys = possibleKeys.String
				r.Key = key.String
				r.KeyLen = keyLen.String
				r.Ref = ref.String
				r.Extra = extra.String
				plan = append(plan, r)
			}
			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading explain rows: %w", err)
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"database": database,
				"query":    query,
				"analyze":  false,
				"plan":     plan,
				"total":    len(plan),
			}, nil
		},
	})
}
