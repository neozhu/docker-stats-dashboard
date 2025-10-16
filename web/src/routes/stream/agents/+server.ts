import type { RequestHandler } from './$types';
import { getHub, type HubEvent } from '$lib/server/agentHub';

export const GET: RequestHandler = async () => {
  const hub = getHub();

  const stream = new ReadableStream({
    start(controller) {
      const enc = new TextEncoder();
      let closed = false;
      const clientId = Math.random().toString(36).slice(2, 10);
      //console.log('[SSE] client connect', clientId);

      const safeEnqueue = (chunk: Uint8Array) => {
        if (!closed) {
          try {
            controller.enqueue(chunk);
          } catch {
            // swallow if already closed concurrently
          }
        }
      };

      const send = (data: unknown) => {
        safeEnqueue(enc.encode(`data: ${JSON.stringify(data)}\n\n`));
        const t = (typeof data === 'object' && data && 'type' in data) ? (data as { type?: string }).type : undefined;
        //console.log('[SSE] send event ->', clientId, t);
      };

      const listener = (evt: HubEvent) => {
        send(evt);
      };

      hub.on('event', listener);

      // initial bootstrap list
      send({ type: 'agent_list', agents: hub.listAgents(), ts: new Date().toISOString() });

      const heartbeat = setInterval(() => {
        safeEnqueue(enc.encode(`: ping ${Date.now()}\n\n`));
        //console.log('[SSE] heartbeat ->', clientId);
      }, 15000);

      const cleanup = () => {
        if (closed) return;
        closed = true;
        //console.log('[SSE] cleanup', clientId);
        clearInterval(heartbeat);
        hub.off('event', listener);
        try { controller.close(); } catch { /* ignore */ }
      };

      // When consumer cancels (client disconnect), cancel() is called
      // Return cleanup for older implementations; also implement cancel
      // in case runtime prefers that path
      // (Some Node versions may not call pull/close semantics for SSE early aborts.)
      // We expose both.
  // @ts-expect-error augmenting underlying controller instance with cancel for early abort handling
      this.cancel = cleanup;
      return cleanup;
    }
  });

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream; charset=utf-8',
      'Cache-Control': 'no-cache, no-transform',
      'Connection': 'keep-alive'
    }
  });
};
