package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/niago-id/ampligo-agent/internal/collector"
	"github.com/niago-id/ampligo-agent/internal/config"
	"github.com/niago-id/ampligo-agent/internal/pusher"
)

func main() {
	configPath := flag.String("config", "/etc/ampligo/agent.yml", "path to agent.yml")
	diskPath := flag.String("disk-path", "/", "filesystem path to report disk usage for")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("ampligo-agent: %v", err)
	}

	p := pusher.New(cfg.IngestURL, cfg.APIKey)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(cfg.Interval())
	defer ticker.Stop()

	log.Printf("ampligo-agent: pushing every %s to %s", cfg.Interval(), cfg.IngestURL)
	collectAndPush(p, *diskPath)

	for {
		select {
		case <-ctx.Done():
			log.Println("ampligo-agent: shutting down")
			return
		case <-ticker.C:
			collectAndPush(p, *diskPath)
		}
	}
}

func collectAndPush(p *pusher.Pusher, diskPath string) {
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
