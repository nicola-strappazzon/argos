package aws_rds_events

import (
	"context"
	"time"

	"github.com/nicola-strappazzon/argos/internal/awsconfig"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Event struct {
	Date             string   `json:"date"`
	Message          string   `json:"message"`
	Categories       []string `json:"categories"`
	SourceIdentifier string   `json:"source_identifier"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_events",
		Description: "List recent RDS events (failovers, maintenance, reboots, storage issues) for an RDS instance.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier to fetch events for.",
				},
				"minutes": map[string]any{
					"type":        "integer",
					"description": "Time window in minutes to look back (default: 1440 = 24 hours).",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)

			minutes := 1440
			if m, ok := args["minutes"].(float64); ok && m > 0 {
				minutes = int(m)
			}

			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := rds.New(sess)
			now := time.Now()
			start := now.Add(-time.Duration(minutes) * time.Minute)

			result, err := svc.DescribeEvents(&rds.DescribeEventsInput{
				SourceIdentifier: aws.String(instanceID),
				SourceType:       aws.String("db-instance"),
				StartTime:        aws.Time(start),
				EndTime:          aws.Time(now),
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			events := make([]Event, 0, len(result.Events))
			for _, e := range result.Events {
				categories := make([]string, 0, len(e.EventCategories))
				for _, c := range e.EventCategories {
					categories = append(categories, aws.StringValue(c))
				}
				events = append(events, Event{
					Date:             aws.TimeValue(e.Date).UTC().Format(time.RFC3339),
					Message:          aws.StringValue(e.Message),
					Categories:       categories,
					SourceIdentifier: aws.StringValue(e.SourceIdentifier),
				})
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"minutes":  minutes,
				"events":   events,
			}, nil
		},
	})
}
