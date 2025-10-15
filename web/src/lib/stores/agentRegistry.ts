import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import type { AgentConnectionState, AgentEndpoint } from '$lib/types/messages';

const STORAGE_KEY = 'docker-stats-dashboard::agent-endpoints';

function readFromStorage(): AgentEndpoint[] {
	if (!browser) return [];

	try {
		const raw = window.localStorage.getItem(STORAGE_KEY);
		if (!raw) return [];
		const parsed = JSON.parse(raw) as AgentEndpoint[];
		if (!Array.isArray(parsed)) return [];
		return parsed.map((agent, index) => ({
			id: agent.id ?? generateID(),
			url: agent.url,
			label: agent.label ?? `Agent ${index + 1}`,
			status: agent.status ?? 'placeholder',
			lastSeenAt: agent.lastSeenAt ?? null,
			createdAt: agent.createdAt ?? new Date().toISOString(),
			notes: agent.notes
		}));
	} catch {
		return [];
	}
}

const registry = writable<AgentEndpoint[]>(readFromStorage());

if (browser) {
	registry.subscribe((agents) => {
		window.localStorage.setItem(STORAGE_KEY, JSON.stringify(agents));
	});
}

function nextLabel(existing: AgentEndpoint[]): string {
	const base = 'Agent';
	let counter = existing.length + 1;
	let label = `${base} ${counter}`;
	const labels = new Set(existing.map((agent) => agent.label));

	while (labels.has(label)) {
		counter += 1;
		label = `${base} ${counter}`;
	}

	return label;
}

export function addAgentEndpoint(url: string, label?: string): void {
	const normalized = url.trim();
	if (!normalized) return;

	registry.update((agents) => {
		if (agents.some((agent) => agent.url === normalized)) {
			return agents;
		}

		const id = generateID();
		const createdAt = new Date().toISOString();

		return [
			...agents,
			{
				id,
				url: normalized,
				label: label?.trim() || nextLabel(agents),
				status: 'placeholder',
				lastSeenAt: null,
				createdAt
			}
		];
	});
}

export function removeAgentEndpoint(id: string): void {
	registry.update((agents) => agents.filter((agent) => agent.id !== id));
}

export function renameAgentEndpoint(id: string, label: string): void {
	registry.update((agents) =>
		agents.map((agent) =>
			agent.id === id
				? {
						...agent,
						label: label.trim() || agent.label
					}
				: agent
		)
	);
}

export function setAgentStatus(
	id: string,
	status: AgentConnectionState,
	lastSeenAt: string | null = new Date().toISOString()
): void {
	registry.update((agents) =>
		agents.map((agent) =>
			agent.id === id
				? {
						...agent,
						status,
						lastSeenAt
					}
				: agent
		)
	);
}

export function clearAgentRegistry(): void {
	registry.set([]);
}

export const agentRegistry = {
	subscribe: registry.subscribe
};

export { STORAGE_KEY as AGENT_REGISTRY_STORAGE_KEY };

function generateID(): string {
	if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
		return crypto.randomUUID();
	}

	const random = Math.random().toString(16).slice(2, 10);
	return `agent-${Date.now().toString(16)}-${random}`;
}
