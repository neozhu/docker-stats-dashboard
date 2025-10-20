import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import type { AgentCpuSample, ContainerStatsBatch } from '$lib/types/messages';

export interface AgentMeta {
  id: string;
  label: string;
  url: string;
  status: 'connecting' | 'connected' | 'error' | 'closed';
  lastSeenAt: string | null;
}

export const agents = writable<AgentMeta[]>([]);
export const latestBatches = writable<Map<string, ContainerStatsBatch>>(new Map());
export const agentCpuHistory = writable<Map<string, AgentCpuSample[]>>(new Map());

let es: EventSource | null = null;

export function startSSE() {
  if (!browser) return;
  if (es) return;
  es = new EventSource('/stream/agents');

  es.onmessage = (ev) => {
    try {
      const data = JSON.parse(ev.data);
      switch (data.type) {
        case 'agent_list': {
          interface IncomingAgent { id: string; label: string; url: string }
          const list: AgentMeta[] = (data.agents as IncomingAgent[]).map((a) => ({
            id: a.id,
            label: a.label,
            url: a.url,
            status: 'connecting',
            lastSeenAt: null
          }));
          agents.set(list);
          agentCpuHistory.update((prev) => {
            const next = new Map<string, AgentCpuSample[]>();
            for (const agent of list) {
              const existing = prev.get(agent.id);
              next.set(agent.id, existing ? [...existing] : []);
            }
            return next;
          });
          break;
        }
        case 'agent_status': {
          agents.update((arr) => arr.map((a) => a.id === data.agent_id ? { ...a, status: data.status, lastSeenAt: data.at } : a));
          break;
        }
        case 'container_stats_batch': {
          const payload = data.payload as ContainerStatsBatch;
          // ensure agent id + label exist
          payload.agent_id = data.agent_id || payload.agent_id;
          agents.update((arr) => arr.map((a) => a.id === payload.agent_id ? { ...a, status: 'connected', lastSeenAt: payload.sent_at || data.received_at } : a));
          latestBatches.update((m) => {
            const next = new Map(m);
            next.set(payload.agent_id, payload);
            return next;
          });
          const history = Array.isArray(data.history)
            ? (data.history as AgentCpuSample[]).map((sample) => ({
                at: sample.at,
                cpu_pct: sample.cpu_pct
              }))
            : [];
          agentCpuHistory.update((m) => {
            const next = new Map(m);
            next.set(payload.agent_id, history);
            return next;
          });
          break;
        }
      }
    } catch (e) {
      console.error('Bad SSE message', e);
    }
  };

  es.onerror = () => {
    console.warn('SSE error');
  };
}

export function stopSSE() {
  es?.close();
  es = null;
}
