import { describe, expect, it } from 'vitest';
import { formatBytes, formatDateRelative, formatDuration, formatPercent } from './format';

describe('formatBytes', () => {
	it('formats bytes with appropriate units', () => {
		expect(formatBytes(500)).toBe('500 B');
		expect(formatBytes(1024)).toBe('1.0 KB');
		expect(formatBytes(1048576)).toBe('1.0 MB');
	});
});

describe('formatPercent', () => {
	it('formats percentage values', () => {
		expect(formatPercent(12.345)).toBe('12.3%');
		expect(formatPercent(0)).toBe('0.0%');
	});
});

describe('formatDuration', () => {
	it('summarises seconds into h/m/s', () => {
		expect(formatDuration(0)).toBe('0s');
		expect(formatDuration(59)).toBe('59s');
		expect(formatDuration(61)).toBe('1m 1s');
		expect(formatDuration(3600)).toBe('1h');
	});
});

describe('formatDateRelative', () => {
	it('handles missing or invalid inputs', () => {
		expect(formatDateRelative(null)).toBe('never');
		expect(formatDateRelative('invalid')).toBe('unknown');
	});

	it('returns human readable relative dates', () => {
		const now = new Date();
		expect(formatDateRelative(new Date(now.valueOf() - 500).toISOString())).toBe('just now');
		expect(formatDateRelative(new Date(now.valueOf() - 30_000).toISOString())).toBe('30s ago');
		expect(formatDateRelative(new Date(now.valueOf() - 120_000).toISOString())).toBe('2m ago');
	});
});
