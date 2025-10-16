import type { RequestHandler } from './$types';
import { getHub, type HubEvent } from '$lib/server/agentHub';

export const GET: RequestHandler = async () => {
  const hub = getHub();

  const stream = new ReadableStream({
    start(controller) {
      const enc = new TextEncoder();
      const send = (data: unknown) => {
        controller.enqueue(enc.encode(`data: ${JSON.stringify(data)}\n\n`));
      };

      const listener = (evt: HubEvent) => {
        send(evt);
      };

      hub.on('event', listener);

      // initial bootstrap list
      send({ type: 'agent_list', agents: hub.listAgents(), ts: new Date().toISOString() });

      const heartbeat = setInterval(() => {
        controller.enqueue(enc.encode(`: ping ${Date.now()}\n\n`));
      }, 15000);

      return () => {
        clearInterval(heartbeat);
        hub.off('event', listener);
      };
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
