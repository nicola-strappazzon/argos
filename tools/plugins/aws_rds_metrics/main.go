package aws_rds_metrics

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

type Metrics struct {
	Identifier       string  `json:"identifier"`
	Engine           string  `json:"engine"`
	CPUPercent       float64 `json:"cpu_percent"`
	Connections      float64 `json:"connections"`
	FreeableMemoryMB float64 `json:"freeable_memory_mb"`
	FreeStorageGB    float64 `json:"free_storage_gb"`
	ReadIOPS         float64 `json:"read_iops"`
	WriteIOPS        float64 `json:"write_iops"`
	ReadLatencyMS    float64 `json:"read_latency_ms"`
	WriteLatencyMS   float64 `json:"write_latency_ms"`
	NetworkRxMBps    float64 `json:"network_rx_mbps"`
	NetworkTxMBps    float64 `json:"network_tx_mbps"`
}

func latestValue(results []*cloudwatch.MetricDataResult, id string, multiplier float64) float64 {
	for _, r := range results {
		if aws.StringValue(r.Id) == id && len(r.Values) > 0 {
			return aws.Float64Value(r.Values[0]) * multiplier
		}
	}
	return 0
}

func query(id, namespace, metricName, instanceID string) *cloudwatch.MetricDataQuery {
	return &cloudwatch.MetricDataQuery{
		Id: aws.String(id),
		MetricStat: &cloudwatch.MetricStat{
			Metric: &cloudwatch.Metric{
				Namespace:  aws.String(namespace),
				MetricName: aws.String(metricName),
				Dimensions: []*cloudwatch.Dimension{{
					Name:  aws.String("DBInstanceIdentifier"),
					Value: aws.String(instanceID),
				}},
			},
			Period: aws.Int64(300),
			Stat:   aws.String("Average"),
		},
	}
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_rds_metrics",
		Description: "Get CloudWatch metrics (CPU, connections, memory, storage, IOPS, latency, network) for an RDS instance.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier to fetch metrics for.",
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

			// Detect engine to pick the right CloudWatch namespace.
			rdsSvc := rds.New(sess)
			dbInfo, err := rdsSvc.DescribeDBInstances(&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(instanceID),
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			engine := aws.StringValue(dbInfo.DBInstances[0].Engine)

			namespace := "AWS/RDS"
			storageMetric := "FreeStorageSpace"
			netRxMetric := "NetworkReceiveThroughput"
			netTxMetric := "NetworkTransmitThroughput"

			if engine == "docdb" {
				namespace = "AWS/DocDB"
				storageMetric = "FreeLocalStorage"
				netRxMetric = "NetworkBytesIn"
				netTxMetric = "NetworkBytesOut"
			}

			cwSvc := cloudwatch.New(sess)
			now := time.Now()
			start := now.Add(-15 * time.Minute)

			result, err := cwSvc.GetMetricData(&cloudwatch.GetMetricDataInput{
				StartTime: aws.Time(start),
				EndTime:   aws.Time(now),
				MetricDataQueries: []*cloudwatch.MetricDataQuery{
					query("cpu", namespace, "CPUUtilization", instanceID),
					query("connections", namespace, "DatabaseConnections", instanceID),
					query("memory", namespace, "FreeableMemory", instanceID),
					query("storage", namespace, storageMetric, instanceID),
					query("read_iops", namespace, "ReadIOPS", instanceID),
					query("write_iops", namespace, "WriteIOPS", instanceID),
					query("read_latency", namespace, "ReadLatency", instanceID),
					query("write_latency", namespace, "WriteLatency", instanceID),
					query("net_rx", namespace, netRxMetric, instanceID),
					query("net_tx", namespace, netTxMetric, instanceID),
				},
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}

			const bytesToMB = 1.0 / (1024 * 1024)
			const bytesToGB = 1.0 / (1024 * 1024 * 1024)
			const secToMS = 1000.0

			metrics := Metrics{
				Identifier:       instanceID,
				Engine:           engine,
				CPUPercent:       latestValue(result.MetricDataResults, "cpu", 1),
				Connections:      latestValue(result.MetricDataResults, "connections", 1),
				FreeableMemoryMB: latestValue(result.MetricDataResults, "memory", bytesToMB),
				FreeStorageGB:    latestValue(result.MetricDataResults, "storage", bytesToGB),
				ReadIOPS:         latestValue(result.MetricDataResults, "read_iops", 1),
				WriteIOPS:        latestValue(result.MetricDataResults, "write_iops", 1),
				ReadLatencyMS:    latestValue(result.MetricDataResults, "read_latency", secToMS),
				WriteLatencyMS:   latestValue(result.MetricDataResults, "write_latency", secToMS),
				NetworkRxMBps:    latestValue(result.MetricDataResults, "net_rx", bytesToMB),
				NetworkTxMBps:    latestValue(result.MetricDataResults, "net_tx", bytesToMB),
			}

			return &mcp.CallToolResult{}, map[string]any{"metrics": metrics}, nil
		},
	})
}
