package aws_rds_performance_insights

import (
	"context"
	"fmt"
	"time"

	"github.com/nicola-strappazzon/argos/internal/awsconfig"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/pi"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TopQuery struct {
	Statement string  `json:"statement"`
	Load      float64 `json:"db_load_avg"`
}

type WaitEvent struct {
	Type string  `json:"type"`
	Name string  `json:"name"`
	Load float64 `json:"db_load_avg"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_performance_insights",
		Description: "Get top SQL queries and wait events by DB load from Performance Insights for a given RDS instance.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier.",
				},
				"minutes": map[string]any{
					"type":        "integer",
					"description": "Time window in minutes to analyze (default: 60).",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			minutes := 60
			if m, ok := args["minutes"].(float64); ok && m > 0 {
				minutes = int(m)
			}

			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			rdsSvc := rds.New(sess)
			dbOutput, err := rdsSvc.DescribeDBInstances(&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(instanceID),
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			if len(dbOutput.DBInstances) == 0 {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("instance %q not found", instanceID)
			}

			db := dbOutput.DBInstances[0]

			if !aws.BoolValue(db.PerformanceInsightsEnabled) {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("Performance Insights is not enabled for instance %q", instanceID)
			}

			dbiResourceID := aws.StringValue(db.DbiResourceId)
			serviceType := "RDS"
			if aws.StringValue(db.Engine) == "docdb" {
				serviceType = "DOCDB"
			}

			piSvc := pi.New(sess)
			endTime := time.Now()
			startTime := endTime.Add(-time.Duration(minutes) * time.Minute)

			// Top queries by DB load
			sqlOutput, err := piSvc.DescribeDimensionKeys(&pi.DescribeDimensionKeysInput{
				ServiceType: aws.String(serviceType),
				Identifier:  aws.String(dbiResourceID),
				StartTime:   &startTime,
				EndTime:     &endTime,
				Metric:      aws.String("db.load.avg"),
				GroupBy: &pi.DimensionGroup{
					Group: aws.String("db.sql_tokenized"),
					Dimensions: []*string{
						aws.String("db.sql_tokenized.statement"),
					},
					Limit: aws.Int64(10),
				},
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			var topQueries []TopQuery
			for _, key := range sqlOutput.Keys {
				stmt := ""
				if v, ok := key.Dimensions["db.sql_tokenized.statement"]; ok {
					stmt = aws.StringValue(v)
				}
				topQueries = append(topQueries, TopQuery{
					Statement: stmt,
					Load:      aws.Float64Value(key.Total),
				})
			}

			// Top wait events by DB load
			waitOutput, err := piSvc.DescribeDimensionKeys(&pi.DescribeDimensionKeysInput{
				ServiceType: aws.String(serviceType),
				Identifier:  aws.String(dbiResourceID),
				StartTime:   &startTime,
				EndTime:     &endTime,
				Metric:      aws.String("db.load.avg"),
				GroupBy: &pi.DimensionGroup{
					Group: aws.String("db.wait_event"),
					Dimensions: []*string{
						aws.String("db.wait_event.type"),
						aws.String("db.wait_event.name"),
					},
					Limit: aws.Int64(10),
				},
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			var waitEvents []WaitEvent
			for _, key := range waitOutput.Keys {
				eventType := ""
				eventName := ""
				if v, ok := key.Dimensions["db.wait_event.type"]; ok {
					eventType = aws.StringValue(v)
				}
				if v, ok := key.Dimensions["db.wait_event.name"]; ok {
					eventName = aws.StringValue(v)
				}
				waitEvents = append(waitEvents, WaitEvent{
					Type: eventType,
					Name: eventName,
					Load: aws.Float64Value(key.Total),
				})
			}

			return &mcp.CallToolResult{}, map[string]any{
				"identifier":  instanceID,
				"period_min":  minutes,
				"top_queries": topQueries,
				"wait_events": waitEvents,
			}, nil
		},
	})
}
