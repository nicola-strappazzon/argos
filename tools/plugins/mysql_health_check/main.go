package mysql_health_check

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/nicola-strappazzon/argos/internal/awsconfig"
	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var reHistoryLength = regexp.MustCompile(`History list length (\d+)`)

// Memory in MB per RDS instance class.
var instanceClassMemoryMB = map[string]float64{
	"db.t3.micro":     1024,
	"db.t3.small":     2048,
	"db.t3.medium":    4096,
	"db.t3.large":     8192,
	"db.t3.xlarge":    16384,
	"db.t3.2xlarge":   32768,
	"db.t4g.micro":    1024,
	"db.t4g.small":    2048,
	"db.t4g.medium":   4096,
	"db.t4g.large":    8192,
	"db.t4g.xlarge":   16384,
	"db.t4g.2xlarge":  32768,
	"db.m5.large":     8192,
	"db.m5.xlarge":    16384,
	"db.m5.2xlarge":   32768,
	"db.m5.4xlarge":   65536,
	"db.m5.8xlarge":   131072,
	"db.m5.12xlarge":  196608,
	"db.m5.16xlarge":  262144,
	"db.m5.24xlarge":  393216,
	"db.m6g.large":    8192,
	"db.m6g.xlarge":   16384,
	"db.m6g.2xlarge":  32768,
	"db.m6g.4xlarge":  65536,
	"db.m6g.8xlarge":  131072,
	"db.m6g.12xlarge": 196608,
	"db.m6g.16xlarge": 262144,
	"db.r5.large":     16384,
	"db.r5.xlarge":    32768,
	"db.r5.2xlarge":   65536,
	"db.r5.4xlarge":   131072,
	"db.r5.8xlarge":   262144,
	"db.r5.12xlarge":  393216,
	"db.r5.16xlarge":  524288,
	"db.r5.24xlarge":  786432,
	"db.r6g.large":    16384,
	"db.r6g.xlarge":   32768,
	"db.r6g.2xlarge":  65536,
	"db.r6g.4xlarge":  131072,
	"db.r6g.8xlarge":  262144,
	"db.r6g.12xlarge": 393216,
	"db.r6g.16xlarge": 524288,
	"db.r7g.large":    16384,
	"db.r7g.xlarge":   32768,
	"db.r7g.2xlarge":  65536,
	"db.r7g.4xlarge":  131072,
	"db.r7g.8xlarge":  262144,
	"db.r7g.12xlarge": 393216,
	"db.r7g.16xlarge": 524288,
}

type Check struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Status      string  `json:"status"`
	Description string  `json:"description"`
	Threshold   string  `json:"threshold"`
}

// queryStatusVars runs SHOW GLOBAL STATUS with the given WHERE/LIKE clause
// and returns a map of variable name → float64 value.
func queryStatusVars(ctx context.Context, db *sql.DB, query string) (map[string]float64, error) {
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

func checkBufferPoolHitRate(ctx context.Context, db *sql.DB) (*Check, error) {
	vars, err := queryStatusVars(ctx, db, "SHOW GLOBAL STATUS LIKE 'Innodb_buffer_pool_read%'")
	if err != nil {
		return nil, fmt.Errorf("buffer pool hit rate: %w", err)
	}

	readRequests := vars["Innodb_buffer_pool_read_requests"]
	reads := vars["Innodb_buffer_pool_reads"]
	if readRequests == 0 {
		return nil, nil
	}

	hitRate := (readRequests - reads) * 100 / readRequests
	status := "ok"
	if hitRate < 95 {
		status = "critical"
	} else if hitRate < 99 {
		status = "warning"
	}

	return &Check{
		Name:        "innodb_buffer_pool_hit_rate",
		Value:       hitRate,
		Unit:        "%",
		Status:      status,
		Description: "Percentage of InnoDB page reads served from the buffer pool without disk I/O. Low values indicate the buffer pool is too small.",
		Threshold:   "ok >= 99%, warning >= 95%, critical < 95%",
	}, nil
}

func checkBufferPoolSizeVsRAM(ctx context.Context, db *sql.DB, instanceClass string) (*Check, error) {
	ramMB, ok := instanceClassMemoryMB[instanceClass]
	if !ok {
		return nil, nil
	}

	vars, err := queryStatusVars(ctx, db, "SHOW GLOBAL VARIABLES LIKE 'innodb_buffer_pool_size'")
	if err != nil {
		return nil, fmt.Errorf("buffer pool size vs RAM: %w", err)
	}

	bufferPoolBytes := vars["innodb_buffer_pool_size"]
	if bufferPoolBytes == 0 {
		return nil, nil
	}

	bufferPoolMB := bufferPoolBytes / 1024 / 1024
	pct := bufferPoolMB * 100 / ramMB

	status := "ok"
	var description string
	if pct > 75 {
		status = "critical"
		description = fmt.Sprintf("Buffer pool (%.0f MB) is above 75%% of RAM (%.0f MB) on %s. Risk of OS memory pressure and swapping.", bufferPoolMB, ramMB, instanceClass)
	} else if pct < 60 {
		status = "warning"
		description = fmt.Sprintf("Buffer pool (%.0f MB) is below 60%% of RAM (%.0f MB) on %s. Consider increasing innodb_buffer_pool_size to improve cache efficiency.", bufferPoolMB, ramMB, instanceClass)
	} else {
		description = fmt.Sprintf("Buffer pool (%.0f MB) is within the recommended range of RAM (%.0f MB) on %s.", bufferPoolMB, ramMB, instanceClass)
	}

	return &Check{
		Name:        "innodb_buffer_pool_size_vs_ram",
		Value:       pct,
		Unit:        "%",
		Status:      status,
		Description: description,
		Threshold:   "ok 60–75% of RAM, warning < 60%, critical > 75%",
	}, nil
}

func checkThreadCacheHitRate(ctx context.Context, db *sql.DB) (*Check, error) {
	vars, err := queryStatusVars(ctx, db, "SHOW GLOBAL STATUS WHERE Variable_name IN ('Threads_created', 'Connections')")
	if err != nil {
		return nil, fmt.Errorf("thread cache hit rate: %w", err)
	}

	connections := vars["Connections"]
	if connections == 0 {
		return nil, nil
	}

	hitRate := 100 - (vars["Threads_created"] * 100 / connections)
	status := "ok"
	var description string
	if hitRate <= 90 {
		status = "warning"
		description = fmt.Sprintf("Thread cache hit rate is %.2f%%. Many threads are being created instead of reused. Consider increasing thread_cache_size.", hitRate)
	} else {
		description = fmt.Sprintf("Thread cache hit rate is %.2f%%. Most connections reuse cached threads efficiently.", hitRate)
	}

	return &Check{
		Name:        "thread_cache_hit_rate",
		Value:       hitRate,
		Unit:        "%",
		Status:      status,
		Description: description,
		Threshold:   "ok > 90%, warning <= 90%",
	}, nil
}

func checkTemporaryTablesOnDisk(ctx context.Context, db *sql.DB) (*Check, error) {
	vars, err := queryStatusVars(ctx, db, "SHOW GLOBAL STATUS WHERE Variable_name IN ('Created_tmp_disk_tables', 'Created_tmp_tables')")
	if err != nil {
		return nil, fmt.Errorf("temporary tables on disk: %w", err)
	}

	tmpTotal := vars["Created_tmp_tables"]
	if tmpTotal == 0 {
		return nil, nil
	}

	tmpDisk := vars["Created_tmp_disk_tables"]
	pct := tmpDisk * 100 / tmpTotal
	status := "ok"
	var description string
	if pct > 25 {
		status = "warning"
		description = fmt.Sprintf("%.2f%% of temporary tables are created on disk (%.0f of %.0f). Increase tmp_table_size and max_heap_table_size, and review queries using ORDER BY / GROUP BY without an index.", pct, tmpDisk, tmpTotal)
	} else {
		description = fmt.Sprintf("%.2f%% of temporary tables are created on disk (%.0f of %.0f). Most temporary tables fit in memory.", pct, tmpDisk, tmpTotal)
	}

	return &Check{
		Name:        "temporary_tables_on_disk",
		Value:       pct,
		Unit:        "%",
		Status:      status,
		Description: description,
		Threshold:   "ok <= 25%, warning > 25%",
	}, nil
}

func checkHistoryListLength(ctx context.Context, db *sql.DB) (*Check, error) {
	rows, err := db.QueryContext(ctx, "SHOW ENGINE INNODB STATUS")
	if err != nil {
		return nil, fmt.Errorf("history list length: %w", err)
	}
	defer rows.Close()

	var raw string
	if rows.Next() {
		var engineType, name string
		if err := rows.Scan(&engineType, &name, &raw); err != nil {
			return nil, fmt.Errorf("history list length: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history list length: %w", err)
	}

	m := reHistoryLength.FindStringSubmatch(raw)
	if m == nil {
		return nil, nil
	}

	hll, _ := strconv.ParseInt(m[1], 10, 64)
	var status, description string
	switch {
	case hll > 1_000_000:
		status = "critical"
		description = fmt.Sprintf("History List Length is %d. EMERGENCY: long-running transactions are severely bloating the undo log. Identify and kill blocking transactions immediately.", hll)
	case hll > 100_000:
		status = "critical"
		description = fmt.Sprintf("History List Length is %d. Serious problem: undo log is growing uncontrolled. Find and close long-running or idle open transactions.", hll)
	case hll > 10_000:
		status = "warning"
		description = fmt.Sprintf("History List Length is %d. Open or slow transactions are holding back InnoDB purge. Review long-running transactions.", hll)
	case hll > 1_000:
		status = "ok"
		description = fmt.Sprintf("History List Length is %d. Normal under load.", hll)
	default:
		status = "ok"
		description = fmt.Sprintf("History List Length is %d. Excellent.", hll)
	}

	return &Check{
		Name:        "history_list_length",
		Value:       float64(hll),
		Unit:        "rows",
		Status:      status,
		Description: description,
		Threshold:   "ok < 10,000 | warning 10,000–100,000 | critical > 100,000 | emergency > 1,000,000",
	}, nil
}

func checkMaxConnectionsUsage(ctx context.Context, db *sql.DB) (*Check, error) {
	statusVars, err := queryStatusVars(ctx, db, "SHOW GLOBAL STATUS WHERE Variable_name = 'Threads_connected'")
	if err != nil {
		return nil, fmt.Errorf("max connections usage: %w", err)
	}

	configVars, err := queryStatusVars(ctx, db, "SHOW GLOBAL VARIABLES WHERE Variable_name = 'max_connections'")
	if err != nil {
		return nil, fmt.Errorf("max connections usage: %w", err)
	}

	maxConnections := configVars["max_connections"]
	if maxConnections == 0 {
		return nil, nil
	}

	threadsConnected := statusVars["Threads_connected"]
	pct := threadsConnected * 100 / maxConnections

	var status, description string
	switch {
	case pct > 80:
		status = "critical"
		description = fmt.Sprintf("%.0f of %.0f connections in use (%.2f%%). Risk of hitting the connection limit. Increase max_connections or reduce connection usage (e.g. use a connection pooler).", threadsConnected, maxConnections, pct)
	case pct > 70:
		status = "warning"
		description = fmt.Sprintf("%.0f of %.0f connections in use (%.2f%%). Approaching the connection limit. Monitor closely and consider a connection pooler.", threadsConnected, maxConnections, pct)
	default:
		status = "ok"
		description = fmt.Sprintf("%.0f of %.0f connections in use (%.2f%%). Connection usage is within safe limits.", threadsConnected, maxConnections, pct)
	}

	return &Check{
		Name:        "max_connections_usage",
		Value:       pct,
		Unit:        "%",
		Status:      status,
		Description: description,
		Threshold:   "ok <= 70%, warning > 70%, critical > 80%",
	}, nil
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_health_check",
		Description: "Run health checks on a MySQL instance. Returns key metrics with status (ok/warning/critical) and thresholds. Checks include: InnoDB Buffer Pool Hit Rate.",
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

			sess, err := awsconfig.NewSession()
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			svc := rds.New(sess)
			rdsResult, err := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(instanceID),
			})
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("describing RDS instance: %w", err)
			}
			if len(rdsResult.DBInstances) == 0 {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("instance %s not found", instanceID)
			}
			instanceClass := aws.StringValue(rdsResult.DBInstances[0].DBInstanceClass)

			db, err := mysqldriver.Connect(instanceID)
			if err != nil {
				return &mcp.CallToolResult{}, nil, err
			}
			defer db.Close()

			type checkFn func() (*Check, error)
			runners := []checkFn{
				func() (*Check, error) { return checkBufferPoolHitRate(ctx, db) },
				func() (*Check, error) { return checkBufferPoolSizeVsRAM(ctx, db, instanceClass) },
				func() (*Check, error) { return checkThreadCacheHitRate(ctx, db) },
				func() (*Check, error) { return checkTemporaryTablesOnDisk(ctx, db) },
				func() (*Check, error) { return checkHistoryListLength(ctx, db) },
				func() (*Check, error) { return checkMaxConnectionsUsage(ctx, db) },
			}

			var checks []Check
			for _, run := range runners {
				c, err := run()
				if err != nil {
					return &mcp.CallToolResult{}, nil, err
				}
				if c != nil {
					checks = append(checks, *c)
				}
			}

			return &mcp.CallToolResult{}, map[string]any{
				"instance": instanceID,
				"checks":   checks,
				"total":    len(checks),
			}, nil
		},
	})
}
