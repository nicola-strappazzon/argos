package docdb_databases

import (
	"context"
	"fmt"

	docdbdriver "github.com/nicola-strappazzon/argos/internal/drivers/docdb"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type dbEntry struct {
	Name       string  `bson:"name"`
	SizeOnDisk float64 `bson:"sizeOnDisk"`
	Empty      bool    `bson:"empty"`
}

type listDatabasesResult struct {
	Databases []dbEntry `bson:"databases"`
	TotalSize float64   `bson:"totalSize"`
}

type Database struct {
	Name   string  `json:"name"`
	SizeMB float64 `json:"size_mb"`
	Empty  bool    `json:"empty"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "docdb_databases",
		Description: "List databases on a DocumentDB instance with their size and stats.",
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

			var result listDatabasesResult
			err = client.Database("admin").RunCommand(ctx, bson.D{{Key: "listDatabases", Value: 1}}).Decode(&result)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("running listDatabases: %w", err)
			}

			databases := make([]Database, 0, len(result.Databases))
			for _, entry := range result.Databases {
				databases = append(databases, Database{
					Name:   entry.Name,
					SizeMB: entry.SizeOnDisk / 1024 / 1024,
					Empty:  entry.Empty,
				})
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance":      instanceID,
				"databases":     databases,
				"total":         len(databases),
				"total_size_mb": result.TotalSize / 1024 / 1024,
			}, nil
		},
	})
}
