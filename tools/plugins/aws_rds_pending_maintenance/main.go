package aws_rds_pending_maintenance

import (
	"context"
	"time"

	"github.com/nicola-strappazzon/argos/internal/awsconfig"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PendingAction struct {
	ResourceIdentifier string `json:"resource_identifier"`
	Action             string `json:"action"`
	AutoAppliedAfter   string `json:"auto_applied_after,omitempty"`
	CurrentApplyDate   string `json:"current_apply_date,omitempty"`
	Description        string `json:"description"`
	OptInStatus        string `json:"opt_in_status"`
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_pending_maintenance",
		Description: "List pending maintenance actions for RDS instances (engine upgrades, OS patches, security updates).",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := rds.New(sess)

			result, err := svc.DescribePendingMaintenanceActions(&rds.DescribePendingMaintenanceActionsInput{})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			actions := make([]PendingAction, 0)
			for _, resource := range result.PendingMaintenanceActions {
				for _, a := range resource.PendingMaintenanceActionDetails {
					actions = append(actions, PendingAction{
						ResourceIdentifier: aws.StringValue(resource.ResourceIdentifier),
						Action:             aws.StringValue(a.Action),
						AutoAppliedAfter:   formatTime(a.AutoAppliedAfterDate),
						CurrentApplyDate:   formatTime(a.CurrentApplyDate),
						Description:        aws.StringValue(a.Description),
						OptInStatus:        aws.StringValue(a.OptInStatus),
					})
				}
			}

			return &mcp.CallToolResult{}, map[string]any{
				"pending_actions": actions,
				"total":           len(actions),
			}, nil
		},
	})
}
