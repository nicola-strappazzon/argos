package aws_docdb_collections

import (
	"context"
	"fmt"

	docdbdriver "github.com/nicola-strappazzon/argos/internal/drivers/docdb"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type collectionEntry struct {
	Name string `bson:"name"`
}

type listCollectionsResult struct {
	Cursor struct {
		FirstBatch []collectionEntry `bson:"firstBatch"`
	} `bson:"cursor"`
}

type collStatsResult struct {
	Count          int64   `bson:"count"`
	Size           float64 `bson:"size"`
	AvgObjSize     float64 `bson:"avgObjSize"`
	NIndexes       int32   `bson:"nindexes"`
	TotalIndexSize float64 `bson:"totalIndexSize"`
}

type Collection struct {
	Name             string  `json:"name"`
	Count            int64   `json:"count"`
	SizeMB           float64 `json:"size_mb"`
	AvgObjSizeBytes  float64 `json:"avg_obj_size_bytes"`
	IndexCount       int32   `json:"index_count"`
	TotalIndexSizeMB float64 `json:"total_index_size_mb"`
}

func init() {
	registry.Add(registry.Property{
		Name:        "aws_docdb_collections",
		Description: "List collections in a DocumentDB database with stats (indexes, size, count).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The DocumentDB instance identifier. Credentials are read from ~/.docdb using this as the section name.",
				},
				"database": map[string]any{
					"type":        "string",
					"description": "The database name to inspect.",
				},
			},
			"required": []string{"db_instance_identifier", "database"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)
			database, _ := args["database"].(string)

			client, err := docdbdriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer client.Disconnect(ctx)

			db := client.Database(database)

			var listResult listCollectionsResult
			err = db.RunCommand(ctx, bson.D{{Key: "listCollections", Value: 1}}).Decode(&listResult)
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("running listCollections: %w", err)
			}

			collections := make([]Collection, 0, len(listResult.Cursor.FirstBatch))
			for _, entry := range listResult.Cursor.FirstBatch {
				var stats collStatsResult
				if err := db.RunCommand(ctx, bson.D{{Key: "collStats", Value: entry.Name}}).Decode(&stats); err != nil {
					collections = append(collections, Collection{Name: entry.Name})
					continue
				}

				collections = append(collections, Collection{
					Name:             entry.Name,
					Count:            stats.Count,
					SizeMB:           stats.Size / 1024 / 1024,
					AvgObjSizeBytes:  stats.AvgObjSize,
					IndexCount:       stats.NIndexes,
					TotalIndexSizeMB: stats.TotalIndexSize / 1024 / 1024,
				})
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance":    instanceID,
				"database":    database,
				"collections": collections,
				"total":       len(collections),
			}, nil
		},
	})
}
