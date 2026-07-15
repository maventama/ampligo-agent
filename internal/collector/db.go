package collector

// DBSnapshot matches the payload shape expected by POST /api/v1/ingest/db-usage.
type DBSnapshot struct {
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
