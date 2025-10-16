import WebSocket from 'ws';
import { EventEmitter } from 'node:events';

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

export type HubEvent =
  | { type: 'agent_status'; agent_id: string; status: 'connecting' | 'connected' | 'error' | 'closed'; label: string; at: string }
  | { type: 'container_stats_batch'; agent_id: string; label: string; payload: any; received_at: string };

class AgentConnection {
  private ws: WebSocket | null = null;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private stopped = false;

  constructor(private cfg: AgentConfigResolved, private emit: (e: HubEvent) => void) {
    this.connect();
  }

  private connect() {
    if (this.stopped) return;
    console.log('[AgentHub] connecting ->', this.cfg.id, this.cfg.url);
    this.emit({ type: 'agent_status', agent_id: this.cfg.id, status: 'connecting', label: this.cfg.label, at: new Date().toISOString() });
    this.ws = new WebSocket(this.cfg.url);

    this.ws.on('open', () => {
      console.log('[AgentHub] connected  ->', this.cfg.id);
      this.emit({ type: 'agent_status', agent_id: this.cfg.id, status: 'connected', label: this.cfg.label, at: new Date().toISOString() });
    });

    this.ws.on('message', (data) => {
      try {
        const parsed = JSON.parse(data.toString());
        const batchType = parsed?.type ?? 'unknown';
        console.log('[AgentHub] message   <-', this.cfg.id, batchType);
        if (batchType === 'container_stats_batch' || batchType === 'agent_status') {
          this.emit({
            type: 'container_stats_batch',
            agent_id: this.cfg.id,
            label: this.cfg.label,
            payload: parsed,
            received_at: new Date().toISOString()
          });
        }
      } catch (err) {
        console.error('Failed to parse agent message', this.cfg.id, err);
      }
    });

    this.ws.on('close', () => {
      console.log('[AgentHub] closed     x', this.cfg.id);
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
    console.log('[AgentHub] reconnect in 3s', this.cfg.id);
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

  constructor(configs: AgentConfigResolved[]) {
    super();
    this.configs = configs;
    console.log('[AgentHub] bootstrap with', configs.length, 'agents');
    this.bootstrap();
  }

  private bootstrap() {
    for (const cfg of this.configs) {
      const conn = new AgentConnection(cfg, (e) => this.emit('event', e));
      this.agents.set(cfg.id, conn);
    }
  }

  listAgents(): { id: string; label: string; url: string }[] {
    return this.configs.map((c) => ({ id: c.id, label: c.label, url: c.url }));
  }
}

let hubSingleton: AgentHub | null = null;

function parseEnv(): AgentConfigResolved[] {
  const raw = process.env.AGENT_ENDPOINTS || '';
  console.log('[AgentHub] parse AGENT_ENDPOINTS raw =', raw);
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
