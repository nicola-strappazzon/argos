package pt_variable_advisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const outputDir = "/tmp/argos/pt-variable-advisor"

func init() {
	registry.Add(registry.Property{
		Name:        "pt_variable_advisor",
		Description: "Run pt-variable-advisor against a MySQL/RDS instance and save the report to /tmp/argos/pt-variable-advisor. The host and port can be obtained from aws_rds_instances.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"host": map[string]any{
					"type":        "string",
					"description": "MySQL host or RDS endpoint.",
				},
				"port": map[string]any{
					"type":        "integer",
					"description": "MySQL port (default: 3306).",
				},
				"username": map[string]any{
					"type":        "string",
					"description": "MySQL username.",
				},
				"password": map[string]any{
					"type":        "string",
					"description": "MySQL password.",
				},
			},
			"required": []string{"host", "username", "password"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			host, _ := args["host"].(string)
			username, _ := args["username"].(string)
			password, _ := args["password"].(string)

			port := 3306
			if p, ok := args["port"].(float64); ok && p > 0 {
				port = int(p)
			}

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("creating output directory: %w", err)
			}

			reportName := fmt.Sprintf("%s_%d.txt", strings.ReplaceAll(host, ".", "_"), port)
			reportPath := filepath.Join(outputDir, reportName)

			reportFile, err := os.Create(reportPath)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("creating report file: %w", err)
			}
			defer reportFile.Close()

			dsn := fmt.Sprintf("h=%s,u=%s,p=%s,P=%d", host, username, password, port)
			cmd := exec.CommandContext(ctx, "pt-variable-advisor", dsn)
			cmd.Stdout = reportFile
			var stderr strings.Builder
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("pt-variable-advisor failed: %w — %s", err, stderr.String())
			}

			info, _ := os.Stat(reportPath)
			sizeKB := int64(0)
			if info != nil {
				sizeKB = info.Size() / 1024
			}

			return &mcp.CallToolResult{}, map[string]any{
				"host":        host,
				"port":        port,
				"report_path": reportPath,
				"size_kb":     sizeKB,
			}, nil
		},
	})
}
