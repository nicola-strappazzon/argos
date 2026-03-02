package mysql_table_indexes

import (
	"context"
	"fmt"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type IndexColumn struct {
	Position int    `json:"position"`
	Column   string `json:"column"`
	SubPart  int64  `json:"sub_part,omitempty"`
	Nullable bool   `json:"nullable"`
}

type Index struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Unique      bool          `json:"unique"`
	Visible     bool          `json:"visible"`
	Cardinality int64         `json:"cardinality"`
	Columns     []IndexColumn `json:"columns"`
	SizeMB      float64       `json:"size_mb"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_table_indexes",
		Description: "List indexes of a MySQL table with type, uniqueness, visibility, cardinality, columns and size per index.",
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
					"description": "The table name to inspect indexes for.",
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

			// Fetch index columns from information_schema.statistics.
			colQuery := `
				SELECT
					index_name,
					index_type,
					non_unique,
					is_visible,
					COALESCE(cardinality, 0),
					seq_in_index,
					column_name,
					COALESCE(sub_part, 0),
					nullable
				FROM information_schema.statistics
				WHERE table_schema = ?
					AND table_name = ?
				ORDER BY index_name, seq_in_index`

			rows, err := db.QueryContext(ctx, colQuery, database, table)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("querying indexes: %w", err)
			}
			defer rows.Close()

			indexMap := make(map[string]*Index)
			indexOrder := make([]string, 0)

			for rows.Next() {
				var (
					indexName, indexType, isVisible, nullable string
					nonUnique, seqInIndex                     int
					cardinality, subPart                      int64
					columnName                                string
				)
				if err := rows.Scan(
					&indexName, &indexType, &nonUnique, &isVisible,
					&cardinality, &seqInIndex, &columnName, &subPart, &nullable,
				); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning index row: %w", err)
				}

				if _, exists := indexMap[indexName]; !exists {
					indexMap[indexName] = &Index{
						Name:        indexName,
						Type:        indexType,
						Unique:      nonUnique == 0,
						Visible:     isVisible == "YES",
						Cardinality: cardinality,
						Columns:     make([]IndexColumn, 0),
					}
					indexOrder = append(indexOrder, indexName)
				}

				indexMap[indexName].Columns = append(indexMap[indexName].Columns, IndexColumn{
					Position: seqInIndex,
					Column:   columnName,
					SubPart:  subPart,
					Nullable: nullable == "YES",
				})
			}

			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading index rows: %w", err)
			}

			// Fetch index sizes from mysql.innodb_index_stats.
			sizeQuery := `
				SELECT
					index_name,
					ROUND(stat_value * @@innodb_page_size / 1024 / 1024, 2) AS size_mb
				FROM mysql.innodb_index_stats
				WHERE database_name = ?
					AND table_name = ?
					AND stat_name = 'size'`

			sizeRows, err := db.QueryContext(ctx, sizeQuery, database, table)
			if err == nil {
				defer sizeRows.Close()
				for sizeRows.Next() {
					var name string
					var sizeMB float64
					if err := sizeRows.Scan(&name, &sizeMB); err == nil {
						if idx, exists := indexMap[name]; exists {
							idx.SizeMB = sizeMB
						}
					}
				}
			}

			// Build ordered result.
			indexes := make([]Index, 0, len(indexOrder))
			for _, name := range indexOrder {
				indexes = append(indexes, *indexMap[name])
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"database": database,
				"table":    table,
				"indexes":  indexes,
				"total":    len(indexes),
			}, nil
		},
	})
}
