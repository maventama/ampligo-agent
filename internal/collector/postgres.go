package collector

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresCollector mirrors MySQLCollector: it keeps the previous cumulative
// transaction count so queries_per_second can be reported as a rate.
type PostgresCollector struct {
	db *sql.DB

	lastSampledAt time.Time
	lastTxns      int64
}

func NewPostgresCollector(dsn string) (*PostgresCollector, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening postgres connection: %w", err)
	}
	db.SetMaxOpenConns(2)
	return &PostgresCollector{db: db}, nil
}

func (c *PostgresCollector) Close() error {
	return c.db.Close()
}

func (c *PostgresCollector) Collect() (DBSnapshot, error) {
	now := time.Now().UTC()
	snap := DBSnapshot{DBType: "postgres", RecordedAt: now.Format(time.RFC3339)}

	var active int64
	if err := c.db.QueryRow("SELECT count(*) FROM pg_stat_activity WHERE state = 'active'").Scan(&active); err != nil {
		return snap, fmt.Errorf("reading active connections: %w", err)
	}
	snap.ConnectionsActive = active

	var maxConn int64
	if err := c.db.QueryRow("SHOW max_connections").Scan(&maxConn); err == nil {
		snap.ConnectionsMax = maxConn
	}

	var txns int64
	var blksHit, blksRead int64
	err := c.db.QueryRow(`
		SELECT coalesce(sum(xact_commit + xact_rollback), 0),
		       coalesce(sum(blks_hit), 0),
		       coalesce(sum(blks_read), 0)
		FROM pg_stat_database
	`).Scan(&txns, &blksHit, &blksRead)
	if err != nil {
		return snap, fmt.Errorf("reading pg_stat_database: %w", err)
	}

	if !c.lastSampledAt.IsZero() {
		elapsed := now.Sub(c.lastSampledAt).Seconds()
		if elapsed > 0 {
			snap.QueriesPerSecond = float64(txns-c.lastTxns) / elapsed
		}
	}
	c.lastTxns = txns
	c.lastSampledAt = now

	if blksHit+blksRead > 0 {
		snap.CacheHitRatio = float64(blksHit) / float64(blksHit+blksRead) * 100
	}

	var lag sql.NullFloat64
	if err := c.db.QueryRow(`
		SELECT CASE WHEN pg_is_in_recovery()
			THEN EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()))
			ELSE NULL END
	`).Scan(&lag); err == nil && lag.Valid {
		snap.ReplicationLagSeconds = lag.Float64
	}

	return snap, nil
}
