import type { ContainerStatsBatch } from '$lib/types/messages';

export interface MockAgentStream {
	subscribe: (listener: (batch: ContainerStatsBatch) => void) => () => void;
	stop: () => void;
}

interface MockAgentStreamOptions {
	agentId: string;
	agentLabel?: string;
	containerCount?: number;
	intervalMs?: number;
}

const containerCatalog = [
	'nginx',
	'redis',
	'worker',
	'postgres',
	'prometheus',
	'fluentd',
	'jaeger',
	'feature-flags',
	'identity',
	'cdn-cache'
];

const randomBetween = (min: number, max: number): number => {
	return Math.random() * (max - min) + min;
};

function createBatch(options: {
	agentId: string;
	agentLabel?: string;
	containerCount: number;
	sequence: number;
}): ContainerStatsBatch {
	const { agentId, agentLabel, containerCount, sequence } = options;
	const now = new Date().toISOString();
	const containers = Array.from({ length: containerCount }).map((_, index) => {
		const name = containerCatalog[(index + sequence) % containerCatalog.length] ?? `service-${index + 1}`;
		const cpu = Number(randomBetween(2, 75).toFixed(1));
		const memLimit = 512 * 1024 * 1024;
		const memUsage = Number(randomBetween(64, memLimit / (1024 * 1024)).toFixed(0)) * 1024 * 1024;
		const uptime = Math.floor(randomBetween(120, 3600 * 12));

		return {
			id: `${agentId}-${name}`,
			name,
			cpu_pct: cpu,
			mem_bytes: memUsage,
			mem_limit_bytes: memLimit,
			uptime_secs: uptime
		};
	});

	const totalCpu = containers.reduce((acc, container) => acc + container.cpu_pct, 0);
	const totalMem = containers.reduce((acc, container) => acc + container.mem_bytes, 0);

	return {
		type: 'container_stats_batch',
		agent_id: agentId,
		agent_label: agentLabel,
		sent_at: now,
		sequence,
		containers,
		agent_metrics: {
			cpu_pct: Number(Math.min(totalCpu / Math.max(containerCount, 1), 100).toFixed(1)),
			mem_bytes: totalMem
		}
	};
}

export function createMockAgentStream(options: MockAgentStreamOptions): MockAgentStream {
	const { agentId, agentLabel, intervalMs = 1500 } = options;
	const containerCount = Math.max(3, Math.min(options.containerCount ?? 5, containerCatalog.length));
	const listeners = new Set<(batch: ContainerStatsBatch) => void>();
	let timer: ReturnType<typeof setInterval> | null = null;
	let sequence = 1;

	const emit = () => {
		const batch = createBatch({ agentId, agentLabel, containerCount, sequence });
		sequence += 1;
		listeners.forEach((listener) => listener(batch));
	};

	const start = () => {
		if (timer !== null || typeof window === 'undefined') return;
		emit();
		timer = window.setInterval(emit, intervalMs);
	};

	const stop = () => {
		if (timer !== null) {
			clearInterval(timer);
			timer = null;
		}
	};

	return {
		subscribe: (listener: (batch: ContainerStatsBatch) => void) => {
			listeners.add(listener);
			if (typeof window !== 'undefined') {
				start();
			} else {
				// SSR fallback emits once synchronously
				listener(createBatch({ agentId, agentLabel, containerCount, sequence }));
				sequence += 1;
			}

			return () => {
				listeners.delete(listener);
				if (listeners.size === 0) {
					stop();
				}
			};
		},
		stop
	};
}
