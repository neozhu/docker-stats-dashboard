<script lang="ts">
  import { browser } from '$app/environment';
  import { onDestroy, onMount } from 'svelte';
  import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { formatBytes, formatDateRelative, formatDuration, formatPercent } from '$lib/utils/format';
  import type { AgentConnectionState, ContainerStatsBatch } from '$lib/types/messages';
  import { cn } from '$lib/utils';
  import { agents as agentsStore, latestBatches, startSSE, stopSSE } from '$lib/stores/agentData';

  let expandedAgents = new Set<string>();
  function toggleExpanded(agentId: string) {
    const next = new Set(expandedAgents);
    if (next.has(agentId)) next.delete(agentId); else next.add(agentId);
    expandedAgents = next;
  }

  let sequenceCounters = new Map<string, number>();

  if (browser) {
    onMount(() => {
      startSSE();
      return () => stopSSE();
    });
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
 
<section class="mb-6">
  <div class="rounded-lg border border-muted/60 bg-muted/30 p-4 text-sm text-muted-foreground">
    通过服务端聚合 (SSE) 自动加载内网 Agents；无需在浏览器中配置 ws 地址。
  </div>
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
      {@const batch = $latestBatches.get(agent.id) as ContainerStatsBatch}
			{@const allContainers = batch ? [...batch.containers].sort((a, b) => b.cpu_pct - a.cpu_pct) : []}
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
								<p class="text-xs text-muted-foreground mt-2">Sequence {sequenceCounters.get(agent.id) ?? batch.sequence}</p>
							</div>
						</div>

							<div class="space-y-4">
								<div class="flex items-center justify-between">
									<h3 class="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
										{isExpanded ? 'Containers (all)' : 'Top containers by CPU'}
									</h3>
									{#if allContainers.length > 5}
										<Button variant="ghost" size="sm" class="text-xs" on:click={() => toggleExpanded(agent.id)}>
											{isExpanded ? 'Show top 5' : `Show all (${allContainers.length})`}
										</Button>
									{/if}
								</div>

								<div class="space-y-4 max-h-96 overflow-y-auto pr-1 scrollbar-thin">
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
