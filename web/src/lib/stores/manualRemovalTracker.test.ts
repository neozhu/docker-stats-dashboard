import { beforeEach, describe, expect, it } from 'vitest';
import {
	clearManualRemoval,
	consumeManualRemoval,
	isMarkedForManualRemoval,
	markManualRemoval,
	resetManualRemovalTracker
} from './manualRemovalTracker';

describe('manualRemovalTracker', () => {
	beforeEach(() => {
		resetManualRemovalTracker();
	});

	it('marks manual removals and consumes them once', () => {
		markManualRemoval('agent-1');
		expect(isMarkedForManualRemoval('agent-1')).toBe(true);
		expect(consumeManualRemoval('agent-1')).toBe(true);
		expect(isMarkedForManualRemoval('agent-1')).toBe(false);
		expect(consumeManualRemoval('agent-1')).toBe(false);
	});

	it('clears manual removal when requested explicitly', () => {
		markManualRemoval('agent-2');
		clearManualRemoval('agent-2');
		expect(isMarkedForManualRemoval('agent-2')).toBe(false);
		expect(consumeManualRemoval('agent-2')).toBe(false);
	});
});
