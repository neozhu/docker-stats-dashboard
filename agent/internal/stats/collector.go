package stats

import (
	"context"
	"encoding/json"
	"errors"
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
	watchersMu sync.Mutex
	sequence   uint64
	lastBatch  *types.ContainerStatsBatch
	lastSentAt time.Time

	samples   map[string]types.ContainerResourceSample
	watchers  map[string]context.CancelFunc
	sampleSem chan struct{}
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
		samples:      make(map[string]types.ContainerResourceSample),
		watchers:     make(map[string]context.CancelFunc),
		sampleSem:    make(chan struct{}, workerLimit),
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
	containers, err := c.client.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(),
		All:     false,
	})
	if err != nil {
		c.log.Warn("failed to list containers", slog.String("error", err.Error()))
		if cached := c.LastBatch(); cached != nil {
			c.log.Debug("serving cached batch after collection failure")
			cachedClone := *cached
			cachedClone.SentAt = time.Now().UTC()
			c.dispatchBatch(ctx, out, cachedClone)
		}
		return
	}

	active := make(map[string]docker.Container, len(containers))
	for _, cont := range containers {
		active[cont.ID] = cont
	}

	c.syncWatchers(ctx, out, active)

	if startup {
		c.log.Info("collector initialised", slog.Int("container_count", len(containers)))
	}
}

func (c *Collector) syncWatchers(ctx context.Context, out chan<- types.ContainerStatsBatch, active map[string]docker.Container) {
	c.watchersMu.Lock()
	// Start watchers for new containers
	for id, cont := range active {
		if _, ok := c.watchers[id]; ok {
			continue
		}
		watchCtx, cancel := context.WithCancel(ctx)
		c.watchers[id] = cancel
		go c.runWatcher(watchCtx, out, cont)
	}

	// Stop watchers for containers that disappeared
	for id, cancel := range c.watchers {
		if _, ok := active[id]; ok {
			continue
		}
		cancel()
		delete(c.watchers, id)
	}
	c.watchersMu.Unlock()

	if batch, removed := c.removeMissingSamples(active); removed {
		c.dispatchBatch(ctx, out, batch)
	}
}

func (c *Collector) runWatcher(ctx context.Context, out chan<- types.ContainerStatsBatch, cont docker.Container) {
	logger := c.log.With(
		slog.String("container_id", cont.ID),
		slog.String("container_name", firstName(cont.Names)),
	)

	// Send first sample immediately for low latency updates
	c.sampleContainer(ctx, out, cont, logger)

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.sampleContainer(ctx, out, cont, logger)
		}
	}
}

func (c *Collector) sampleContainer(ctx context.Context, out chan<- types.ContainerStatsBatch, cont docker.Container, logger *slog.Logger) {
	select {
	case c.sampleSem <- struct{}{}:
	case <-ctx.Done():
		return
	}
	defer func() { <-c.sampleSem }()

	stats, err := c.fetchStats(ctx, cont.ID)
	if err != nil {
		logger.Debug("failed to fetch container stats", slog.String("error", err.Error()))
		return
	}

	batch := c.upsertSample(cont, stats)
	c.dispatchBatch(ctx, out, batch)
}

func (c *Collector) upsertSample(cont docker.Container, stats docker.StatsJSON) types.ContainerStatsBatch {
	sample := convertStats(cont, stats)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.samples[sample.ID] = sample
	c.sequence++

	batch := c.snapshotLocked(time.Now().UTC())
	batch.Sequence = c.sequence
	c.lastBatch = &batch
	c.lastSentAt = batch.SentAt

	c.log.Debug("collected sample",
		slog.Uint64("sequence", batch.Sequence),
		slog.Int("containers", len(batch.Containers)),
		slog.Float64("cpu_pct", batch.AgentMetrics.CPUPct),
		slog.Uint64("mem_bytes", batch.AgentMetrics.MemBytes),
	)

	return batch
}

func (c *Collector) removeMissingSamples(active map[string]docker.Container) (types.ContainerStatsBatch, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.samples) == 0 {
		return types.ContainerStatsBatch{}, false
	}

	removed := false
	for id := range c.samples {
		if _, ok := active[id]; ok {
			continue
		}
		delete(c.samples, id)
		removed = true
	}

	if !removed {
		return types.ContainerStatsBatch{}, false
	}

	c.sequence++
	batch := c.snapshotLocked(time.Now().UTC())
	batch.Sequence = c.sequence
	c.lastBatch = &batch
	c.lastSentAt = batch.SentAt

	return batch, true
}

func (c *Collector) snapshotLocked(sentAt time.Time) types.ContainerStatsBatch {
	containers := make([]types.ContainerResourceSample, 0, len(c.samples))
	var totalCPU float64
	var totalMem uint64

	for _, sample := range c.samples {
		containers = append(containers, sample)
		totalCPU += sample.CPUPct
		totalMem += sample.MemBytes
	}

	return types.ContainerStatsBatch{
		Type:       "container_stats_batch",
		AgentID:    c.agentID,
		AgentLabel: c.agentLabel,
		SentAt:     sentAt,
		Containers: containers,
		AgentMetrics: types.AgentMetricsSummary{
			CPUPct:   clamp(totalCPU, 0, 100),
			MemBytes: totalMem,
		},
	}
}

func (c *Collector) dispatchBatch(ctx context.Context, out chan<- types.ContainerStatsBatch, batch types.ContainerStatsBatch) {
	select {
	case out <- batch:
	case <-ctx.Done():
	default:
		// If the downstream consumer is slow, drop the newest update to keep the pipeline non-blocking
		c.log.Warn("dropping stats batch due to slow consumer",
			slog.Int("containers", len(batch.Containers)),
			slog.Uint64("sequence", batch.Sequence),
		)
	}
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
