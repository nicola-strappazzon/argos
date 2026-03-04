package docdb_current_ops

import (
	"context"
	"fmt"

	docdbdriver "github.com/nicola-strappazzon/argos/internal/drivers/docdb"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Operation struct {
	OpID        any    `json:"opid"`
	SecsRunning int64  `json:"secs_running"`
	Op          string `json:"op"`
	NS          string `json:"ns"`
	Command     any    `json:"command,omitempty"`
	Client      string `json:"client"`
	Desc        string `json:"desc"`
}

type opEntry struct {
	OpID        any    `bson:"opid"`
	SecsRunning int64  `bson:"secs_running"`
	Op          string `bson:"op"`
	NS          string `bson:"ns"`
	Command     bson.D `bson:"command"`
	Client      string `bson:"client"`
	Desc        string `bson:"desc"`
}

type currentOpResult struct {
	Inprog []opEntry `bson:"inprog"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "docdb_current_ops",
		Description: "Show active operations on a DocumentDB instance (equivalent to db.currentOp()).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The DocumentDB instance identifier. Credentials are read from ~/.docdb using this as the section name.",
				},
				"min_secs": map[string]any{
					"type":        "integer",
					"description": "Only show operations running for at least this many seconds. Default: 0 (show all).",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			minSecs := int64(0)
			if m, ok := args["min_secs"].(float64); ok && m > 0 {
				minSecs = int64(m)
			}

			client, err := docdbdriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer client.Disconnect(ctx)

			var result currentOpResult
			err = client.Database("admin").RunCommand(ctx, bson.D{{Key: "currentOp", Value: 1}}).Decode(&result)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("running currentOp: %w", err)
			}

			ops := make([]Operation, 0, len(result.Inprog))
			for _, entry := range result.Inprog {
				if entry.SecsRunning < minSecs {
					continue
				}
				if entry.NS == "admin.$cmd" && len(entry.Command) > 0 && entry.Command[0].Key == "currentOp" {
					continue
				}
				ops = append(ops, Operation{
					OpID:        entry.OpID,
					SecsRunning: entry.SecsRunning,
					Op:          entry.Op,
					NS:          entry.NS,
					Command:     entry.Command,
					Client:      entry.Client,
					Desc:        entry.Desc,
				})
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"inprog":   ops,
				"total":    len(ops),
			}, nil
		},
	})
}
