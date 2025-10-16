package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"golang.org/x/sync/errgroup"

	"github.com/your-org/docker-stats-dashboard/agent/internal/config"
	"github.com/your-org/docker-stats-dashboard/agent/internal/logging"
	"github.com/your-org/docker-stats-dashboard/agent/internal/stats"
	"github.com/your-org/docker-stats-dashboard/agent/internal/stream"
	"github.com/your-org/docker-stats-dashboard/agent/internal/transport"
	"github.com/your-org/docker-stats-dashboard/agent/internal/types"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Fatalf("agent exited: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	hostName, err := os.Hostname()
	if err != nil || hostName == "" {
		hostName = "unknown-host"
	}

	agentLabel := cfg.HostLabel
	if agentLabel == "" {
		agentLabel = hostName
	}

	cli, err := client.NewClientWithOpts(
		client.WithHost(cfg.DockerEndpoint),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer cli.Close()

	collector := stats.NewCollector(cli, logger.With(slog.String("component", "collector")), cfg.PollInterval, hostName, agentLabel, cfg.WorkerLimit)
	hub := stream.NewHub(logger.With(slog.String("component", "hub")))
	server := transport.NewServer(logger.With(slog.String("component", "http")), cfg.ListenAddr, hub)

	statsCh := make(chan types.ContainerStatsBatch, 64)
	startedAt := time.Now()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		hub.Run(ctx)
		return nil
	})

	g.Go(func() error {
		collector.Collect(ctx, statsCh)
		return nil
	})

	g.Go(func() error {
		return server.Run(ctx)
	})

	g.Go(func() error {
		return dispatchLoop(ctx, logger, hub, statsCh, startedAt, hostName, agentLabel)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

func dispatchLoop(
	ctx context.Context,
	logger *slog.Logger,
	hub *stream.Hub,
	statsCh <-chan types.ContainerStatsBatch,
	startedAt time.Time,
	agentID string,
	agentLabel string,
) error {
	statusTicker := time.NewTicker(30 * time.Second)
	defer statusTicker.Stop()

	sendStatus := func() {
		uptime := uint64(time.Since(startedAt).Seconds())
		status := types.AgentStatusMessage{
			Type:       "agent_status",
			AgentID:    agentID,
			AgentLabel: agentLabel,
			SentAt:     time.Now().UTC(),
			UptimeSecs: uptime,
			Version:    version,
			Features:   []string{"container_stats"},
		}
		payload, err := json.Marshal(status)
		if err != nil {
			logger.Warn("failed to marshal agent status", slog.String("error", err.Error()))
			return
		}
		logger.Debug("dispatching agent status",
			slog.Time("sent_at", status.SentAt),
			slog.Uint64("uptime_secs", status.UptimeSecs),
		)
		hub.Broadcast(payload)
	}

	sendStatus()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batch := <-statsCh:
			payload, err := json.Marshal(batch)
			if err != nil {
				logger.Warn("failed to marshal stats batch", slog.String("error", err.Error()))
				continue
			}
			logger.Debug("dispatching stats batch",
				slog.Uint64("sequence", batch.Sequence),
				slog.Time("sent_at", batch.SentAt),
				slog.Int("containers", len(batch.Containers)),
			)
			hub.Broadcast(payload)
		case <-statusTicker.C:
			sendStatus()
		}
	}
}
