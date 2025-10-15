package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("AGENT_DOCKER_ENDPOINT", "tcp://remote:2375")
	t.Setenv("AGENT_LISTEN_ADDR", ":9000")
	t.Setenv("AGENT_HOST_LABEL", "staging-a")
	t.Setenv("AGENT_LOG_LEVEL", "debug")
	t.Setenv("AGENT_POLL_INTERVAL", "2s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.DockerEndpoint != "tcp://remote:2375" {
		t.Fatalf("unexpected docker endpoint: %s", cfg.DockerEndpoint)
	}
	if cfg.ListenAddr != ":9000" {
		t.Fatalf("unexpected listen addr: %s", cfg.ListenAddr)
	}
	if cfg.HostLabel != "staging-a" {
		t.Fatalf("unexpected host label: %s", cfg.HostLabel)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("unexpected log level: %s", cfg.LogLevel)
	}
	if cfg.PollInterval != 2*time.Second {
		t.Fatalf("unexpected poll interval: %s", cfg.PollInterval)
	}
}

func TestLoadInvalidInterval(t *testing.T) {
	t.Setenv("AGENT_POLL_INTERVAL", "not-a-duration")
	defer os.Unsetenv("AGENT_POLL_INTERVAL")

	if _, err := Load(); err == nil {
		t.Fatalf("expected error for invalid duration")
	}
}
