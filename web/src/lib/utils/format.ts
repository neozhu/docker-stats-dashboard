export function formatBytes(bytes: number, fractionDigits = 1): string {
	if (!Number.isFinite(bytes)) return '0 B';
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	let value = Math.max(bytes, 0);
	let unitIndex = 0;

	while (value >= 1024 && unitIndex < units.length - 1) {
		value /= 1024;
		unitIndex += 1;
	}

	return `${value.toFixed(unitIndex === 0 ? 0 : fractionDigits)} ${units[unitIndex]}`;
}

export function formatPercent(value: number, fractionDigits = 1): string {
	if (!Number.isFinite(value)) return '0%';
	return `${value.toFixed(fractionDigits)}%`;
}

export function formatDuration(seconds: number): string {
	if (!Number.isFinite(seconds) || seconds <= 0) return '0s';

	const units: Array<[label: string, value: number]> = [
		['h', 3600],
		['m', 60],
		['s', 1]
	];

	let remaining = Math.floor(seconds);
	const parts: string[] = [];

	for (const [label, unitSeconds] of units) {
		if (remaining >= unitSeconds) {
			const amount = Math.floor(remaining / unitSeconds);
			remaining %= unitSeconds;
			parts.push(`${amount}${label}`);
		}
	}

	return parts.join(' ');
}

export function formatDateRelative(isoString: string | null): string {
	if (!isoString) return 'never';
	const from = new Date(isoString);
	if (Number.isNaN(from.valueOf())) return 'unknown';

	const deltaMs = Date.now() - from.valueOf();
	if (deltaMs < 1000) return 'just now';
	if (deltaMs < 60_000) return `${Math.floor(deltaMs / 1000)}s ago`;
	if (deltaMs < 3_600_000) return `${Math.floor(deltaMs / 60_000)}m ago`;
	if (deltaMs < 86_400_000) return `${Math.floor(deltaMs / 3_600_000)}h ago`;

	return from.toLocaleString();
}
