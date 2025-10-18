package types

import "time"

type AgentMetricsSummary struct {
	CPUPct   float64 `json:"cpu_pct"`
	MemBytes uint64  `json:"mem_bytes"`
}

type ContainerResourceSample struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	CPUPct        float64 `json:"cpu_pct"`
	MemBytes      uint64  `json:"mem_bytes"`
	MemLimitBytes uint64  `json:"mem_limit_bytes"`
	NetIOBytes    uint64  `json:"net_io_bytes"`
}

type ContainerStatsBatch struct {
	Type         string                    `json:"type"`
	AgentID      string                    `json:"agent_id"`
	AgentLabel   string                    `json:"agent_label,omitempty"`
	SentAt       time.Time                 `json:"sent_at"`
	Sequence     uint64                    `json:"sequence"`
	Containers   []ContainerResourceSample `json:"containers"`
	AgentMetrics AgentMetricsSummary       `json:"agent_metrics"`
}

type AgentStatusMessage struct {
	Type       string    `json:"type"`
	AgentID    string    `json:"agent_id"`
	AgentLabel string    `json:"agent_label,omitempty"`
	SentAt     time.Time `json:"sent_at"`
	UptimeSecs uint64    `json:"uptime_secs"`
	Version    string    `json:"version,omitempty"`
	Features   []string  `json:"features,omitempty"`
}

type DashboardMessage interface{}
