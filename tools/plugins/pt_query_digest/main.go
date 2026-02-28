package pt_query_digest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nicola-strappazzon/mcp/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const outputDir = "/tmp/pt-query-digest"

func init() {
	registry.Add(registry.Property{
		Name:        "pt_query_digest",
		Description: "Run pt-query-digest on a downloaded RDS slow query log file and save the report to /tmp/pt-query-digest.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"log_file_path": map[string]any{
					"type":        "string",
					"description": "Absolute path to the slow query log file to analyze (e.g. /tmp/aws_rds_logs/instance/slowquery/mysql-slowquery.log).",
				},
			},
			"required": []string{"log_file_path"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			logFilePath, _ := args["log_file_path"].(string)

			if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("log file not found: %s", logFilePath)
			}

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("creating output directory: %w", err)
			}

			// Derive report filename from log file path.
			reportName := strings.ReplaceAll(filepath.Base(logFilePath), "/", "_") + ".txt"
			reportPath := filepath.Join(outputDir, reportName)

			reportFile, err := os.Create(reportPath)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("creating report file: %w", err)
			}
			defer reportFile.Close()

			cmd := exec.CommandContext(ctx, "pt-query-digest", logFilePath)
			cmd.Stdout = reportFile
			var stderr strings.Builder
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("pt-query-digest failed: %w — %s", err, stderr.String())
			}

			info, _ := os.Stat(reportPath)
			sizeKB := int64(0)
			if info != nil {
				sizeKB = info.Size() / 1024
			}

			return &mcp.CallToolResult{}, map[string]any{
				"log_file_path": logFilePath,
				"report_path":   reportPath,
				"size_kb":       sizeKB,
			}, nil
		},
	})
}
