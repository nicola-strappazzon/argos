package mysql_process_detail

import (
	"context"
	"database/sql"
	"fmt"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Statement struct {
	SQLText      string  `json:"sql_text"`
	DurationMs   float64 `json:"duration_ms"`
	RowsAffected int64   `json:"rows_affected"`
	RowsSent     int64   `json:"rows_sent"`
	RowsExamined int64   `json:"rows_examined"`
	Errors       int64   `json:"errors"`
}

type ProcessDetail struct {
	ID         int64       `json:"id"`
	User       string      `json:"user"`
	Host       string      `json:"host"`
	DB         string      `json:"db,omitempty"`
	Command    string      `json:"command"`
	TimeSec    int64       `json:"time_sec"`
	State      string      `json:"state,omitempty"`
	Info       string      `json:"info,omitempty"`
	Statements []Statement `json:"statements"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_process_detail",
		Description: "Get full details of a MySQL process by ID, including the complete SQL statement and recent query history from performance_schema.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.my.cnf using this as the section name.",
				},
				"process_id": map[string]any{
					"type":        "integer",
					"description": "The process ID to inspect (from SHOW PROCESSLIST or mysql_processlist).",
				},
			},
			"required": []string{"db_instance_identifier", "process_id"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			processID, _ := args["process_id"].(float64)

			db, err := mysqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			var p ProcessDetail
			var (
				dbName sql.NullString
				state  sql.NullString
				info   sql.NullString
			)

			err = db.QueryRowContext(ctx, `
				SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO
				FROM information_schema.PROCESSLIST
				WHERE ID = ?`, int64(processID)).
				Scan(&p.ID, &p.User, &p.Host, &dbName, &p.Command, &p.TimeSec, &state, &info)
			if err == sql.ErrNoRows {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("process %d not found", int64(processID))
			}
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("querying process: %w", err)
			}

			p.DB = dbName.String
			p.State = state.String
			p.Info = info.String

			rows, err := db.QueryContext(ctx, `
				SELECT
					stmt.sql_text,
					stmt.timer_wait / 1000000000 AS duration_ms,
					stmt.rows_affected,
					stmt.rows_sent,
					stmt.rows_examined,
					stmt.errors
				FROM performance_schema.threads thr
				JOIN performance_schema.events_statements_history stmt
					ON stmt.thread_id = thr.thread_id
				WHERE thr.processlist_id = ?
				ORDER BY stmt.timer_start ASC`, int64(processID))
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("querying statement history: %w", err)
			}
			defer rows.Close()

			p.Statements = make([]Statement, 0)
			for rows.Next() {
				var s Statement
				var sqlText sql.NullString
				if err := rows.Scan(&sqlText, &s.DurationMs, &s.RowsAffected, &s.RowsSent, &s.RowsExamined, &s.Errors); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning statement: %w", err)
				}
				s.SQLText = sqlText.String
				p.Statements = append(p.Statements, s)
			}
			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading statements: %w", err)
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"process":  p,
			}, nil
		},
	})
}
