package aws_rds_log_download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const baseDir = "/tmp/aws_rds_logs"

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_log_download",
		Description: "Download the content of a specific RDS log file.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier.",
				},
				"log_file_name": map[string]any{
					"type":        "string",
					"description": "The name of the log file to download (e.g. slowquery/mysql-slowquery.log).",
				},
			},
			"required": []string{"db_instance_identifier", "log_file_name"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			logFileName, _ := args["log_file_name"].(string)

			sess, err := session.NewSession(&aws.Config{
				Region: aws.String("eu-west-1"),
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := rds.New(sess)

			var sb strings.Builder

			err = svc.DownloadDBLogFilePortionPages(
				&rds.DownloadDBLogFilePortionInput{
					DBInstanceIdentifier: aws.String(instanceID),
					LogFileName:          aws.String(logFileName),
					NumberOfLines:        aws.Int64(2000),
				},
				func(page *rds.DownloadDBLogFilePortionOutput, lastPage bool) bool {
					sb.WriteString(aws.StringValue(page.LogFileData))
					return !lastPage
				},
			)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			// Build output path: /tmp/aws_rds_logs/<instance>/<log_file_name>
			outputPath := filepath.Join(baseDir, instanceID, logFileName)
			if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("creating output directory: %w", err)
			}

			if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("writing log file: %w", err)
			}

			return &mcp.CallToolResult{}, map[string]any{
				"identifier":    instanceID,
				"log_file_name": logFileName,
				"output_path":   outputPath,
				"size_kb":       sb.Len() / 1024,
			}, nil
		},
	})
}
