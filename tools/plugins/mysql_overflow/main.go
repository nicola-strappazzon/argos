package mysql_overflow

import (
	"context"
	"fmt"
	"strings"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type OverflowColumn struct {
	Table          string  `json:"table"`
	Column         string  `json:"column"`
	ColumnType     string  `json:"column_type"`
	Unsigned       bool    `json:"unsigned"`
	CurrentValue   uint64  `json:"current_value"`
	MaxValue       uint64  `json:"max_value"`
	PctUsed        float64 `json:"pct_used"`
	RemainingValue uint64  `json:"remaining_value"`
}

// maxValueForType returns the maximum value for a given MySQL integer column type.
// Signed ranges: TINYINT [-128,127], SMALLINT [-32768,32767], MEDIUMINT [-8388608,8388607],
// INT [-2147483648,2147483647], BIGINT [-9223372036854775808,9223372036854775807].
// For AUTO_INCREMENT purposes the effective min is 1, so max is the positive bound.
func maxValueForType(columnType string) uint64 {
	t := strings.ToLower(columnType)
	unsigned := strings.Contains(t, "unsigned")

	switch {
	case strings.Contains(t, "tinyint"):
		if unsigned {
			return 255
		}
		return 127
	case strings.Contains(t, "smallint"):
		if unsigned {
			return 65535
		}
		return 32767
	case strings.Contains(t, "mediumint"):
		if unsigned {
			return 16777215
		}
		return 8388607
	case strings.Contains(t, "bigint"):
		if unsigned {
			return 18446744073709551615 // 2^64 - 1
		}
		return 9223372036854775807 // 2^63 - 1
	default: // INT / INTEGER
		if unsigned {
			return 4294967295
		}
		return 2147483647
	}
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_overflow",
		Description: "Check AUTO_INCREMENT overflow risk for all tables in a MySQL database. Returns current value, max value, percentage used, and remaining capacity per column, sorted by percentage used descending.",
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
					c.table_name,
					c.column_name,
					c.column_type,
					COALESCE(t.auto_increment, 0)
				FROM information_schema.columns c
				INNER JOIN information_schema.tables t
					ON t.table_schema = c.table_schema
					AND t.table_name = c.table_name
				WHERE c.table_schema = ?
					AND c.extra LIKE '%auto_increment%'
					AND t.table_type = 'BASE TABLE'
				ORDER BY c.table_name, c.column_name`

			rows, err := db.QueryContext(ctx, query, database)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			columns := make([]OverflowColumn, 0)
			for rows.Next() {
				var col OverflowColumn
				if err := rows.Scan(&col.Table, &col.Column, &col.ColumnType, &col.CurrentValue); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning row: %w", err)
				}
				col.Unsigned = strings.Contains(strings.ToLower(col.ColumnType), "unsigned")
				col.MaxValue = maxValueForType(col.ColumnType)
				if col.MaxValue > 0 {
					col.PctUsed = float64(col.CurrentValue) / float64(col.MaxValue) * 100
				}
				if col.MaxValue >= col.CurrentValue {
					col.RemainingValue = col.MaxValue - col.CurrentValue
				}
				columns = append(columns, col)
			}

			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading rows: %w", err)
			}

			// Sort by pct_used descending.
			for i := 1; i < len(columns); i++ {
				for j := i; j > 0 && columns[j].PctUsed > columns[j-1].PctUsed; j-- {
					columns[j], columns[j-1] = columns[j-1], columns[j]
				}
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"database": database,
				"columns":  columns,
				"total":    len(columns),
			}, nil
		},
	})
}
