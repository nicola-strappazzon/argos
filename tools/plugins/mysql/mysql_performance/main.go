package mysql_performance

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Recommendation struct {
	Name             string `json:"name"`
	Status           string `json:"status"`
	CurrentValue     string `json:"current_value"`
	RecommendedValue string `json:"recommended_value,omitempty"`
	Description      string `json:"description"`
}

func queryVars(ctx context.Context, db *sql.DB, query string) (map[string]float64, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vars := make(map[string]float64)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			vars[name] = f
		}
	}
	return vars, rows.Err()
}

func mb(bytes float64) float64 {
	return bytes / 1024 / 1024
}

// bestChunkSize finds the chunk_size (in bytes) closest to the 3.5% midpoint of
// the 2–5% optimal range that satisfies both constraints:
//  1. chunk_size is a multiple of 1 MB
//  2. pool_size is exactly divisible by (instances × chunk_size)
//
// It first searches candidates within the 2–5% range; if none exist (e.g. the
// pool is too small relative to the number of instances), it falls back to the
// valid candidate closest to the 3.5% target.
func bestChunkSize(poolSize, instances float64) float64 {
	oneMB := float64(1024 * 1024)
	poolPerInstance := poolSize / instances
	target := poolSize * 0.035 // 3.5% midpoint

	var bestInRange, bestFallback float64
	bestInRangeDiff := math.MaxFloat64
	bestFallbackDiff := math.MaxFloat64

	for n := 1; n <= 10000; n++ {
		// chunk must divide poolPerInstance exactly
		if math.Mod(poolPerInstance, float64(n)) != 0 {
			continue
		}
		chunk := poolPerInstance / float64(n)
		// chunk must be a multiple of 1 MB
		if math.Mod(chunk, oneMB) != 0 {
			continue
		}
		pct := chunk * 100.0 / poolSize
		diff := math.Abs(chunk - target)

		if pct >= 2 && pct <= 5 {
			if diff < bestInRangeDiff {
				bestInRangeDiff = diff
				bestInRange = chunk
			}
		} else {
			if diff < bestFallbackDiff {
				bestFallbackDiff = diff
				bestFallback = chunk
			}
		}
	}

	if bestInRange > 0 {
		return bestInRange
	}
	return bestFallback
}

func checkInnoDBBufferPoolChunkSize(ctx context.Context, db *sql.DB) (*Recommendation, error) {
	vars, err := queryVars(ctx, db, `SHOW GLOBAL VARIABLES WHERE Variable_name IN
		('innodb_buffer_pool_size', 'innodb_buffer_pool_instances', 'innodb_buffer_pool_chunk_size')`)
	if err != nil {
		return nil, fmt.Errorf("innodb_buffer_pool_chunk_size: %w", err)
	}

	poolSize := vars["innodb_buffer_pool_size"]
	instances := vars["innodb_buffer_pool_instances"]
	chunkSize := vars["innodb_buffer_pool_chunk_size"]

	if poolSize == 0 || instances == 0 || chunkSize == 0 {
		return nil, nil
	}

	unit := instances * chunkSize
	isAligned := math.Mod(poolSize, unit) == 0
	chunkPct := chunkSize * 100.0 / poolSize

	currentValue := fmt.Sprintf(
		"innodb_buffer_pool_size=%.0f MB, innodb_buffer_pool_instances=%.0f, innodb_buffer_pool_chunk_size=%.0f MB",
		mb(poolSize), instances, mb(chunkSize),
	)

	recommendedChunk := bestChunkSize(poolSize, instances)
	recommendedPct := recommendedChunk * 100.0 / poolSize
	chunksPerInstance := poolSize / (instances * recommendedChunk)

	var status, description, recommendedValue string

	switch {
	case !isAligned:
		status = "warning"
		description = fmt.Sprintf(
			"innodb_buffer_pool_size (%.0f MB) is not a multiple of innodb_buffer_pool_instances (%.0f) × innodb_buffer_pool_chunk_size (%.0f MB) = %.0f MB. "+
				"MySQL will silently auto-adjust the buffer pool upward to the next valid multiple, so the actual memory used differs from what is configured.",
			mb(poolSize), instances, mb(chunkSize), mb(unit),
		)
		recommendedValue = fmt.Sprintf(
			"innodb_buffer_pool_chunk_size=%.0f MB (%.2f%% of buffer pool, %.0f chunk(s) per instance)",
			mb(recommendedChunk), recommendedPct, chunksPerInstance,
		)

	case chunkPct < 2:
		status = "warning"
		description = fmt.Sprintf(
			"innodb_buffer_pool_chunk_size (%.0f MB) is %.2f%% of innodb_buffer_pool_size (%.0f MB), below the recommended 2–5%%. "+
				"Too many small chunks increase memory management overhead during buffer pool resizing operations.",
			mb(chunkSize), chunkPct, mb(poolSize),
		)
		recommendedValue = fmt.Sprintf(
			"innodb_buffer_pool_chunk_size=%.0f MB (%.2f%% of buffer pool, %.0f chunk(s) per instance)",
			mb(recommendedChunk), recommendedPct, chunksPerInstance,
		)

	case chunkPct > 5:
		status = "warning"
		description = fmt.Sprintf(
			"innodb_buffer_pool_chunk_size (%.0f MB) is %.2f%% of innodb_buffer_pool_size (%.0f MB), above the recommended 2–5%%. "+
				"Large chunks make online buffer pool resizing coarser and less flexible.",
			mb(chunkSize), chunkPct, mb(poolSize),
		)
		recommendedValue = fmt.Sprintf(
			"innodb_buffer_pool_chunk_size=%.0f MB (%.2f%% of buffer pool, %.0f chunk(s) per instance)",
			mb(recommendedChunk), recommendedPct, chunksPerInstance,
		)

	default:
		status = "ok"
		description = fmt.Sprintf(
			"innodb_buffer_pool_size (%.0f MB) is correctly aligned with %.0f instance(s) × %.0f MB chunk_size = %.0f MB. "+
				"Chunk size is %.2f%% of the buffer pool, within the optimal 2–5%% range.",
			mb(poolSize), instances, mb(chunkSize), mb(unit), chunkPct,
		)
	}

	return &Recommendation{
		Name:             "innodb_buffer_pool_chunk_size",
		Status:           status,
		CurrentValue:     currentValue,
		RecommendedValue: recommendedValue,
		Description:      description,
	}, nil
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_performance",
		Description: "Analyze MySQL configuration variables and return tuning recommendations with status (ok/warning/critical). Checks whether key variables are optimally set for the instance's resources. Recommendations include current values and suggested changes. Checks: InnoDB buffer pool chunk size alignment and sizing.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"db_instance_identifier": map[string]any{
					"type":        "string",
					"description": "The RDS DB instance identifier. Credentials are read from ~/.my.cnf using this as the section name.",
				},
			},
			"required": []string{"db_instance_identifier"},
		},
		Function: func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			instanceID, _ := args["db_instance_identifier"].(string)

			db, err := mysqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			type checkFn func() (*Recommendation, error)
			runners := []checkFn{
				func() (*Recommendation, error) { return checkInnoDBBufferPoolChunkSize(ctx, db) },
			}

			var recommendations []Recommendation
			for _, run := range runners {
				r, err := run()
				if err != nil {
					return &mcp.CallToolResult{}, nil, err
				}
				if r != nil {
					recommendations = append(recommendations, *r)
				}
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance":        instanceID,
				"recommendations": recommendations,
				"total":           len(recommendations),
			}, nil
		},
	})
}
