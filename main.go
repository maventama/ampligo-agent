package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/niago-id/ampligo-agent/internal/collector"
	"github.com/niago-id/ampligo-agent/internal/config"
	"github.com/niago-id/ampligo-agent/internal/pusher"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	configPath := flag.String("config", "/etc/ampligo/agent.yml", "path to agent.yml")
	diskPath := flag.String("disk-path", "/", "filesystem path to report disk usage for")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("ampligo-agent " + version)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("ampligo-agent: %v", err)
	}

	p := pusher.New(cfg.IngestURL, cfg.APIKey)

	var dbCollector collector.DBCollector
	if cfg.Database != nil {
		dbCollector, err = newDBCollector(cfg.Database.Type, cfg.Database.DSN)
		if err != nil {
			log.Fatalf("ampligo-agent: database collector: %v", err)
		}
		defer dbCollector.Close()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(cfg.Interval())
	defer ticker.Stop()

	log.Printf("ampligo-agent %s: pushing every %s to %s", version, cfg.Interval(), cfg.IngestURL)
	tick(p, *diskPath, dbCollector, cfg)

	for {
		select {
		case <-ctx.Done():
			log.Println("ampligo-agent: shutting down")
			return
		case <-ticker.C:
			tick(p, *diskPath, dbCollector, cfg)
		}
	}
}

func newDBCollector(dbType, dsn string) (collector.DBCollector, error) {
	switch dbType {
	case "mysql":
		return collector.NewMySQLCollector(dsn)
	case "postgres":
		return collector.NewPostgresCollector(dsn)
	default:
		return nil, fmt.Errorf("unsupported database.type %q", dbType)
	}
}

func tick(p *pusher.Pusher, diskPath string, dbCollector collector.DBCollector, cfg *config.Config) {
	collectAndPushSystem(p, diskPath)

	if dbCollector != nil {
		collectAndPushDB(p, dbCollector, cfg.Database.DBIngestURL)
	}
}

func collectAndPushSystem(p *pusher.Pusher, diskPath string) {
	snap, err := collector.Collect(diskPath)
	if err != nil {
		log.Printf("ampligo-agent: collect failed: %v", err)
		return
	}

	if err := p.Push(snap); err != nil {
		log.Printf("ampligo-agent: push failed: %v", err)
		return
	}

	log.Printf("ampligo-agent: pushed cpu=%.1f%% mem=%.1f%% disk=%.1f%%", snap.CPUPercent, snap.MemoryPercent, snap.DiskPercent)
}

func collectAndPushDB(p *pusher.Pusher, dbCollector collector.DBCollector, dbIngestURL string) {
	snap, err := dbCollector.Collect()
	if err != nil {
		log.Printf("ampligo-agent: db collect failed: %v", err)
		return
	}

	if err := p.PushDB(dbIngestURL, snap); err != nil {
		log.Printf("ampligo-agent: db push failed: %v", err)
		return
	}

	log.Printf("ampligo-agent: pushed db connections=%d qps=%.1f", snap.ConnectionsActive, snap.QueriesPerSecond)
}
