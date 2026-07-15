package collector

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// Snapshot matches the payload shape expected by POST /api/v1/ingest/usage.
type Snapshot struct {
	Hostname        string  `json:"hostname"`
	CPUPercent      float64 `json:"cpu_percent"`
	MemoryPercent   float64 `json:"memory_percent"`
	MemoryUsedMB    uint64  `json:"memory_used_mb"`
	DiskPercent     float64 `json:"disk_percent"`
	DiskUsedGB      uint64  `json:"disk_used_gb"`
	NetworkInBytes  uint64  `json:"network_in_bytes"`
	NetworkOutBytes uint64  `json:"network_out_bytes"`
	LoadAvg1m       float64 `json:"load_avg_1m"`
	LoadAvg5m       float64 `json:"load_avg_5m"`
	LoadAvg15m      float64 `json:"load_avg_15m"`
	RecordedAt      string  `json:"recorded_at"`
}

// Collect samples the host's current CPU, memory, disk and network usage.
// NetworkInBytes/NetworkOutBytes are cumulative counters since boot, not deltas
// since the last sample - the server is expected to derive rates from consecutive readings.
func Collect(diskPath string) (Snapshot, error) {
	snap := Snapshot{
		Hostname:   hostname(),
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
	}

	cpuPercents, err := cpu.Percent(time.Second, false)
	if err != nil {
		return snap, fmt.Errorf("reading cpu percent: %w", err)
	}
	if len(cpuPercents) > 0 {
		snap.CPUPercent = cpuPercents[0]
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return snap, fmt.Errorf("reading memory: %w", err)
	}
	snap.MemoryPercent = vm.UsedPercent
	snap.MemoryUsedMB = vm.Used / 1024 / 1024

	du, err := disk.Usage(diskPath)
	if err != nil {
		return snap, fmt.Errorf("reading disk usage for %s: %w", diskPath, err)
	}
	snap.DiskPercent = du.UsedPercent
	snap.DiskUsedGB = du.Used / 1024 / 1024 / 1024

	if counters, err := net.IOCounters(false); err == nil && len(counters) > 0 {
		snap.NetworkInBytes = counters[0].BytesRecv
		snap.NetworkOutBytes = counters[0].BytesSent
	}

	if avg, err := load.Avg(); err == nil {
		snap.LoadAvg1m = avg.Load1
		snap.LoadAvg5m = avg.Load5
		snap.LoadAvg15m = avg.Load15
	}
	// load.Avg() is unsupported on some platforms (e.g. Windows); leave zeros rather than failing the whole collection.

	return snap, nil
}
