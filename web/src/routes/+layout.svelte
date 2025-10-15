<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="flex min-h-screen flex-col bg-background text-foreground">
	<header class="border-b border-border/60 bg-background/80 backdrop-blur">
		<div class="mx-auto flex w-full max-w-6xl items-center justify-between px-6 py-4">
			<a class="flex items-center gap-2 font-semibold tracking-tight text-foreground" href="/">
				<div class="grid h-9 w-9 place-items-center rounded-lg bg-accent/20 ring-1 ring-accent/40">
					<span class="text-sm font-mono text-accent">DP</span>
				</div>
				<span>Docker Performance Monitor</span>
			</a>
			<span class="text-xs uppercase tracking-wide text-muted-foreground">
				Dashboard preview
			</span>
		</div>
	</header>

	<main class="flex-1">
		<div class="mx-auto w-full max-w-6xl px-6 py-10">
			{@render children?.()}
		</div>
	</main>

	<footer class="border-t border-border/60 bg-background/80">
		<div class="mx-auto w-full max-w-6xl px-6 py-4 text-xs text-muted-foreground">
			Keep an eye on the Go agent logs if a connection stops updating; the dashboard will auto-retry every few seconds.
		</div>
	</footer>
</div>
