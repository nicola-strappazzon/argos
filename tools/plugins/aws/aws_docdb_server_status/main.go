package aws_docdb_server_status

import (
	"context"
	"fmt"

	docdbdriver "github.com/nicola-strappazzon/argos/internal/drivers/docdb"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type serverStatusResult struct {
	Host    string  `bson:"host"`
	Version string  `bson:"version"`
	Uptime  float64 `bson:"uptime"`

	Connections struct {
		Current      int64 `bson:"current"`
		Available    int64 `bson:"available"`
		TotalCreated int64 `bson:"totalCreated"`
	} `bson:"connections"`

	Opcounters struct {
		Insert  int64 `bson:"insert"`
		Query   int64 `bson:"query"`
		Update  int64 `bson:"update"`
		Delete  int64 `bson:"delete"`
		Getmore int64 `bson:"getmore"`
		Command int64 `bson:"command"`
	} `bson:"opcounters"`

	Mem struct {
		ResidentMB int64 `bson:"resident"`
		VirtualMB  int64 `bson:"virtual"`
	} `bson:"mem"`

	Network struct {
		BytesIn     int64 `bson:"bytesIn"`
		BytesOut    int64 `bson:"bytesOut"`
		NumRequests int64 `bson:"numRequests"`
	} `bson:"network"`

	GlobalLock struct {
		TotalTimeMicros int64 `bson:"totalTime"`
		CurrentQueue    struct {
			Total   int64 `bson:"total"`
			Readers int64 `bson:"readers"`
			Writers int64 `bson:"writers"`
		} `bson:"currentQueue"`
		ActiveClients struct {
			Total   int64 `bson:"total"`
			Readers int64 `bson:"readers"`
			Writers int64 `bson:"writers"`
		} `bson:"activeClients"`
	} `bson:"globalLock"`
}

func init() {
	registry.Add(registry.Property{
		Name: "aws_docdb_server_status",
		Description: "Show DocumentDB server telemetry (equivalent to serverStatus in MongoDB). " +
			"Returns connections, operation counters, memory usage, network I/O, and global lock statistics.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The DocumentDB instance identifier. Credentials are read from ~/.docdb using this as the section name.",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)

			client, err := docdbdriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer client.Disconnect(ctx)

			var result serverStatusResult
			err = client.Database("admin").RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("serverStatus: %w", err)
			}

			uptimeHours := result.Uptime / 3600

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"host":     result.Host,
				"version":  result.Version,
				"uptime": map[string]any{
					"seconds": result.Uptime,
					"hours":   uptimeHours,
				},
				"connections": map[string]any{
					"current":       result.Connections.Current,
					"available":     result.Connections.Available,
					"total_created": result.Connections.TotalCreated,
				},
				"opcounters": map[string]any{
					"insert":  result.Opcounters.Insert,
					"query":   result.Opcounters.Query,
					"update":  result.Opcounters.Update,
					"delete":  result.Opcounters.Delete,
					"getmore": result.Opcounters.Getmore,
					"command": result.Opcounters.Command,
				},
				"memory": map[string]any{
					"resident_mb": result.Mem.ResidentMB,
					"virtual_mb":  result.Mem.VirtualMB,
				},
				"network": map[string]any{
					"bytes_in":     result.Network.BytesIn,
					"bytes_out":    result.Network.BytesOut,
					"num_requests": result.Network.NumRequests,
				},
				"global_lock": map[string]any{
					"total_time_secs": result.GlobalLock.TotalTimeMicros / 1_000_000,
					"current_queue": map[string]any{
						"total":   result.GlobalLock.CurrentQueue.Total,
						"readers": result.GlobalLock.CurrentQueue.Readers,
						"writers": result.GlobalLock.CurrentQueue.Writers,
					},
					"active_clients": map[string]any{
						"total":   result.GlobalLock.ActiveClients.Total,
						"readers": result.GlobalLock.ActiveClients.Readers,
						"writers": result.GlobalLock.ActiveClients.Writers,
					},
				},
			}, nil
		},
	})
}
