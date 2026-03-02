package aws_secrets_list

import (
	"context"
	"time"

	"github.com/nicola-strappazzon/argos/internal/awsconfig"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Secret struct {
	Name            string `json:"name"`
	ARN             string `json:"arn"`
	Description     string `json:"description,omitempty"`
	LastChanged     string `json:"last_changed,omitempty"`
	LastAccessed    string `json:"last_accessed,omitempty"`
	RotationEnabled bool   `json:"rotation_enabled"`
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_secrets_list",
		Description: "List AWS Secrets Manager secrets. Optionally filter by name using a search string.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filter": map[string]any{
					"type":        "string",
					"description": "Optional string to filter secrets by name.",
				},
			},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			filter, _ := args["filter"].(string)

			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := secretsmanager.New(sess)

			input := &secretsmanager.ListSecretsInput{}
			if filter != "" {
				input.Filters = []*secretsmanager.Filter{{
					Key:    aws.String("name"),
					Values: []*string{aws.String(filter)},
				}}
			}

			secrets := make([]Secret, 0)
			err = svc.ListSecretsPagesWithContext(ctx, input, func(page *secretsmanager.ListSecretsOutput, _ bool) bool {
				for _, s := range page.SecretList {
					secrets = append(secrets, Secret{
						Name:            aws.StringValue(s.Name),
						ARN:             aws.StringValue(s.ARN),
						Description:     aws.StringValue(s.Description),
						LastChanged:     formatTime(s.LastChangedDate),
						LastAccessed:    formatTime(s.LastAccessedDate),
						RotationEnabled: aws.BoolValue(s.RotationEnabled),
					})
				}
				return true
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			return &mcp.CallToolResult{}, map[string]any{
				"secrets": secrets,
				"total":   len(secrets),
			}, nil
		},
	})
}
