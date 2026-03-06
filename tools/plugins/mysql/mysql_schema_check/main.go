package mysql_schema_check

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Check struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
	Tables      []string `json:"tables,omitempty"`
}

func checkDeprecatedEngine(ctx context.Context, db *sql.DB) (*Check, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT TABLE_SCHEMA, TABLE_NAME
		FROM information_schema.TABLES
		WHERE ENGINE = 'MyISAM'
		  AND TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
		ORDER BY TABLE_SCHEMA, TABLE_NAME`)
	if err != nil {
		return nil, fmt.Errorf("deprecated engine check: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var schema, table string
		if err := rows.Scan(&schema, &table); err != nil {
			return nil, fmt.Errorf("deprecated engine check: %w", err)
		}
		tables = append(tables, schema+"."+table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("deprecated engine check: %w", err)
	}

	if len(tables) > 0 {
		return &Check{
			Name:   "deprecated_table_engine",
			Status: "warning",
			Description: fmt.Sprintf(
				"%d table(s) are using the MyISAM storage engine: %s. "+
					"MyISAM is deprecated and does not support transactions, crash recovery, or row-level locking. "+
					"InnoDB is the recommended engine: it is fully ACID-compliant, supports foreign keys and full-text search, "+
					"recovers automatically from crashes, and provides row-level locking for better concurrency. "+
					"Migrate with: ALTER TABLE <table_name> ENGINE=InnoDB;",
				len(tables), strings.Join(tables, ", ")),
			Tables: tables,
		}, nil
	}

	return &Check{
		Name:        "deprecated_table_engine",
		Status:      "ok",
		Description: "No user tables are using the MyISAM engine. All tables use InnoDB or another supported engine.",
	}, nil
}

func init() {
	registry.Add(registry.Property{
		Name: "mysql_schema_check",
		Description: `Run schema-level checks on a MySQL instance. Returns structural issues with status (ok/warning) and remediation guidance.

Checks include:
- Deprecated Table Engine: detects tables still using MyISAM. MyISAM lacks transactions, crash recovery, and row-level locking. InnoDB is the recommended engine due to ACID compliance, automatic crash recovery, row-level locking for high concurrency, and active development support from the MySQL community.`,
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

			type checkFn func() (*Check, error)
			runners := []checkFn{
				func() (*Check, error) { return checkDeprecatedEngine(ctx, db) },
			}

			var checks []Check
			for _, run := range runners {
				c, err := run()
				if err != nil {
					return &mcp.CallToolResult{}, nil, err
				}
				if c != nil {
					checks = append(checks, *c)
				}
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"checks":   checks,
				"total":    len(checks),
			}, nil
		},
	})
}
