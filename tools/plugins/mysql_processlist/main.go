package mysql_processlist

import (
	"context"
	"database/sql"
	"fmt"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Process struct {
	ID      int64  `json:"id"`
	User    string `json:"user"`
	Host    string `json:"host"`
	DB      string `json:"db,omitempty"`
	Command string `json:"command"`
	Time    int64  `json:"time_sec"`
	State   string `json:"state,omitempty"`
	Info    string `json:"info,omitempty"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_processlist",
		Description: "Run SHOW FULL PROCESSLIST on a MySQL instance. Idle connections (Command=Sleep) and processes without an active SQL statement are excluded by default.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.my.cnf using this as the section name.",
				},
				"include_idle": map[string]any{
					"type":        "boolean",
					"description": "Include idle connections (Command=Sleep). Default: false.",
				},
				"min_time_sec": map[string]any{
					"type":        "integer",
					"description": "Only include processes running for at least this many seconds. Default: 0 (no filter).",
				},
				"include_no_statement": map[string]any{
					"type":        "boolean",
					"description": "Include processes without an active SQL statement (INFO is null). Default: false.",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			includeIdle, _ := args["include_idle"].(bool)
			minTimeSec, _ := args["min_time_sec"].(float64)
			includeNoStatement, _ := args["include_no_statement"].(bool)

			db, err := mysqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			rows, err := db.QueryContext(ctx, `
				SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO
				FROM information_schema.PROCESSLIST
				WHERE ID <> CONNECTION_ID()`)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			processes := make([]Process, 0)
			totalIdle := 0

			for rows.Next() {
				var p Process
				var (
					db    sql.NullString
					state sql.NullString
					info  sql.NullString
				)
				if err := rows.Scan(&p.ID, &p.User, &p.Host, &db, &p.Command, &p.Time, &state, &info); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning row: %w", err)
				}
				p.DB = db.String
				p.State = state.String
				if len(info.String) > 500 {
					p.Info = info.String[:500]
				} else {
					p.Info = info.String
				}

				if p.Command == "Sleep" {
					totalIdle++
					if !includeIdle {
						continue
					}
				}
				if minTimeSec > 0 && p.Time < int64(minTimeSec) {
					continue
				}
				if !includeNoStatement && p.Info == "" {
					continue
				}
				processes = append(processes, p)
			}
			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading rows: %w", err)
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance":     instanceID,
				"processes":    processes,
				"total":        len(processes),
				"total_idle":   totalIdle,
				"include_idle": includeIdle,
			}, nil
		},
	})
}
