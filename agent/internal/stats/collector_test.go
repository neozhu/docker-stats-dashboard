package stats

import (
	"testing"
	"time"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

func TestConvertStats(t *testing.T) {
	now := time.Now().Add(-2 * time.Minute)
	cont := docker.Container{
		ID:      "abc123",
		Names:   []string{"/web"},
		Created: now.Unix(),
	}

	stats := docker.StatsJSON{
		Stats: container.Stats{
			CPUStats: container.CPUStats{
				CPUUsage: container.CPUUsage{
					TotalUsage:  200000000,
					PercpuUsage: []uint64{100000000, 100000000},
				},
				SystemUsage: 400000000,
				OnlineCPUs:  2,
			},
			PreCPUStats: container.CPUStats{
				CPUUsage: container.CPUUsage{
					TotalUsage: 100000000,
				},
				SystemUsage: 200000000,
			},
			MemoryStats: container.MemoryStats{
				Usage: 256 * 1024 * 1024,
				Limit: 512 * 1024 * 1024,
			},
		},
		Networks: map[string]docker.NetworkStats{
			"eth0": {
				RxBytes: 5120,
				TxBytes: 2048,
			},
		},
	}

	sample := convertStats(cont, stats)

	if sample.ID != cont.ID {
		t.Fatalf("expected id %s, got %s", cont.ID, sample.ID)
	}

	if sample.Name != "web" {
		t.Fatalf("expected name web, got %s", sample.Name)
	}

	if sample.CPUPct <= 0 {
		t.Fatalf("expected CPU percent > 0, got %f", sample.CPUPct)
	}

	if sample.MemBytes != 256*1024*1024 {
		t.Fatalf("unexpected memory bytes: %d", sample.MemBytes)
	}

	if sample.MemLimitBytes != 512*1024*1024 {
		t.Fatalf("unexpected memory limit bytes: %d", sample.MemLimitBytes)
	}

	expectedNetIO := uint64(5120 + 2048)
	if sample.NetIOBytes != expectedNetIO {
		t.Fatalf("unexpected net IO bytes: got %d, want %d", sample.NetIOBytes, expectedNetIO)
	}
}

func TestCalculateCPUPercentClamp(t *testing.T) {
	stats := docker.StatsJSON{
		Stats: container.Stats{
			CPUStats: container.CPUStats{
				CPUUsage: container.CPUUsage{
					TotalUsage:  500,
					PercpuUsage: []uint64{250, 250},
				},
				SystemUsage: 1000,
				OnlineCPUs:  2,
			},
			PreCPUStats: container.CPUStats{
				CPUUsage: container.CPUUsage{
					TotalUsage: 100,
				},
				SystemUsage: 200,
			},
		},
	}

	value := calculateCPUPercent(stats)
	if value == 0 {
		t.Fatalf("expected non-zero CPU percent")
	}
	if value > 100 {
		t.Fatalf("expected clamp <= 100, got %f", value)
	}
}

func TestFirstNameFallback(t *testing.T) {
	name := firstName([]string{"", "/api"})
	if name != "api" {
		t.Fatalf("expected api, got %s", name)
	}

	if got := firstName([]string{}); got != "unknown" {
		t.Fatalf("expected unknown fallback, got %s", got)
	}
}
