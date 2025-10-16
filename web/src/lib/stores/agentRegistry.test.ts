import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { AgentEndpoint } from '../types/messages';

interface WindowWithStorage {
	localStorage: Storage;
}

function createLocalStorageMock(): Storage {
	const store = new Map<string, string>();
	return {
		getItem: (key: string) => store.get(key) ?? null,
		setItem: (key: string, value: string) => {
			store.set(key, value);
		},
		removeItem: (key: string) => {
			store.delete(key);
		},
		clear: () => {
			store.clear();
		},
		key: (index: number) => Array.from(store.keys())[index] ?? null,
		get length() {
			return store.size;
		}
	};
}

describe('agentRegistry', () => {
	let agentRegistryModule: typeof import('./agentRegistry');
	let windowStub: WindowWithStorage;

	beforeEach(async () => {
		vi.resetModules();
		vi.doMock('$app/environment', () => ({ browser: true }));
		windowStub = {
			localStorage: createLocalStorageMock()
		};
		vi.stubGlobal('window', windowStub);
		agentRegistryModule = await import('./agentRegistry');
	});

	afterEach(() => {
		vi.resetModules();
		vi.unstubAllGlobals();
		vi.clearAllMocks();
	});

	it('removes entries and clears persistence when an agent is deleted', () => {
		const { addAgentEndpoint, removeAgentEndpoint, agentRegistry, AGENT_REGISTRY_STORAGE_KEY } =
			agentRegistryModule;

		let snapshot: AgentEndpoint[] = [];
		const unsubscribe = agentRegistry.subscribe((agents) => {
			snapshot = agents;
		});

		addAgentEndpoint('ws://example');
		expect(snapshot).toHaveLength(1);
		const agentId = snapshot[0].id;

		removeAgentEndpoint(agentId);
		expect(snapshot).toHaveLength(0);

		const stored = windowStub.localStorage.getItem(AGENT_REGISTRY_STORAGE_KEY);
		expect(stored).toBe(JSON.stringify([]));

		unsubscribe();
	});
});
