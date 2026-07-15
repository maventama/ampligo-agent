package collector

import "os"

// hostname reports the local hostname, falling back to "unknown" so a
// transient os.Hostname() failure doesn't abort the whole collection - the
// server treats a stable identifier as required to tell hosts apart.
func hostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}

// DBSnapshot matches the payload shape expected by POST /api/v1/ingest/db-usage.
type DBSnapshot struct {
	Hostname              string  `json:"hostname"`
	DBType                string  `json:"db_type"`
	ConnectionsActive     int64   `json:"connections_active,omitempty"`
	ConnectionsMax        int64   `json:"connections_max,omitempty"`
	QueriesPerSecond      float64 `json:"queries_per_second,omitempty"`
	SlowQueryCount        int64   `json:"slow_query_count,omitempty"`
	CacheHitRatio         float64 `json:"cache_hit_ratio,omitempty"`
	ReplicationLagSeconds float64 `json:"replication_lag_seconds,omitempty"`
	RecordedAt            string  `json:"recorded_at"`
}

// DBCollector is implemented by MySQLCollector and PostgresCollector.
type DBCollector interface {
	Collect() (DBSnapshot, error)
	Close() error
}
