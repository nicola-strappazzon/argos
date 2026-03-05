package pt_variable_advisor

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

const outputDir = "/tmp/argos/pt-variable-advisor"

func init() {
	registry.Add(registry.Property{
		Name:        "pt_variable_advisor",
		Description: "Run pt-variable-advisor against a MySQL/RDS instance and save the report to /tmp/argos/pt-variable-advisor. The host and port can be obtained from aws_rds_instances.",
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

			dsn := fmt.Sprintf("h=%s,u=%s,p=%s,P=%d", creds.Host, creds.User, creds.Password, creds.Port)
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
				"instance":    instanceID,
				"report_path": reportPath,
				"size_kb":     sizeKB,
			}, nil
		},
	})
}
