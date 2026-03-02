package mysql_status

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	mysqldriver "github.com/nicola-strappazzon/argos/internal/drivers/mysql"
	"github.com/nicola-strappazzon/argos/tools/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Semaphores struct {
	ReservationCount int64 `json:"reservation_count"`
	SignalCount      int64 `json:"signal_count"`
	RWSharedSpins    int64 `json:"rw_shared_spins"`
	RWSharedOSWaits  int64 `json:"rw_shared_os_waits"`
	RWExclSpins      int64 `json:"rw_excl_spins"`
	RWExclOSWaits    int64 `json:"rw_excl_os_waits"`
}

type DeadlockTx struct {
	Query string `json:"query"`
}

type Deadlock struct {
	Timestamp    string       `json:"timestamp"`
	Transactions []DeadlockTx `json:"transactions"`
	Victim       int          `json:"victim_transaction"`
}

type Transactions struct {
	TrxIDCounter  int64 `json:"trx_id_counter"`
	HistoryLength int64 `json:"history_list_length"`
}

type FileIO struct {
	ReadsPerSec     float64 `json:"reads_per_sec"`
	WritesPerSec    float64 `json:"writes_per_sec"`
	LogWritesPerSec float64 `json:"log_writes_per_sec"`
}

type Log struct {
	SequenceNumber int64 `json:"sequence_number"`
	FlushedUpTo    int64 `json:"flushed_up_to"`
	LastCheckpoint int64 `json:"last_checkpoint"`
}

type BufferPool struct {
	SizePages     int64   `json:"size_pages"`
	FreePages     int64   `json:"free_pages"`
	DatabasePages int64   `json:"database_pages"`
	ModifiedPages int64   `json:"modified_pages"`
	HitRatePct    float64 `json:"hit_rate_pct"`
	ReadsPerSec   float64 `json:"reads_per_sec"`
	WritesPerSec  float64 `json:"writes_per_sec"`
}

type RowOperations struct {
	QueriesInsideInnoDB int64   `json:"queries_inside_innodb"`
	QueriesInQueue      int64   `json:"queries_in_queue"`
	InsertsPerSec       float64 `json:"inserts_per_sec"`
	UpdatesPerSec       float64 `json:"updates_per_sec"`
	DeletesPerSec       float64 `json:"deletes_per_sec"`
	ReadsPerSec         float64 `json:"reads_per_sec"`
}

type InnoDBStatus struct {
	Instance     string        `json:"instance"`
	Timestamp    string        `json:"timestamp"`
	Semaphores   Semaphores    `json:"semaphores"`
	Deadlock     *Deadlock     `json:"latest_deadlock,omitempty"`
	Transactions Transactions  `json:"transactions"`
	FileIO       FileIO        `json:"file_io"`
	Log          Log           `json:"log"`
	BufferPool   BufferPool    `json:"buffer_pool"`
	RowOps       RowOperations `json:"row_operations"`
}

var (
	reTimestamp      = regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) \d+ INNODB MONITOR OUTPUT`)
	reReservation    = regexp.MustCompile(`OS WAIT ARRAY INFO: reservation count (\d+)`)
	reSignal         = regexp.MustCompile(`OS WAIT ARRAY INFO: signal count (\d+)`)
	reRWShared       = regexp.MustCompile(`RW-shared spins (\d+), rounds \d+, OS waits (\d+)`)
	reRWExcl         = regexp.MustCompile(`RW-excl spins (\d+), rounds \d+, OS waits (\d+)`)
	reDeadlockTime   = regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) \d+\n\*\*\*`)
	reTxQuery        = regexp.MustCompile(`MySQL thread id \d+[^\n]*\n([^\n]+)`)
	reDeadlockVictim = regexp.MustCompile(`WE ROLL BACK TRANSACTION \((\d+)\)`)
	reTrxIDCounter   = regexp.MustCompile(`Trx id counter (\d+)`)
	reHistoryLength  = regexp.MustCompile(`History list length (\d+)`)
	reFileIORate     = regexp.MustCompile(`(\d+\.?\d*) reads/s, \d+ avg bytes/read, (\d+\.?\d*) writes/s, (\d+\.?\d*) log writes/s`)
	reLogSeq         = regexp.MustCompile(`Log sequence number (\d+)`)
	reLogFlushed     = regexp.MustCompile(`Log flushed up to\s+(\d+)`)
	reCheckpoint     = regexp.MustCompile(`Last checkpoint at\s+(\d+)`)
	reBPSize         = regexp.MustCompile(`Buffer pool size\s+(\d+)`)
	reBPFree         = regexp.MustCompile(`Free buffers\s+(\d+)`)
	reBPDatabase     = regexp.MustCompile(`Database pages\s+(\d+)`)
	reBPModified     = regexp.MustCompile(`Modified db pages\s+(\d+)`)
	reBPHitRate      = regexp.MustCompile(`Buffer pool hit rate (\d+) / (\d+)`)
	reBPRWRate       = regexp.MustCompile(`(\d+\.?\d*) reads/s, (\d+\.?\d*) creates/s, (\d+\.?\d*) writes/s`)
	reQueriesInside  = regexp.MustCompile(`(\d+) queries inside InnoDB, (\d+) queries in queue`)
	reRowOpsRate     = regexp.MustCompile(`(\d+\.?\d*) inserts/s, (\d+\.?\d*) updates/s, (\d+\.?\d*) deletes/s, (\d+\.?\d*) reads/s`)
)

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func parseFloat64(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// sectionBetween returns the text between two string markers (exclusive).
// Useful to restrict regex parsing to a specific section.
func sectionBetween(text, start, end string) string {
	si := strings.Index(text, start)
	if si == -1 {
		return ""
	}
	si += len(start)
	ei := strings.Index(text[si:], end)
	if ei == -1 {
		return text[si:]
	}
	return text[si : si+ei]
}

func parseInnoDBStatus(instanceID, raw string) InnoDBStatus {
	out := InnoDBStatus{Instance: instanceID}

	// Timestamp from the header line.
	if m := reTimestamp.FindStringSubmatch(raw); m != nil {
		out.Timestamp = m[1]
	}

	// Semaphores.
	if m := reReservation.FindStringSubmatch(raw); m != nil {
		out.Semaphores.ReservationCount = parseInt64(m[1])
	}
	if m := reSignal.FindStringSubmatch(raw); m != nil {
		out.Semaphores.SignalCount = parseInt64(m[1])
	}
	if m := reRWShared.FindStringSubmatch(raw); m != nil {
		out.Semaphores.RWSharedSpins = parseInt64(m[1])
		out.Semaphores.RWSharedOSWaits = parseInt64(m[2])
	}
	if m := reRWExcl.FindStringSubmatch(raw); m != nil {
		out.Semaphores.RWExclSpins = parseInt64(m[1])
		out.Semaphores.RWExclOSWaits = parseInt64(m[2])
	}

	// Latest deadlock — restrict parsing to the deadlock section to avoid
	// matching queries from other sections (e.g. TRANSACTIONS list).
	if strings.Contains(raw, "LATEST DETECTED DEADLOCK") {
		dl := &Deadlock{}
		dlSection := sectionBetween(raw, "LATEST DETECTED DEADLOCK\n", "\nTRANSACTIONS\n")
		if m := reDeadlockTime.FindStringSubmatch(dlSection); m != nil {
			dl.Timestamp = m[1]
		}
		for _, q := range reTxQuery.FindAllStringSubmatch(dlSection, -1) {
			query := strings.TrimSpace(q[1])
			if query != "" {
				dl.Transactions = append(dl.Transactions, DeadlockTx{Query: query})
			}
		}
		if m := reDeadlockVictim.FindStringSubmatch(dlSection); m != nil {
			dl.Victim, _ = strconv.Atoi(m[1])
		}
		out.Deadlock = dl
	}

	// Transactions.
	if m := reTrxIDCounter.FindStringSubmatch(raw); m != nil {
		out.Transactions.TrxIDCounter = parseInt64(m[1])
	}
	if m := reHistoryLength.FindStringSubmatch(raw); m != nil {
		out.Transactions.HistoryLength = parseInt64(m[1])
	}

	// File I/O — pattern includes "avg bytes/read" so it won't match buffer pool.
	if m := reFileIORate.FindStringSubmatch(raw); m != nil {
		out.FileIO.ReadsPerSec = parseFloat64(m[1])
		out.FileIO.WritesPerSec = parseFloat64(m[2])
		out.FileIO.LogWritesPerSec = parseFloat64(m[3])
	}

	// Log.
	if m := reLogSeq.FindStringSubmatch(raw); m != nil {
		out.Log.SequenceNumber = parseInt64(m[1])
	}
	if m := reLogFlushed.FindStringSubmatch(raw); m != nil {
		out.Log.FlushedUpTo = parseInt64(m[1])
	}
	if m := reCheckpoint.FindStringSubmatch(raw); m != nil {
		out.Log.LastCheckpoint = parseInt64(m[1])
	}

	// Buffer pool — all patterns here are unique to this section.
	if m := reBPSize.FindStringSubmatch(raw); m != nil {
		out.BufferPool.SizePages = parseInt64(m[1])
	}
	if m := reBPFree.FindStringSubmatch(raw); m != nil {
		out.BufferPool.FreePages = parseInt64(m[1])
	}
	if m := reBPDatabase.FindStringSubmatch(raw); m != nil {
		out.BufferPool.DatabasePages = parseInt64(m[1])
	}
	if m := reBPModified.FindStringSubmatch(raw); m != nil {
		out.BufferPool.ModifiedPages = parseInt64(m[1])
	}
	if m := reBPHitRate.FindStringSubmatch(raw); m != nil {
		total := parseInt64(m[2])
		if total > 0 {
			out.BufferPool.HitRatePct = float64(parseInt64(m[1])) / float64(total) * 100
		}
	}
	// Pattern includes "creates/s" so it won't match the FILE I/O reads/writes line.
	if m := reBPRWRate.FindStringSubmatch(raw); m != nil {
		out.BufferPool.ReadsPerSec = parseFloat64(m[1])
		out.BufferPool.WritesPerSec = parseFloat64(m[3])
	}

	// Row operations.
	if m := reQueriesInside.FindStringSubmatch(raw); m != nil {
		out.RowOps.QueriesInsideInnoDB = parseInt64(m[1])
		out.RowOps.QueriesInQueue = parseInt64(m[2])
	}
	if m := reRowOpsRate.FindStringSubmatch(raw); m != nil {
		out.RowOps.InsertsPerSec = parseFloat64(m[1])
		out.RowOps.UpdatesPerSec = parseFloat64(m[2])
		out.RowOps.DeletesPerSec = parseFloat64(m[3])
		out.RowOps.ReadsPerSec = parseFloat64(m[4])
	}

	return out
}

func init() {
	registry.Add(registry.Property{
		Name:        "mysql_status",
		Description: "Run SHOW ENGINE INNODB STATUS on a MySQL instance and return the full output.",
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

			rows, err := db.QueryContext(ctx, "SHOW ENGINE INNODB STATUS")
			if err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("executing query: %w", err)
			}
			defer rows.Close()

			var engineType, name, status string
			if rows.Next() {
				if err := rows.Scan(&engineType, &name, &status); err != nil {
					return &mcp.CallToolResult{}, nil, fmt.Errorf("scanning row: %w", err)
				}
			}
			if err := rows.Err(); err != nil {
				return &mcp.CallToolResult{}, nil, fmt.Errorf("reading rows: %w", err)
			}

			return &mcp.CallToolResult{}, parseInnoDBStatus(instanceID, status), nil
		},
	})
}
