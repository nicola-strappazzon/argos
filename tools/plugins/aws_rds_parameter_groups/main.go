package aws_rds_parameter_groups

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/argos/internal/awsconfig"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Parameter struct {
	Name         string `json:"name"`
	Value        string `json:"value"`
	Source       string `json:"source"`
	DataType     string `json:"data_type"`
	ApplyType    string `json:"apply_type"`
	IsModifiable bool   `json:"is_modifiable"`
	Description  string `json:"description"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_parameter_groups",
		Description: "List all parameters of the parameter group associated with a given RDS DB instance.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier.",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)

			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := rds.New(sess)

			dbOutput, err := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(instanceID),
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			if len(dbOutput.DBInstances) == 0 {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("instance %q not found", instanceID)
			}

			dbInstance := dbOutput.DBInstances[0]
			if len(dbInstance.DBParameterGroups) == 0 {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("no parameter groups found for instance %q", instanceID)
			}

			pgName := aws.StringValue(dbInstance.DBParameterGroups[0].DBParameterGroupName)

			var parameters []Parameter

			err = svc.DescribeDBParametersPages(
				&rds.DescribeDBParametersInput{
					DBParameterGroupName: aws.String(pgName),
					Source:               aws.String("user"),
				},
				func(page *rds.DescribeDBParametersOutput, lastPage bool) bool {
					for _, p := range page.Parameters {
						parameters = append(parameters, Parameter{
							Name:         aws.StringValue(p.ParameterName),
							Value:        aws.StringValue(p.ParameterValue),
							Source:       aws.StringValue(p.Source),
							DataType:     aws.StringValue(p.DataType),
							ApplyType:    aws.StringValue(p.ApplyType),
							IsModifiable: aws.BoolValue(p.IsModifiable),
							Description:  aws.StringValue(p.Description),
						})
					}
					return true
				},
			)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			return &mcp.CallToolResult{}, map[string]any{
				"identifier":      instanceID,
				"parameter_group": pgName,
				"count":           len(parameters),
				"parameters":      parameters,
			}, nil
		},
	})
}
