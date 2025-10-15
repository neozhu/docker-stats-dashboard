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
	import { createMockAgentStream } from '$lib/mocks/mockAgentStream';
	import {
		formatBytes,
		formatDateRelative,
		formatDuration,
		formatPercent
	} from '$lib/utils/format';
import type { AgentConnectionState, ContainerStatsBatch } from '$lib/types/messages';
	import { cn } from '$lib/utils';

	const agentsStore = agentRegistry;
	let newEndpoint = '';
	let formError = '';
	let mockSnapshots = new Map<string, ContainerStatsBatch>();

	const streamDisposers = new Map<string, () => void>();

function ensureStream(agentId: string, agentLabel: string) {
	if (streamDisposers.has(agentId)) return;
	setAgentStatus(agentId, 'connecting', null);
	const stream = createMockAgentStream({ agentId, agentLabel });
	const unsubscribe = stream.subscribe((batch) => {
		mockSnapshots = new Map(mockSnapshots).set(agentId, batch);
		setAgentStatus(agentId, 'connected', batch.sent_at);
	});

		streamDisposers.set(agentId, () => {
			unsubscribe();
			stream.stop();
		});
	}

	function teardownStream(agentId: string) {
		const disposer = streamDisposers.get(agentId);
		if (!disposer) return;
		disposer();
		streamDisposers.delete(agentId);
		const next = new Map(mockSnapshots);
		next.delete(agentId);
		mockSnapshots = next;
	}

	$: if (browser) {
		const agents = $agentsStore;
		const seen = new Set<string>();

		for (const agent of agents) {
			seen.add(agent.id);
			ensureStream(agent.id, agent.label);
		}

		for (const [agentId] of streamDisposers) {
			if (!seen.has(agentId)) {
				teardownStream(agentId);
			}
		}
	}

	onDestroy(() => {
		streamDisposers.forEach((stop) => stop());
		streamDisposers.clear();
	});

	function handleAddAgent() {
		formError = '';
		const url = newEndpoint.trim();

		if (!url) {
			formError = 'Provide a WebSocket URL (e.g. ws://127.0.0.1:8080).';
			return;
		}

		if (!/^wss?:\/\//i.test(url)) {
			formError = 'Endpoint must start with ws:// or wss://';
			return;
		}

		addAgentEndpoint(url);
		newEndpoint = '';
	}

	function handleRemoveAgent(id: string) {
		teardownStream(id);
		removeAgentEndpoint(id);
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

	const exampleEndpoints = ['ws://127.0.0.1:8080', 'ws://192.168.1.40:8080'];
</script>

<section class="mb-6">
	<div class="rounded-lg border border-dashed border-muted/60 bg-muted/30 p-4 text-sm text-muted-foreground">
		WebSocket connections are not yet wired to live agents. The dashboard renders mock metrics so you
		can validate layout and workflows ahead of the agent integration work.
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
					on:click={() => {
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
				This preview focuses on shell components. Upcoming tasks will replace mock data with live WebSocket streams.
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
					Stats update every 1.5s via the mock feed to mimic the intended real-time cadence.
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
		{#each $agentsStore as agent (agent.id)}
			{@const batch = mockSnapshots.get(agent.id)}
			{@const containers =
				batch ? [...batch.containers].sort((a, b) => b.cpu_pct - a.cpu_pct).slice(0, 5) : []}

			<Card>
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
							Waiting for the mock stream to dispatch the first batch...
						</p>
					{/if}
				</CardContent>

				<CardFooter class="flex items-center justify-between">
					<div class="text-xs text-muted-foreground">
						Created {formatDateRelative(agent.createdAt)}
					</div>
					<Button variant="ghost" size="sm" on:click={() => handleRemoveAgent(agent.id)}>
						Remove
					</Button>
				</CardFooter>
			</Card>
		{/each}
	{/if}
</section>
