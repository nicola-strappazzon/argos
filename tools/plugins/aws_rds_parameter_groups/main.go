package aws_rds_parameter_groups

import (
	"context"

	"github.com/nicola-strappazzon/argos/internal/awsconfig"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ParameterGroup struct {
	Name        string `json:"name"`
	Family      string `json:"family"`
	Description string `json:"description"`
	ARN         string `json:"arn"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_parameter_groups",
		Description: "List all RDS DB parameter groups.",
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := rds.New(sess)

			var groups []ParameterGroup

			err = svc.DescribeDBParameterGroupsPages(
				&rds.DescribeDBParameterGroupsInput{},
				func(page *rds.DescribeDBParameterGroupsOutput, lastPage bool) bool {
					for _, g := range page.DBParameterGroups {
						groups = append(groups, ParameterGroup{
							Name:        aws.StringValue(g.DBParameterGroupName),
							Family:      aws.StringValue(g.DBParameterGroupFamily),
							Description: aws.StringValue(g.Description),
							ARN:         aws.StringValue(g.DBParameterGroupArn),
						})
					}
					return true
				},
			)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			return &mcp.CallToolResult{}, map[string]any{
				"count":            len(groups),
				"parameter_groups": groups,
			}, nil
		},
	})
}
