package pt_index_usage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nicola-strappazzon/argos/internal/config/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const outputDir = "/tmp/argos/pt-index-usage"

func init() {
	registry.Add(registry.Property{
		Name:        "pt_index_usage",
		Description: "Run pt-index-usage on a downloaded slow query log file to find unused indexes. Saves the report to /tmp/argos/pt-index-usage/. The slow query log can be obtained with aws_rds_log_download.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.my.cnf using this as the section name.",
				},
				"log_file_path": map[string]any{
					"type":        "string",
					"description": "Absolute path to the slow query log file to analyze (e.g. /tmp/argos/aws_rds_logs/instance/slowquery/mysql-slowquery.log).",
				},
				"database": map[string]any{
					"type":        "string",
					"description": "Restrict analysis to this database (schema) name. Optional but recommended to reduce noise.",
				},
			},
			"required": []string{"db_instance_identifier", "log_file_path"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			logFilePath, _ := args["log_file_path"].(string)
			database, _ := args["database"].(string)

			if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("log file not found: %s", logFilePath)
			}

			creds, err := mysqlconfig.Load(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("creating output directory: %w", err)
			}

			reportName := fmt.Sprintf("%s.txt", strings.ReplaceAll(instanceID, "-", "_"))
			reportPath := filepath.Join(outputDir, reportName)

			reportFile, err := os.Create(reportPath)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("creating report file: %w", err)
			}
			defer reportFile.Close()

			cmdArgs := []string{
				fmt.Sprintf("--host=%s", creds.Host),
				fmt.Sprintf("--port=%d", creds.Port),
				fmt.Sprintf("--user=%s", creds.User),
				fmt.Sprintf("--password=%s", creds.Password),
			}
			if database != "" {
				cmdArgs = append(cmdArgs, fmt.Sprintf("--databases=%s", database))
			}
			cmdArgs = append(cmdArgs, logFilePath)

			cmd := exec.CommandContext(ctx, "pt-index-usage", cmdArgs...)
			cmd.Stdout = reportFile
			var stderr strings.Builder
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("pt-index-usage failed: %w — %s", err, stderr.String())
			}

			info, _ := os.Stat(reportPath)
			sizeKB := int64(0)
			if info != nil {
				sizeKB = info.Size() / 1024
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance":      instanceID,
				"log_file_path": logFilePath,
				"database":      database,
				"report_path":   reportPath,
				"size_kb":       sizeKB,
			}, nil
		},
	})
}
