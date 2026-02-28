package aws_rds_logs

import (
	"context"
	"time"

	"github.com/nicola-strappazzon/mcp/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type LogFile struct {
	Name        string `json:"name"`
	SizeKB      int64  `json:"size_kb"`
	LastWritten string `json:"last_written"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_logs",
		Description: "List available log files for an RDS instance.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier to list logs for.",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)

			sess, err := session.NewSession(&aws.Config{
				Region: aws.String("eu-west-1"),
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := rds.New(sess)

			var logs []LogFile

			err = svc.DescribeDBLogFilesPages(
				&rds.DescribeDBLogFilesInput{
					DBInstanceIdentifier: aws.String(instanceID),
				},
				func(page *rds.DescribeDBLogFilesOutput, lastPage bool) bool {
					for _, f := range page.DescribeDBLogFiles {
						lastWritten := ""
						if f.LastWritten != nil {
							lastWritten = time.UnixMilli(aws.Int64Value(f.LastWritten)).UTC().Format(time.RFC3339)
						}
						logs = append(logs, LogFile{
							Name:        aws.StringValue(f.LogFileName),
							SizeKB:      aws.Int64Value(f.Size) / 1024,
							LastWritten: lastWritten,
						})
					}
					return true
				},
			)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			return &mcp.CallToolResult{}, map[string]any{
				"identifier": instanceID,
				"count":      len(logs),
				"logs":       logs,
			}, nil
		},
	})
}
