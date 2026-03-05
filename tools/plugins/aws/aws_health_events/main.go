package aws_health_events

import (
	"context"
	"time"

	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/health"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The AWS Health API endpoint is global and only available in us-east-1.
func newHealthSession() (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
}

type HealthEvent struct {
	ARN          string `json:"arn"`
	Service      string `json:"service"`
	TypeCode     string `json:"type_code"`
	TypeCategory string `json:"type_category"`
	Region       string `json:"region"`
	Status       string `json:"status"`
	StartTime    string `json:"start_time,omitempty"`
	EndTime      string `json:"end_time,omitempty"`
	LastUpdated  string `json:"last_updated,omitempty"`
	Description  string `json:"description,omitempty"`
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_health_events",
		Description: "List AWS Health events (end-of-support notices, deprecations, service incidents) from the Personal Health Dashboard.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"service": map[string]any{
					"type":        "string",
					"description": "Filter by AWS service (e.g. RDS, EC2). If omitted, returns events for all services.",
				},
				"status": map[string]any{
					"type":        "string",
					"description": "Filter by event status: open, closed, or upcoming. If omitted, returns all.",
					"enum":        []string{"open", "closed", "upcoming"},
				},
			},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			service, _ := args["service"].(string)
			status, _ := args["status"].(string)

			sess, err := newHealthSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := health.New(sess)

			filter := &health.EventFilter{}
			if service != "" {
				filter.Services = []*string{aws.String(service)}
			}
			if status != "" {
				filter.EventStatusCodes = []*string{aws.String(status)}
			}

			result, err := svc.DescribeEvents(&health.DescribeEventsInput{
				Filter: filter,
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			if len(result.Events) == 0 {
				return &mcp.CallToolResult{}, map[string]any{
					"events": []HealthEvent{},
					"total":  0,
				}, nil
			}

			// Fetch full descriptions for all events.
			arns := make([]*string, 0, len(result.Events))
			for _, e := range result.Events {
				arns = append(arns, e.Arn)
			}

			details, err := svc.DescribeEventDetails(&health.DescribeEventDetailsInput{
				EventArns: arns,
			})

			descByARN := map[string]string{}
			if err == nil {
				for _, d := range details.SuccessfulSet {
					if d.EventDescription != nil {
						descByARN[aws.StringValue(d.Event.Arn)] = aws.StringValue(d.EventDescription.LatestDescription)
					}
				}
			}

			events := make([]HealthEvent, 0, len(result.Events))
			for _, e := range result.Events {
				arn := aws.StringValue(e.Arn)
				events = append(events, HealthEvent{
					ARN:          arn,
					Service:      aws.StringValue(e.Service),
					TypeCode:     aws.StringValue(e.EventTypeCode),
					TypeCategory: aws.StringValue(e.EventTypeCategory),
					Region:       aws.StringValue(e.Region),
					Status:       aws.StringValue(e.StatusCode),
					StartTime:    formatTime(e.StartTime),
					EndTime:      formatTime(e.EndTime),
					LastUpdated:  formatTime(e.LastUpdatedTime),
					Description:  descByARN[arn],
				})
			}

			return &mcp.CallToolResult{}, map[string]any{
				"events": events,
				"total":  len(events),
			}, nil
		},
	})
}
