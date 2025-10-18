<script lang="ts">
  import { browser } from '$app/environment';
	import { onDestroy, onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { flip } from 'svelte/animate';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '$lib/components/ui/card';
import { Button } from '$lib/components/ui/button';
import { formatBytes, formatDateRelative, formatPercent } from '$lib/utils/format';
import type { AgentConnectionState, ContainerResourceSample, ContainerStatsBatch } from '$lib/types/messages';
  import { cn } from '$lib/utils';
  import { agents as agentsStore, latestBatches, startSSE, stopSSE } from '$lib/stores/agentData';

  let expandedAgents = new Set<string>();
  function toggleExpanded(agentId: string) {
    const next = new Set(expandedAgents);
    if (next.has(agentId)) next.delete(agentId); else next.add(agentId);
    expandedAgents = next;
  }

function containerKey(agentId: string, container: ContainerResourceSample, index: number): string {
  if (container.id && container.id.trim().length > 0) {
    return container.id;
  }
  const suffix = container.name && container.name.trim().length > 0 ? container.name : index.toString();
  return `${agentId}-${suffix}-${index}`;
}

  if (browser) {
    onMount(() => {
      startSSE();
      return () => stopSSE();
    });
  }

type UIAgentStatus = AgentConnectionState | 'closed';

function statusBadgeClasses(status: UIAgentStatus): string {
	const base = 'inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium shadow-sm';
	switch (status) {
		case 'connected':
			return cn(base, 'border-emerald-500/40 bg-emerald-500/10 text-emerald-300');
		case 'connecting':
			return cn(base, 'border-amber-400/40 bg-amber-400/10 text-amber-200');
		case 'error':
			return cn(base, 'border-destructive/40 bg-destructive/10 text-destructive');
		case 'closed':
			return cn(base, 'border-muted/40 bg-muted/20 text-muted-foreground');
		default:
			return cn(base, 'border-muted bg-muted text-muted-foreground');
	}
}

function statusBadgeLabel(status: UIAgentStatus): string {
	switch (status) {
		case 'connected':
			return 'connected';
		case 'connecting':
			return 'connecting';
		case 'error':
			return 'error';
		case 'closed':
			return 'closed';
		default:
			return 'placeholder';
	}
}

	// Map CPU % to semantic CSS custom property (defined in theme)
	function cpuBarColor(pct: number): string {
		const p = Math.max(0, Math.min(pct, 100));
		if (p < 30) return 'var(--color-cpu-bar-low)';
		if (p < 50) return 'var(--color-cpu-bar-moderate)';
		if (p < 70) return 'var(--color-cpu-bar-elevated)';
		if (p < 85) return 'var(--color-cpu-bar-high)';
		return 'var(--color-cpu-bar-critical)';
	}

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

  // removed manual endpoint input logic
</script>
 

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
      {@const batch = $latestBatches.get(agent.id) as ContainerStatsBatch}
      {@const rawContainers = (batch && Array.isArray((batch as any).containers)) ? (batch as any).containers : []}
		{@const allContainers = rawContainers.length ? [...rawContainers].sort((a, b) => b.cpu_pct - a.cpu_pct) : []}
		{@const isExpanded = expandedAgents.has(agent.id)}
		{@const containers = isExpanded ? allContainers : allContainers.slice(0, 5)}

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
                                                        </div>
                                                </div>

							<div class="space-y-4">
								<div class="flex items-center justify-between">
									<h3 class="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
										{isExpanded ? 'Containers (all)' : 'Top containers by CPU'}
									</h3>
									{#if allContainers.length > 5}
										<Button variant="ghost" size="sm" class="text-xs" onclick={() => toggleExpanded(agent.id)}>
											{isExpanded ? 'Show top 5' : `Show all (${allContainers.length})`}
										</Button>
									{/if}
								</div>

        <div class="space-y-4">
        {#each containers as container, i (containerKey(agent.id, container, i))}
									<div class="space-y-1.5 container-row"
										 animate:flip={{ duration: 220 }}
										 transition:fade={{ duration: 120 }}
										 on:introend={(e) => {
										   // Force highlight after first intro for new items
										   const el = e.currentTarget as HTMLElement;
										   el.classList.add('flash-highlight');
										   setTimeout(() => el.classList.remove('flash-highlight'), 500);
										 }}>
										<div class="flex items-center justify-between text-sm font-medium">
											<span>{container.name}</span>
											<span class="text-muted-foreground">
												{formatPercent(container.cpu_pct)}
											</span>
										</div>
										<div class="h-2 rounded-full bg-muted">
											<div
												class="h-2 rounded-full transition-[width,background-color] duration-300"
												style={`width: ${Math.min(container.cpu_pct, 100)}%; background:${cpuBarColor(container.cpu_pct)}`}
											></div>
										</div>
										<div class="flex justify-between text-xs text-muted-foreground">
											<span>
												{formatBytes(container.mem_bytes)} / {formatBytes(container.mem_limit_bytes)}
											</span>
                                                                                        <span>Net I/O {formatBytes(container.net_io_bytes)}</span>
                                                                                </div>
									</div>
								{/each}
							</div>
						</div>
					{:else}
						<p class="text-sm text-muted-foreground">Waiting for the first stats batch from the agent...</p>
					{/if}
				</CardContent>

					<CardFooter class="mt-auto flex items-center justify-between">
						<div class="text-xs text-muted-foreground">Last seen {formatDateRelative(agent.lastSeenAt)}</div>
					</CardFooter>
			</Card>
				{/each}
				</div>
			{/if}
			{/key}
	{/if}
</section>

<style>
	:global(.container-row) {
		position: relative;
	}

	/* Flash highlight when a container row first appears */
	:global(.container-row.flash-highlight) {
		animation: flash-bg 0.5s ease-out;
	}

	@keyframes flash-bg {
		0% { box-shadow: 0 0 0 0 var(--accent); background: var(--accent); color: var(--accent-foreground); }
		60% { box-shadow: 0 0 0 2px var(--accent); }
		100% { box-shadow: 0 0 0 0 transparent; background: transparent; }
	}

	/* Respect reduced motion */
	@media (prefers-reduced-motion: reduce) {
		:global(.container-row[style*="transform"]) { transition: none !important; animation: none !important; }
	}
</style>
