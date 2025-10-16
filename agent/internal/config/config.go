package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultDockerEndpoint = "unix:///var/run/docker.sock"
	defaultListenAddr     = ":8080"
	defaultHostLabel      = ""
	defaultPollInterval   = 500 * time.Millisecond
	// defaultFetchTimeout limits per-container stats fetch to avoid a single slow container
	// extending the whole batch duration. Tuned lower than the previous hardcoded 5s to
	// favour overall refresh smoothness.
	defaultFetchTimeout   = 2 * time.Second
	defaultLogLevel       = "info"
	defaultWorkerLimit    = 16
)

type Config struct {
	DockerEndpoint string
	ListenAddr     string
	HostLabel      string
	PollInterval   time.Duration
	FetchTimeout   time.Duration
	LogLevel       string
	WorkerLimit    int
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func parseDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return value, nil
}

func Load() (Config, error) {
	cfg := Config{}

	pollInterval := defaultPollInterval
	if duration, err := parseDurationEnv("AGENT_POLL_INTERVAL", defaultPollInterval); err != nil {
		return Config{}, err
	} else {
		pollInterval = duration
	}

	fetchTimeout := defaultFetchTimeout
	if duration, err := parseDurationEnv("AGENT_FETCH_TIMEOUT", defaultFetchTimeout); err != nil {
		return Config{}, err
	} else {
		fetchTimeout = duration
	}

	workerLimit := defaultWorkerLimit
	if raw := envOrDefault("AGENT_MAX_WORKERS", ""); raw != "" {
		value, err := parseWorkerLimit(raw)
		if err != nil {
			return Config{}, err
		}
		workerLimit = value
	}

	defaults := Config{
		DockerEndpoint: envOrDefault("AGENT_DOCKER_ENDPOINT", defaultDockerEndpoint),
		ListenAddr:     envOrDefault("AGENT_LISTEN_ADDR", defaultListenAddr),
		HostLabel:      envOrDefault("AGENT_HOST_LABEL", defaultHostLabel),
		PollInterval:   pollInterval,
		FetchTimeout:   fetchTimeout,
		LogLevel:       strings.ToLower(envOrDefault("AGENT_LOG_LEVEL", defaultLogLevel)),
		WorkerLimit:    workerLimit,
	}

	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.StringVar(&cfg.DockerEndpoint, "docker-endpoint", defaults.DockerEndpoint, "Docker engine endpoint (unix socket or TCP URL)")
	flagSet.StringVar(&cfg.ListenAddr, "listen", defaults.ListenAddr, "HTTP listen address for WebSocket server")
	flagSet.StringVar(&cfg.HostLabel, "host-label", defaults.HostLabel, "Human readable label for this agent")
	flagSet.DurationVar(&cfg.PollInterval, "poll-interval", defaults.PollInterval, "Interval for sampling container stats")
	flagSet.DurationVar(&cfg.FetchTimeout, "fetch-timeout", defaults.FetchTimeout, "Per-container stats fetch timeout")
	flagSet.StringVar(&cfg.LogLevel, "log-level", defaults.LogLevel, "Log level (debug, info, warn, error)")
	flagSet.IntVar(&cfg.WorkerLimit, "max-workers", defaults.WorkerLimit, "Maximum number of concurrent stats workers")

	if err := flagSet.Parse(filterArgs(os.Args[1:])); err != nil {
		return Config{}, err
	}

	cfg.LogLevel = strings.ToLower(strings.TrimSpace(cfg.LogLevel))

	if cfg.PollInterval <= 0 {
		return Config{}, fmt.Errorf("poll interval must be positive")
	}
	if cfg.FetchTimeout <= 0 {
		return Config{}, fmt.Errorf("fetch timeout must be positive")
	}
	if cfg.WorkerLimit <= 0 {
		cfg.WorkerLimit = 1
	}

	return cfg, nil
}

func filterArgs(args []string) []string {
	allowed := map[string]bool{
		"--docker-endpoint": true,
		"--listen":          true,
		"--host-label":      true,
		"--poll-interval":   true,
		"--fetch-timeout":   true,
		"--log-level":       true,
		"--max-workers":     true,
	}

	var filtered []string
	skipNext := false

	for i := 0; i < len(args); i++ {
		if skipNext {
			skipNext = false
			continue
		}

		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg, "=", 2)
			if allowed[parts[0]] {
				filtered = append(filtered, arg)
				if len(parts) == 1 {
					if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
						filtered = append(filtered, args[i+1])
						skipNext = true
					}
				}
			}
		}
	}

	return filtered
}

func parseWorkerLimit(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultWorkerLimit, nil
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid max worker value %q: %w", value, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("max workers must be positive")
	}
	return parsed, nil
}
