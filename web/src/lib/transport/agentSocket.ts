import type { DashboardMessage, ContainerStatsBatch, AgentStatusMessage } from '$lib/types/messages';

export interface AgentSocketHandlers {
	onConnect: () => void;
	onDisconnect: (reason: string) => void;
	onStats: (payload: ContainerStatsBatch) => void;
	onStatus: (payload: AgentStatusMessage) => void;
}

export interface AgentSocket {
	close: () => void;
}

export function connectAgentSocket(endpoint: string, handlers: AgentSocketHandlers): AgentSocket {
	const socket = new WebSocket(endpoint);

	socket.addEventListener('open', () => {
		handlers.onConnect();
	});

	socket.addEventListener('message', (event) => {
		try {
			const payload = JSON.parse(event.data) as DashboardMessage;
			if (!payload || typeof payload !== 'object' || !('type' in payload)) {
				return;
			}

			if (payload.type === 'container_stats_batch') {
				handlers.onStats(payload as ContainerStatsBatch);
			} else {
				handlers.onStatus(payload as AgentStatusMessage);
			}
		} catch (error) {
			console.error('Failed to parse WebSocket message', error);
		}
	});

	socket.addEventListener('error', (event) => {
		console.error('Agent WebSocket error', event);
		handlers.onDisconnect('error');
		socket.close();
	});

	socket.addEventListener('close', (event) => {
		handlers.onDisconnect(event.reason || 'closed');
	});

	return {
		close: () => {
			if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
				socket.close();
			}
		}
	};
}
