package aws_ec2_list

import (
	"context"
	"strings"

	awsconfig "github.com/nicola-strappazzon/argos/internal/config/aws"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Instance struct {
	Name       string `json:"name"`
	InstanceID string `json:"instance_id"`
	PrivateIP  string `json:"private_ip"`
	PublicIP   string `json:"public_ip,omitempty"`
	Zone       string `json:"availability_zone"`
	Type       string `json:"instance_type"`
	State      string `json:"state"`
}

func getTag(tags []*ec2.Tag, key string) string {
	for _, t := range tags {
		if aws.StringValue(t.Key) == key {
			return aws.StringValue(t.Value)
		}
	}
	return ""
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_ec2_list",
		Description: "List AWS EC2 instances. Optionally filter by name tag using a search string.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filter": map[string]any{
					"type":        "string",
					"description": "Optional string to filter instances by Name tag (case-insensitive substring match).",
				},
			},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			filter, _ := args["filter"].(string)

			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			svc := ec2.New(sess)

			input := &ec2.DescribeInstancesInput{}
			if filter != "" {
				input.Filters = []*ec2.Filter{{
					Name:   aws.String("tag:Name"),
					Values: []*string{aws.String("*" + filter + "*")},
				}}
			}

			instances := make([]Instance, 0)
			err = svc.DescribeInstancesPagesWithContext(ctx, input, func(page *ec2.DescribeInstancesOutput, _ bool) bool {
				for _, r := range page.Reservations {
					for _, i := range r.Instances {
						name := getTag(i.Tags, "Name")
						if filter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
							// AWS wildcard filter is case-sensitive on some regions; apply local filter too
							if !strings.Contains(strings.ToLower(aws.StringValue(i.InstanceId)), strings.ToLower(filter)) {
								continue
							}
						}

						inst := Instance{
							Name:       name,
							InstanceID: aws.StringValue(i.InstanceId),
							PrivateIP:  aws.StringValue(i.PrivateIpAddress),
							PublicIP:   aws.StringValue(i.PublicIpAddress),
							Zone:       aws.StringValue(i.Placement.AvailabilityZone),
							Type:       aws.StringValue(i.InstanceType),
						}
						if i.State != nil {
							inst.State = aws.StringValue(i.State.Name)
						}

						instances = append(instances, inst)
					}
				}
				return true
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instances": instances,
				"total":     len(instances),
			}, nil
		},
	})
}
