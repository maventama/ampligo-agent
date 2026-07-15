package collector

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLCollector keeps the previous sample's cumulative counters so it can
// report queries_per_second and slow_query_count as rates over the interval
// between two collections, rather than as ever-growing totals.
type MySQLCollector struct {
	db *sql.DB

	lastSampledAt time.Time
	lastQuestions int64
	lastSlow      int64
}

func NewMySQLCollector(dsn string) (*MySQLCollector, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening mysql connection: %w", err)
	}
	db.SetMaxOpenConns(2)
	return &MySQLCollector{db: db}, nil
}

func (c *MySQLCollector) Close() error {
	return c.db.Close()
}

// statusVar reads one counter from SHOW GLOBAL STATUS. MySQL's prepared
// statement protocol doesn't accept placeholders in SHOW commands, so this
// interpolates name directly - safe because callers only ever pass the
// hardcoded literals below, never external input.
func (c *MySQLCollector) statusVar(name string) (int64, error) {
	var varName string
	var value int64
	err := c.db.QueryRow(fmt.Sprintf("SHOW GLOBAL STATUS LIKE '%s'", name)).Scan(&varName, &value)
	return value, err
}

func (c *MySQLCollector) Collect() (DBSnapshot, error) {
	now := time.Now().UTC()
	snap := DBSnapshot{Hostname: hostname(), DBType: "mysql", RecordedAt: now.Format(time.RFC3339)}

	if connected, err := c.statusVar("Threads_connected"); err == nil {
		snap.ConnectionsActive = connected
	} else {
		return snap, fmt.Errorf("reading Threads_connected: %w", err)
	}

	var maxConn int64
	if err := c.db.QueryRow("SELECT @@max_connections").Scan(&maxConn); err == nil {
		snap.ConnectionsMax = maxConn
	}

	questions, qErr := c.statusVar("Questions")
	slow, sErr := c.statusVar("Slow_queries")

	if qErr == nil && sErr == nil && !c.lastSampledAt.IsZero() {
		elapsed := now.Sub(c.lastSampledAt).Seconds()
		if elapsed > 0 {
			snap.QueriesPerSecond = float64(questions-c.lastQuestions) / elapsed
			snap.SlowQueryCount = slow - c.lastSlow
		}
	}
	if qErr == nil {
		c.lastQuestions = questions
	}
	if sErr == nil {
		c.lastSlow = slow
	}
	c.lastSampledAt = now

	readRequests, rrErr := c.statusVar("Innodb_buffer_pool_read_requests")
	reads, rErr := c.statusVar("Innodb_buffer_pool_reads")
	if rrErr == nil && rErr == nil && readRequests > 0 {
		snap.CacheHitRatio = float64(readRequests-reads) / float64(readRequests) * 100
	}

	// Only replicas have a non-empty SHOW SLAVE STATUS result; skip silently on primaries.
	if lag, err := c.replicationLag(); err == nil {
		snap.ReplicationLagSeconds = lag
	}

	return snap, nil
}

func (c *MySQLCollector) replicationLag() (float64, error) {
	rows, err := c.db.Query("SHOW SLAVE STATUS")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	if !rows.Next() {
		return 0, fmt.Errorf("not a replica")
	}

	values := make([]sql.RawBytes, len(cols))
	scanArgs := make([]any, len(cols))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	if err := rows.Scan(scanArgs...); err != nil {
		return 0, err
	}

	for i, col := range cols {
		if col == "Seconds_Behind_Master" && values[i] != nil {
			var lag float64
			_, err := fmt.Sscanf(string(values[i]), "%f", &lag)
			return lag, err
		}
	}

	return 0, fmt.Errorf("Seconds_Behind_Master not found")
}
