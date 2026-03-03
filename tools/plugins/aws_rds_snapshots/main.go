package aws_rds_snapshots

import (
	"context"
	"time"

	"github.com/nicola-strappazzon/argos/internal/config/aws"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Snapshot struct {
	Identifier         string  `json:"identifier"`
	DBInstanceID       string  `json:"db_instance_identifier"`
	Status             string  `json:"status"`
	Type               string  `json:"type"`
	Engine             string  `json:"engine"`
	EngineVersion      string  `json:"engine_version"`
	CreatedAt          string  `json:"created_at"`
	AllocatedStorageGB int64   `json:"allocated_storage_gb"`
	Encrypted          bool    `json:"encrypted"`
	PercentProgress    float64 `json:"percent_progress"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_snapshots",
		Description: "List RDS snapshots (automated and manual) for an instance or all instances.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "Filter snapshots by RDS instance identifier. If omitted, returns snapshots for all instances.",
				},
				"snapshot_type": map[string]any{
					"type":        "string",
					"description": "Filter by snapshot type: automated or manual. If omitted, returns both.",
					"enum":        []string{"automated", "manual"},
				},
			},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			snapshotType, _ := args["snapshot_type"].(string)

			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := rds.New(sess)

			input := &rds.DescribeDBSnapshotsInput{}
			if instanceID != "" {
				input.DBInstanceIdentifier = aws.String(instanceID)
			}
			if snapshotType != "" {
				input.SnapshotType = aws.String(snapshotType)
			}

			result, err := svc.DescribeDBSnapshots(input)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			snapshots := make([]Snapshot, 0, len(result.DBSnapshots))
			for _, s := range result.DBSnapshots {
				createdAt := ""
				if s.SnapshotCreateTime != nil {
					createdAt = s.SnapshotCreateTime.UTC().Format(time.RFC3339)
				}
				snapshots = append(snapshots, Snapshot{
					Identifier:         aws.StringValue(s.DBSnapshotIdentifier),
					DBInstanceID:       aws.StringValue(s.DBInstanceIdentifier),
					Status:             aws.StringValue(s.Status),
					Type:               aws.StringValue(s.SnapshotType),
					Engine:             aws.StringValue(s.Engine),
					EngineVersion:      aws.StringValue(s.EngineVersion),
					CreatedAt:          createdAt,
					AllocatedStorageGB: int64(aws.Int64Value(s.AllocatedStorage)),
					Encrypted:          aws.BoolValue(s.Encrypted),
					PercentProgress:    float64(aws.Int64Value(s.PercentProgress)),
				})
			}

			return &mcp.CallToolResult{}, map[string]any{
				"snapshots": snapshots,
				"total":     len(snapshots),
			}, nil
		},
	})
}
