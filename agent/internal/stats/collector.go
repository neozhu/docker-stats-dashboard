package stats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/your-org/docker-stats-dashboard/agent/internal/types"
)

type Collector struct {
	client       *client.Client
	log          *slog.Logger
	pollInterval time.Duration
	agentID      string
	agentLabel   string
	workerLimit  int

	mu         sync.RWMutex
	sequence   uint64
	lastBatch  *types.ContainerStatsBatch
	lastSentAt time.Time
}

func NewCollector(cli *client.Client, logger *slog.Logger, pollInterval time.Duration, agentID, agentLabel string, workerLimit int) *Collector {
	if workerLimit <= 0 {
		workerLimit = 1
	}
	return &Collector{
		client:       cli,
		log:          logger,
		pollInterval: pollInterval,
		agentID:      agentID,
		agentLabel:   agentLabel,
		workerLimit:  workerLimit,
	}
}

func (c *Collector) LastBatch() *types.ContainerStatsBatch {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastBatch == nil {
		return nil
	}
	clone := *c.lastBatch
	return &clone
}

func (c *Collector) Collect(ctx context.Context, out chan<- types.ContainerStatsBatch) {
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	// Collect immediately at startup
	c.collectOnce(ctx, out, true)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collectOnce(ctx, out, false)
		}
	}
}

func (c *Collector) collectOnce(ctx context.Context, out chan<- types.ContainerStatsBatch, startup bool) {
	batch, err := c.buildBatch(ctx)
	if err != nil {
		c.log.Warn("failed to collect container stats", slog.String("error", err.Error()))
		// Use cached batch if available
		if cached := c.LastBatch(); cached != nil {
			c.log.Debug("serving cached batch after collection failure")
			cachedClone := *cached
			cachedClone.SentAt = time.Now().UTC()
			select {
			case out <- cachedClone:
			case <-ctx.Done():
			}
		}
		return
	}

	if startup {
		c.log.Info("collector initialised", slog.Int("container_count", len(batch.Containers)))
	}

	select {
	case out <- batch:
	case <-ctx.Done():
		return
	}
}

func (c *Collector) buildBatch(ctx context.Context) (types.ContainerStatsBatch, error) {
	containers, err := c.client.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(),
		All:     false,
	})
	if err != nil {
		return types.ContainerStatsBatch{}, fmt.Errorf("list containers: %w", err)
	}

	batch := types.ContainerStatsBatch{
		Type:       "container_stats_batch",
		AgentID:    c.agentID,
		AgentLabel: c.agentLabel,
		SentAt:     time.Now().UTC(),
	}

	samples := c.collectSamplesConcurrently(ctx, containers)

	var (
		totalCPU float64
		totalMem uint64
	)

	for _, sample := range samples {
		batch.Containers = append(batch.Containers, sample)
		totalCPU += sample.CPUPct
		totalMem += sample.MemBytes
	}

	batch.AgentMetrics = types.AgentMetricsSummary{
		CPUPct:   clamp(totalCPU, 0, 100),
		MemBytes: totalMem,
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.sequence++
	batch.Sequence = c.sequence
	c.lastBatch = &batch
	c.lastSentAt = batch.SentAt

	c.log.Debug("collected batch",
		slog.Uint64("sequence", batch.Sequence),
		slog.Int("containers", len(batch.Containers)),
		slog.Float64("cpu_pct", batch.AgentMetrics.CPUPct),
		slog.Uint64("mem_bytes", batch.AgentMetrics.MemBytes),
	)

	return batch, nil
}

func (c *Collector) collectSamplesConcurrently(ctx context.Context, containers []docker.Container) []types.ContainerResourceSample {
	if len(containers) == 0 {
		return nil
	}

	type result struct {
		sample types.ContainerResourceSample
		ok     bool
	}

	results := make(chan result, len(containers))
	var wg sync.WaitGroup

	workerLimit := c.workerLimit
	if workerLimit <= 0 {
		workerLimit = 4
	}
	if workerLimit > len(containers) {
		workerLimit = len(containers)
	}

	sem := make(chan struct{}, workerLimit)

	for _, cont := range containers {
		wg.Add(1)
		go func(cont docker.Container) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			stats, err := c.fetchStats(ctx, cont.ID)
			if err != nil {
				c.log.Debug("failed to fetch container stats", slog.String("container_id", cont.ID), slog.String("error", err.Error()))
				return
			}

			sample := convertStats(cont, stats)
			select {
			case results <- result{sample: sample, ok: true}:
			case <-ctx.Done():
			}
		}(cont)
	}

	wg.Wait()
	close(results)

	var samples []types.ContainerResourceSample
	for res := range results {
		if res.ok {
			samples = append(samples, res.sample)
		}
	}

	return samples
}

func (c *Collector) fetchStats(ctx context.Context, containerID string) (docker.StatsJSON, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.ContainerStats(requestCtx, containerID, false)
	if err != nil {
		return docker.StatsJSON{}, err
	}
	defer resp.Body.Close()

	if resp.Body == nil {
		return docker.StatsJSON{}, errors.New("nil stats body")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return docker.StatsJSON{}, err
	}

	var stats docker.StatsJSON
	if err := json.Unmarshal(data, &stats); err != nil {
		return docker.StatsJSON{}, err
	}

	return stats, nil
}

func convertStats(cont docker.Container, stats docker.StatsJSON) types.ContainerResourceSample {
	cpuPct := calculateCPUPercent(stats)
	memUsage := stats.MemoryStats.Usage
	memLimit := stats.MemoryStats.Limit
	if memLimit == 0 {
		memLimit = 1
	}

	created := time.Unix(cont.Created, 0)
	if cont.Created == 0 {
		created = time.Now()
	}
	uptime := uint64(time.Since(created).Seconds())

	return types.ContainerResourceSample{
		ID:            cont.ID,
		Name:          firstName(cont.Names),
		CPUPct:        cpuPct,
		MemBytes:      uint64(memUsage),
		MemLimitBytes: uint64(memLimit),
		UptimeSecs:    uptime,
	}
}

func firstName(names []string) string {
	for _, name := range names {
		if name == "" {
			continue
		}
		return strings.TrimPrefix(name, "/")
	}
	return "unknown"
}

func calculateCPUPercent(stats docker.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if systemDelta <= 0 || cpuDelta <= 0 {
		return 0
	}

	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		if onlineCPUs == 0 {
			onlineCPUs = 1
		}
	}

	cpuPercent := (cpuDelta / systemDelta) * onlineCPUs * 100
	return clamp(cpuPercent, 0, 100)
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
