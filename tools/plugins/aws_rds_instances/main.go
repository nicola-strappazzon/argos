package aws_rds_instances

import (
	"context"

	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Endpoint struct {
	Address string `json:"address"`
	Port    int64  `json:"port"`
}

type Instance struct {
	Identifier    string   `json:"identifier"`
	Class         string   `json:"class"`
	Status        string   `json:"status"`
	Engine        string   `json:"engine"`
	EngineVersion string   `json:"engine_version"`
	AZ            string   `json:"availability_zone"`
	MultiAZ       bool     `json:"multi_az"`
	Endpoint      Endpoint `json:"endpoint"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_instances",
		Description: "List AWS RDS Instances.",
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			sess, err := session.NewSession(&aws.Config{
				Region: aws.String("eu-west-1"),
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := rds.New(sess)

			result, err := svc.DescribeDBInstances(nil)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			instances := make([]Instance, 0, len(result.DBInstances))
			for _, db := range result.DBInstances {
				inst := Instance{
					Identifier:    aws.StringValue(db.DBInstanceIdentifier),
					Class:         aws.StringValue(db.DBInstanceClass),
					Status:        aws.StringValue(db.DBInstanceStatus),
					Engine:        aws.StringValue(db.Engine),
					EngineVersion: aws.StringValue(db.EngineVersion),
					AZ:            aws.StringValue(db.AvailabilityZone),
					MultiAZ:       aws.BoolValue(db.MultiAZ),
				}

				if db.Endpoint != nil {
					inst.Endpoint = Endpoint{
						Address: aws.StringValue(db.Endpoint.Address),
						Port:    aws.Int64Value(db.Endpoint.Port),
					}
				}

				instances = append(instances, inst)
			}

			return &mcp.CallToolResult{}, map[string]any{"instances": instances}, nil
		},
	})
}
