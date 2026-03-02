package aws_rds_read_replicas

import (
	"context"
	"time"

	"github.com/nicola-strappazzon/argos/internal/awsconfig"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Replica struct {
	Identifier   string  `json:"identifier"`
	Source       string  `json:"source"`
	Status       string  `json:"status"`
	Class        string  `json:"class"`
	AZ           string  `json:"availability_zone"`
	MultiAZ      bool    `json:"multi_az"`
	Engine       string  `json:"engine"`
	EngineVersion string `json:"engine_version"`
	ReplicaLagS  float64 `json:"replica_lag_seconds"`
}

func replicaLag(cwSvc *cloudwatch.CloudWatch, instanceID string) float64 {
	now := time.Now()
	result, err := cwSvc.GetMetricData(&cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(now.Add(-5 * time.Minute)),
		EndTime:   aws.Time(now),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{{
			Id: aws.String("lag"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String("AWS/RDS"),
					MetricName: aws.String("ReplicaLag"),
					Dimensions: []*cloudwatch.Dimension{{
						Name:  aws.String("DBInstanceIdentifier"),
						Value: aws.String(instanceID),
					}},
				},
				Period: aws.Int64(60),
				Stat:   aws.String("Average"),
			},
		}},
	})
	if err != nil || len(result.MetricDataResults) == 0 || len(result.MetricDataResults[0].Values) == 0 {
		return -1
	}
	return aws.Float64Value(result.MetricDataResults[0].Values[0])
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_read_replicas",
		Description: "List RDS read replicas and their replication lag in seconds.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "Filter by source instance identifier. If omitted, returns all read replicas.",
				},
			},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			filterSource, _ := args["db_instance_identifier"].(string)

			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			rdsSvc := rds.New(sess)
			cwSvc := cloudwatch.New(sess)

			result, err := rdsSvc.DescribeDBInstances(nil)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			replicas := make([]Replica, 0)
			for _, db := range result.DBInstances {
				source := aws.StringValue(db.ReadReplicaSourceDBInstanceIdentifier)
				if source == "" {
					continue
				}
				if filterSource != "" && source != filterSource {
					continue
				}

				id := aws.StringValue(db.DBInstanceIdentifier)
				replicas = append(replicas, Replica{
					Identifier:    id,
					Source:        source,
					Status:        aws.StringValue(db.DBInstanceStatus),
					Class:         aws.StringValue(db.DBInstanceClass),
					AZ:            aws.StringValue(db.AvailabilityZone),
					MultiAZ:       aws.BoolValue(db.MultiAZ),
					Engine:        aws.StringValue(db.Engine),
					EngineVersion: aws.StringValue(db.EngineVersion),
					ReplicaLagS:   replicaLag(cwSvc, id),
				})
			}

			return &mcp.CallToolResult{}, map[string]any{
				"replicas": replicas,
				"total":    len(replicas),
			}, nil
		},
	})
}
