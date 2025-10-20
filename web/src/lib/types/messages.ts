export type AgentConnectionState = 'placeholder' | 'connecting' | 'connected' | 'error';

export interface AgentEndpoint {
	id: string;
	url: string;
	label: string;
	status: AgentConnectionState;
	lastSeenAt: string | null;
	createdAt: string;
	notes?: string;
}

export interface AgentMetricsSummary {
        cpu_pct: number;
        mem_bytes: number;
}

export interface AgentCpuSample {
        at: string;
        cpu_pct: number;
}

export interface ContainerResourceSample {
        id: string;
        name: string;
        cpu_pct: number;
        mem_bytes: number;
	mem_limit_bytes: number;
	net_io_bytes: number;
}

export interface ContainerStatsBatch {
	type: 'container_stats_batch';
	agent_id: string;
	agent_label?: string;
	sent_at: string;
	sequence: number;
	containers: ContainerResourceSample[];
	agent_metrics: AgentMetricsSummary;
}

export interface AgentStatusMessage {
	type: 'agent_status';
	agent_id: string;
	agent_label?: string;
	sent_at: string;
	uptime_secs: number;
	version?: string;
	features?: string[];
}

export type DashboardMessage = ContainerStatsBatch | AgentStatusMessage;
