package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Database holds an optional local read-only connection the agent uses to
// collect MySQL/Postgres usage metrics. The DSN never leaves this file -
// only the derived metrics are pushed to Ampligo.
type Database struct {
	Type        string `yaml:"type"` // mysql or postgres
	DSN         string `yaml:"dsn"`
	DBIngestURL string `yaml:"db_ingest_url"` // optional override; derived from ingest_url if empty
}

// Config is the agent's local configuration, loaded from agent.yml.
// The API key and ingest URL are the only secrets involved, and they never
// leave this file for anything other than authenticating pushes to Ampligo.
type Config struct {
	APIKey          string    `yaml:"api_key"`
	IngestURL       string    `yaml:"ingest_url"`
	IntervalSeconds int       `yaml:"interval_seconds"`
	Database        *Database `yaml:"database"`
}

const defaultIntervalSeconds = 15

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("config %s: api_key is required", path)
	}
	if cfg.IngestURL == "" {
		return nil, fmt.Errorf("config %s: ingest_url is required", path)
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = defaultIntervalSeconds
	}

	if cfg.Database != nil {
		if cfg.Database.Type != "mysql" && cfg.Database.Type != "postgres" {
			return nil, fmt.Errorf("config %s: database.type must be \"mysql\" or \"postgres\"", path)
		}
		if cfg.Database.DSN == "" {
			return nil, fmt.Errorf("config %s: database.dsn is required when database is set", path)
		}
		if cfg.Database.DBIngestURL == "" {
			cfg.Database.DBIngestURL = strings.Replace(cfg.IngestURL, "/usage", "/db-usage", 1)
		}
	}

	return &cfg, nil
}

func (c *Config) Interval() time.Duration {
	return time.Duration(c.IntervalSeconds) * time.Second
}
