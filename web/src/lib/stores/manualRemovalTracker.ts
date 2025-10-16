const manualRemovals = new Set<string>();

export function markManualRemoval(id: string): void {
	manualRemovals.add(id);
}

export function consumeManualRemoval(id: string): boolean {
	const isManual = manualRemovals.has(id);
	if (isManual) {
		manualRemovals.delete(id);
	}
	return isManual;
}

export function clearManualRemoval(id: string): void {
	manualRemovals.delete(id);
}

export function isMarkedForManualRemoval(id: string): boolean {
	return manualRemovals.has(id);
}

export function resetManualRemovalTracker(): void {
	manualRemovals.clear();
}
