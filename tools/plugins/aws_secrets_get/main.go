package aws_secrets_get

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nicola-strappazzon/argos/internal/awsconfig"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	registry.Add(registry.Property{
		Name:        "aws_secrets_get",
		Description: "Get the value of an AWS Secrets Manager secret. If the secret is JSON, returns key-value pairs with optional filtering by key name.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The secret name or ARN to retrieve.",
				},
				"filter": map[string]any{
					"type":        "string",
					"description": "Optional partial match to filter keys within the secret (e.g. 'password', 'db_').",
				},
			},
			"required": []string{"name"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			name, _ := args["name"].(string)
			filter, _ := args["filter"].(string)

			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := secretsmanager.New(sess)

			result, err := svc.GetSecretValueWithContext(ctx, &secretsmanager.GetSecretValueInput{
				SecretId: aws.String(name),
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("getting secret %s: %w", name, err)
			}

			secretValue := aws.StringValue(result.SecretString)

			// Try to parse as JSON for structured output.
			var parsed map[string]any
			if err := json.Unmarshal([]byte(secretValue), &parsed); err != nil {
				// Not JSON — return raw string value.
				return &mcp.CallToolResult{}, map[string]any{
					"name":  name,
					"value": secretValue,
				}, nil
			}

			// Apply optional key filter.
			if filter != "" {
				filtered := make(map[string]any)
				lowerFilter := strings.ToLower(filter)
				for k, v := range parsed {
					if strings.Contains(strings.ToLower(k), lowerFilter) {
						filtered[k] = v
					}
				}
				parsed = filtered
			}

			return &mcp.CallToolResult{}, map[string]any{
				"name":   name,
				"values": parsed,
				"total":  len(parsed),
			}, nil
		},
	})
}
