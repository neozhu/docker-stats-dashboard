<script lang="ts">
	import { browser } from '$app/environment';
	import { onDestroy } from 'svelte';
	import {
		Card,
		CardContent,
		CardDescription,
		CardFooter,
		CardHeader,
		CardTitle
	} from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
import { agentRegistry, addAgentEndpoint, removeAgentEndpoint, setAgentStatus } from '$lib/stores/agentRegistry';
import { clearManualRemoval, consumeManualRemoval, markManualRemoval } from '$lib/stores/manualRemovalTracker';
	import { connectAgentSocket, type AgentSocket } from '$lib/transport/agentSocket';
	import {
		formatBytes,
		formatDateRelative,
		formatDuration,
		formatPercent
	} from '$lib/utils/format';
	import type { AgentConnectionState, ContainerStatsBatch, AgentStatusMessage } from '$lib/types/messages';
	import { cn } from '$lib/utils';
	import { onMount } from 'svelte';

	const agentsStore = agentRegistry;
	let newEndpoint = '';
	let formError = '';
	let latestSnapshots = new Map<string, ContainerStatsBatch>();
	let sequenceCounters = new Map<string, number>();

	const activeSockets = new Map<string, AgentSocket>();
	const reconnectTimers = new Map<string, ReturnType<typeof setTimeout>>();

	function ensureConnection(agentId: string, agentLabel: string, endpoint: string) {
		if (!browser) return;
		if (activeSockets.has(agentId)) return;

		clearReconnect(agentId);
		setAgentStatus(agentId, 'connecting', null);

		const socket = connectAgentSocket(endpoint, {
			onConnect: () => {
				setAgentStatus(agentId, 'connected', new Date().toISOString());
			},
			onDisconnect: () => {
				activeSockets.delete(agentId);
				if (consumeManualRemoval(agentId)) {
					return;
				}
				setAgentStatus(agentId, 'error', null);
				scheduleReconnect(agentId, agentLabel, endpoint);
			},
			onStats: (payload) => {
				latestSnapshots = new Map(latestSnapshots).set(agentId, payload);
				sequenceCounters = new Map(sequenceCounters).set(agentId, payload.sequence);
				setAgentStatus(agentId, 'connected', payload.sent_at);
			},
			onStatus: (payload) => {
				const status = payload as AgentStatusMessage;
				setAgentStatus(agentId, 'connected', status.sent_at);
			}
		});

		activeSockets.set(agentId, socket);
	}

	function scheduleReconnect(agentId: string, agentLabel: string, endpoint: string) {
		if (reconnectTimers.has(agentId)) return;
		const timer = setTimeout(() => {
			reconnectTimers.delete(agentId);
			const agent = $agentsStore.find((entry) => entry.id === agentId);
			if (agent) {
				ensureConnection(agentId, agentLabel, endpoint);
			}
		}, 3000);
		reconnectTimers.set(agentId, timer);
	}

	function clearReconnect(agentId: string) {
		const timer = reconnectTimers.get(agentId);
		if (timer) {
			clearTimeout(timer);
			reconnectTimers.delete(agentId);
		}
	}

	function teardownConnection(agentId: string): boolean {
		clearReconnect(agentId);
		const socket = activeSockets.get(agentId);
		const hadSocket = Boolean(socket);
		if (socket) {
			socket.close();
		}
		activeSockets.delete(agentId);
		const next = new Map(latestSnapshots);
		next.delete(agentId);
		latestSnapshots = next;
		const seqNext = new Map(sequenceCounters);
		seqNext.delete(agentId);
		sequenceCounters = seqNext;
		return hadSocket;
	}

	$: if (browser) {
		const agents = $agentsStore;
		const seen = new Set<string>();

		for (const agent of agents) {
			seen.add(agent.id);
			ensureConnection(agent.id, agent.label, agent.url);
		}

		for (const [agentId] of activeSockets) {
			if (!seen.has(agentId)) {
				teardownConnection(agentId);
			}
		}
	}

	onDestroy(() => {
		activeSockets.forEach((socket) => socket.close());
		activeSockets.clear();
		reconnectTimers.forEach((timer) => clearTimeout(timer));
		reconnectTimers.clear();
	});

	function handleAddAgent() {
		formError = '';
		const sanitized = sanitizeEndpoint(newEndpoint.trim());

		if (!sanitized) {
			formError = 'Provide a WebSocket URL (e.g. ws://127.0.0.1:8080).';
			return;
		}

		addAgentEndpoint(sanitized);
		newEndpoint = '';
	}

	function handleRemoveAgent(id: string) {
		console.log('removeAgentEndpoint',id)
		markManualRemoval(id);
		const hadSocket = teardownConnection(id);
		removeAgentEndpoint(id);
		
		if (!hadSocket) {
			clearManualRemoval(id);
		}
	}

function statusBadgeClasses(status: AgentConnectionState): string {
	const base = 'inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium shadow-sm';

	switch (status) {
		case 'connected':
				return cn(base, 'border-emerald-500/40 bg-emerald-500/10 text-emerald-300');
			case 'connecting':
				return cn(base, 'border-amber-400/40 bg-amber-400/10 text-amber-200');
			case 'error':
				return cn(base, 'border-destructive/40 bg-destructive/10 text-destructive');
			default:
				return cn(base, 'border-muted bg-muted text-muted-foreground');
	}
}

function statusBadgeLabel(status: AgentConnectionState): string {
	switch (status) {
		case 'connected':
			return 'connected';
		case 'connecting':
			return 'connecting';
		case 'error':
			return 'error';
		default:
			return 'placeholder';
	}
	}

	const exampleEndpoints = ['ws://127.0.0.1:8080/ws', 'ws://192.168.1.40:8080/ws'];

	// Responsive column breakpoints: <640 => 1, 640-1023 => 2, >=1024 => 3
	let maxResponsiveCols = 1;

	function computeMaxCols(width: number): number {
		if (width < 640) return 1;
		if (width < 1024) return 2;
		return 3;
	}

	if (browser) {
		maxResponsiveCols = computeMaxCols(window.innerWidth);
		const resizeHandler = () => {
			const next = computeMaxCols(window.innerWidth);
			if (next !== maxResponsiveCols) {
				maxResponsiveCols = next;
			}
		};
		window.addEventListener('resize', resizeHandler);
		onDestroy(() => window.removeEventListener('resize', resizeHandler));
	}

	function sanitizeEndpoint(raw: string): string | null {
		if (!raw) return null;

		let url: URL;
		try {
			url = new URL(raw);
		} catch {
			try {
				url = new URL(`ws://${raw}`);
			} catch {
				return null;
			}
		}

		let protocol = url.protocol;
		if (protocol === 'http:') {
			protocol = 'ws:';
		} else if (protocol === 'https:') {
			protocol = 'wss:';
		}

		if (protocol !== 'ws:' && protocol !== 'wss:') {
			return null;
		}

		let pathname = url.pathname;
		if (!pathname || pathname === '/') {
			pathname = '/ws';
		}

		return `${protocol}//${url.host}${pathname}${url.search}${url.hash}`;
	}
</script>
 
<section class="mb-6">
	<div class="rounded-lg border border-muted/60 bg-muted/30 p-4 text-sm text-muted-foreground">
		Add any running agent to stream live Docker stats over WebSocket. Connections reconnect automatically if the agent restarts.
	</div>
 
</section>

<section class="grid gap-6 lg:grid-cols-[2fr_1fr]">
	<Card>
		<CardHeader class="space-y-1">
			<CardTitle>Add agent endpoint</CardTitle>
			<CardDescription>
				Paste the WebSocket URL exposed by the Go agent. Endpoints persist to localStorage so your list
				survives reloads.
			</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="space-y-3" on:submit|preventDefault={handleAddAgent}>
				<label class="text-sm font-medium text-muted-foreground" for="agent-endpoint">
					Agent endpoint
				</label>
				<div class="flex flex-col gap-2 sm:flex-row">
					<Input
						id="agent-endpoint"
						placeholder="ws://host:8080"
						bind:value={newEndpoint}
						class="sm:flex-1"
					/>
					<Button type="submit" class="sm:self-start">Save endpoint</Button>
				 
				</div>
				{#if formError}
					<p class="text-sm text-destructive">{formError}</p>
				{/if}
			</form>
		</CardContent>
		<CardFooter class="flex flex-wrap gap-2 text-xs text-muted-foreground">
			<span class="font-medium text-foreground">Example:</span>
			{#each exampleEndpoints as endpoint}
				<Button
					variant="outline"
					size="sm"
					class="font-mono"
					onclick={() => {
						newEndpoint = endpoint;
						formError = '';
					}}
				>
					{endpoint}
				</Button>
			{/each}
		</CardFooter>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>Workflow tips</CardTitle>
			<CardDescription>
				Live data arrives directly from each connected agent. Use these helpers while monitoring hosts.
			</CardDescription>
		</CardHeader>
		<CardContent class="space-y-3 text-sm text-muted-foreground">
			<ul class="space-y-2">
				<li class="flex items-start gap-2">
					<span class="mt-1 size-2 rounded-full bg-accent"></span>
					Add multiple endpoints to compare host utilisation side-by-side.
				</li>
				<li class="flex items-start gap-2">
					<span class="mt-1 size-2 rounded-full bg-accent"></span>
					Use the remove action once you finish inspecting a host; you can re-add it later from the form.
				</li>
				<li class="flex items-start gap-2">
					<span class="mt-1 size-2 rounded-full bg-accent"></span>
					The dashboard refreshes automatically whenever new batches arrive (target cadence: 1s).
				</li>
			</ul>
		</CardContent>
	</Card>
</section>

<section class="mt-10 space-y-6">
	{#if $agentsStore.length === 0}
		<Card>
			<CardHeader>
				<CardTitle>No agents yet</CardTitle>
				<CardDescription>
					Add your first agent above to preview container stats. The dashboard will list each host as a card with CPU and memory snapshots.
				</CardDescription>
			</CardHeader>
			<CardContent class="space-y-2 text-sm text-muted-foreground">
				<p>Need a target? Start the Go agent with:</p>
				<pre class="rounded-md bg-muted p-3 font-mono text-xs text-foreground">
docker run --rm -it \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/your-org/docker-monitor-agent:latest
				</pre>
			</CardContent>
		</Card>
	{:else}

		<!-- Responsive adaptive grid: max 3 columns; <640px=1; 640-1023px=2; >=1024px=3; never exceed agent count -->
		{#key $agentsStore.length}
		{#if $agentsStore.length > 0}
			{@const agentColumns = Math.min($agentsStore.length, maxResponsiveCols)}
			<div class="grid gap-6" style={`grid-template-columns: repeat(${agentColumns}, minmax(0,1fr));`}>
			{#each $agentsStore as agent (agent.id)}
			{@const batch = latestSnapshots.get(agent.id)}
			{@const containers =
				batch ? [...batch.containers].sort((a, b) => b.cpu_pct - a.cpu_pct).slice(0, 5) : []}

			<Card class="h-full flex flex-col">
				<CardHeader class="flex flex-row items-start justify-between gap-4">
					<div class="space-y-1">
						<CardTitle class="text-lg">{agent.label}</CardTitle>
						<CardDescription class="font-mono text-xs">{agent.url}</CardDescription>
					</div>
					<div class="flex flex-col items-end gap-2 text-right">
						<span class={statusBadgeClasses(agent.status)}>{statusBadgeLabel(agent.status)}</span>
						<span class="text-xs text-muted-foreground">
							Last sample {formatDateRelative(agent.lastSeenAt)}
						</span>
					</div>
				</CardHeader>

				<CardContent class="space-y-4">
					{#if batch}
						<div class="grid gap-4 md:grid-cols-3">
							<div class="rounded-lg border border-border/60 bg-card/50 p-4 text-sm">
								<p class="text-muted-foreground">Host CPU (approx)</p>
								<p class="mt-1 text-2xl font-semibold">{formatPercent(batch.agent_metrics.cpu_pct)}</p>
							</div>
							<div class="rounded-lg border border-border/60 bg-card/50 p-4 text-sm">
								<p class="text-muted-foreground">Host Memory (sum)</p>
								<p class="mt-1 text-2xl font-semibold">{formatBytes(batch.agent_metrics.mem_bytes)}</p>
							</div>
							<div class="rounded-lg border border-border/60 bg-card/50 p-4 text-sm">
								<p class="text-muted-foreground">Containers sampled</p>
								<p class="mt-1 text-2xl font-semibold">{batch.containers.length}</p>
								<p class="text-xs text-muted-foreground mt-2">Sequence {sequenceCounters.get(agent.id) ?? batch.sequence}</p>
							</div>
						</div>

						<div class="space-y-4">
							<h3 class="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
								Top containers by CPU
							</h3>

							<div class="space-y-4">
								{#each containers as container}
									<div class="space-y-1.5">
										<div class="flex items-center justify-between text-sm font-medium">
											<span>{container.name}</span>
											<span class="text-muted-foreground">
												{formatPercent(container.cpu_pct)}
											</span>
										</div>
										<div class="h-2 rounded-full bg-muted">
											<div
												class="h-2 rounded-full bg-accent transition-all"
												style={`width: ${Math.min(container.cpu_pct, 100)}%`}
											></div>
										</div>
										<div class="flex justify-between text-xs text-muted-foreground">
											<span>
												{formatBytes(container.mem_bytes)} / {formatBytes(container.mem_limit_bytes)}
											</span>
											<span>Uptime {formatDuration(container.uptime_secs)}</span>
										</div>
									</div>
								{/each}
							</div>
						</div>
					{:else}
						<p class="text-sm text-muted-foreground">
							Waiting for the first stats batch from the agent...
						</p>
					{/if}
				</CardContent>

				<CardFooter class="mt-auto flex items-center justify-between">
					<div class="text-xs text-muted-foreground">
						Created {formatDateRelative(agent.createdAt)}
					</div>
					<Button variant="ghost" size="sm" onclick={() => { 
						handleRemoveAgent(agent.id);}}>
						Remove
					</Button>
				</CardFooter>
			</Card>
			{/each}
			</div>
		{/if}
		{/key}
	{/if}
</section>
