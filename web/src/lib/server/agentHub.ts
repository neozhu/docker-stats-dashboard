import WebSocket from 'ws';
import { EventEmitter } from 'node:events';
import { env as dynamicEnv } from '$env/dynamic/private';
import type { AgentCpuSample, ContainerStatsBatch } from '$lib/types/messages';

export interface RawAgentConfig {
  id?: string;
  label?: string;
  url: string; // ws://host:port/ws
}

export interface AgentConfigResolved {
  id: string;
  label: string;
  url: string;
}

const HISTORY_WINDOW_MS = 30 * 60 * 1000;
const MAX_HISTORY_POINTS = 360;

type AgentStatusEvent = {
  type: 'agent_status';
  agent_id: string;
  status: 'connecting' | 'connected' | 'error' | 'closed';
  label: string;
  at: string;
};

type ContainerStatsEventBase = {
  type: 'container_stats_batch';
  agent_id: string;
  label: string;
  payload: unknown;
  received_at: string;
};

type AgentConnectionEvent = AgentStatusEvent | ContainerStatsEventBase;

export type HubEvent = AgentStatusEvent | (ContainerStatsEventBase & { history: AgentCpuSample[] });

function isContainerStatsBatch(payload: unknown): payload is ContainerStatsBatch {
  if (typeof payload !== 'object' || payload === null) return false;
  if (!('agent_metrics' in payload)) return false;
  const metrics = (payload as { agent_metrics?: unknown }).agent_metrics;
  if (typeof metrics !== 'object' || metrics === null) return false;
  return 'cpu_pct' in metrics;
}

class AgentConnection {
  private ws: WebSocket | null = null;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private stopped = false;

  constructor(private cfg: AgentConfigResolved, private emit: (e: AgentConnectionEvent) => void) {
    this.connect();
  }

  private connect() {
    if (this.stopped) return;
    //console.log('[AgentHub] connecting ->', this.cfg.id, this.cfg.url);
    this.emit({ type: 'agent_status', agent_id: this.cfg.id, status: 'connecting', label: this.cfg.label, at: new Date().toISOString() });
    this.ws = new WebSocket(this.cfg.url);

    this.ws.on('open', () => {
      //console.log('[AgentHub] connected  ->', this.cfg.id);
      this.emit({ type: 'agent_status', agent_id: this.cfg.id, status: 'connected', label: this.cfg.label, at: new Date().toISOString() });
    });

    this.ws.on('message', (data) => {
      try {
        const parsed = JSON.parse(data.toString());
        const batchType = parsed?.type ?? 'unknown';
        //console.log('[AgentHub] message   <-', this.cfg.id, batchType);
        if (batchType === 'container_stats_batch') {
          this.emit({
            type: 'container_stats_batch',
            agent_id: this.cfg.id,
            label: this.cfg.label,
            payload: parsed,
            received_at: new Date().toISOString()
          });
        } else if (batchType === 'agent_status') {
          // translate agent_status coming from agent into hub-level agent_status event
          this.emit({
            type: 'agent_status',
            agent_id: this.cfg.id,
            status: 'connected', // treat agent self-status as a liveness signal
            label: this.cfg.label,
            at: new Date().toISOString()
          });
        }
      } catch (err) {
        console.error('Failed to parse agent message', this.cfg.id, err);
      }
    });

    this.ws.on('close', () => {
      //console.log('[AgentHub] closed     x', this.cfg.id);
      this.emit({ type: 'agent_status', agent_id: this.cfg.id, status: 'closed', label: this.cfg.label, at: new Date().toISOString() });
      this.scheduleReconnect();
    });

    this.ws.on('error', (err) => {
      console.log('[AgentHub] error      !', this.cfg.id, err?.message);
      this.emit({ type: 'agent_status', agent_id: this.cfg.id, status: 'error', label: this.cfg.label, at: new Date().toISOString() });
      console.error('Agent socket error', this.cfg.id, err);
      this.ws?.close();
    });
  }

  private scheduleReconnect() {
    if (this.stopped) return;
    if (this.reconnectTimer) return;
    //console.log('[AgentHub] reconnect in 3s', this.cfg.id);
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, 3000);
  }

  stop() {
    this.stopped = true;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
  }
}

export class AgentHub extends EventEmitter {
  private agents = new Map<string, AgentConnection>();
  private configs: AgentConfigResolved[] = [];
  private cpuHistory = new Map<string, AgentCpuSample[]>();

  constructor(configs: AgentConfigResolved[]) {
    super();
    this.configs = configs;
    //console.log('[AgentHub] bootstrap with', configs.length, 'agents');
    this.bootstrap();
  }

  private bootstrap() {
    for (const cfg of this.configs) {
      const conn = new AgentConnection(cfg, (e) => this.handleConnectionEvent(e));
      this.agents.set(cfg.id, conn);
    }
  }

  private handleConnectionEvent(event: AgentConnectionEvent) {
    if (event.type === 'container_stats_batch') {
      const history = this.recordCpuHistory(event);
      const enriched: HubEvent = { ...event, history };
      this.emit('event', enriched);
      return;
    }

    this.emit('event', event);
  }

  private recordCpuHistory(event: ContainerStatsEventBase): AgentCpuSample[] {
    const nextHistory = [...(this.cpuHistory.get(event.agent_id) ?? [])];

    if (isContainerStatsBatch(event.payload)) {
      const payload = event.payload as ContainerStatsBatch;
      const cpu = Number(payload.agent_metrics?.cpu_pct);
      if (Number.isFinite(cpu)) {
        const rawTimestamp = typeof payload.sent_at === 'string' ? payload.sent_at : event.received_at;
        const ts = new Date(rawTimestamp);
        const iso = Number.isNaN(ts.getTime()) ? new Date(event.received_at).toISOString() : ts.toISOString();
        nextHistory.push({ at: iso, cpu_pct: cpu });
      }
    }

    const cutoff = Date.now() - HISTORY_WINDOW_MS;
    const filtered = nextHistory.filter((sample) => {
      const sampleDate = new Date(sample.at);
      return !Number.isNaN(sampleDate.getTime()) && sampleDate.getTime() >= cutoff;
    });

    if (filtered.length > MAX_HISTORY_POINTS) {
      filtered.splice(0, filtered.length - MAX_HISTORY_POINTS);
    }

    this.cpuHistory.set(event.agent_id, filtered);
    return filtered.map((sample) => ({ ...sample }));
  }

  listAgents(): { id: string; label: string; url: string }[] {
    return this.configs.map((c) => ({ id: c.id, label: c.label, url: c.url }));
  }
}

let hubSingleton: AgentHub | null = null;

function parseEnv(): AgentConfigResolved[] {
  const raw = dynamicEnv.AGENT_ENDPOINTS || dynamicEnv.VITE_AGENT_ENDPOINTS || process.env.AGENT_ENDPOINTS || '';
  //console.log('[AgentHub] parse AGENT_ENDPOINTS raw =', raw);
  // Format: id|label|ws://host:port/ws;id2|label2|ws://...
  // Or simpler: ws://host:port/ws;ws://other:8080/ws (auto id,label)
  return raw
    .split(';')
    .map((s) => s.trim())
    .filter(Boolean)
    .map((entry, idx) => {
      const parts = entry.split('|');
      if (parts.length === 3) {
        const [id, label, url] = parts;
        return { id: id || `agent-${idx + 1}`, label: label || `Agent ${idx + 1}`, url };
      }
      const url = parts[0];
      return { id: `agent-${idx + 1}`, label: `Agent ${idx + 1}`, url };
    });
}

export function getHub(): AgentHub {
  if (!hubSingleton) {
    const configs = parseEnv();
    hubSingleton = new AgentHub(configs);
  }
  return hubSingleton;
}
