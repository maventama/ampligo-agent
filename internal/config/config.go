package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the agent's local configuration, loaded from agent.yml.
// The API key and ingest URL are the only secrets involved, and they never
// leave this file for anything other than authenticating pushes to Ampligo.
type Config struct {
	APIKey          string `yaml:"api_key"`
	IngestURL       string `yaml:"ingest_url"`
	IntervalSeconds int    `yaml:"interval_seconds"`
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

	return &cfg, nil
}

func (c *Config) Interval() time.Duration {
	return time.Duration(c.IntervalSeconds) * time.Second
}
