import { describe, expect, it } from 'vitest';
import { formatDatetimeLocal, formatDuration, formatTime } from './time.js';

describe('time utilities', () => {
    it('formats elapsed seconds as HH:MM:SS', () => {
        expect(formatTime(0)).toBe('00:00:00');
        expect(formatTime(3661)).toBe('01:01:01');
    });

    it('formats duration with hours and minutes', () => {
        expect(formatDuration(0)).toBe('0m');
        expect(formatDuration(59)).toBe('0m');
        expect(formatDuration(3540)).toBe('59m');
        expect(formatDuration(3660)).toBe('1h 1m');
    });

    it('formats date for datetime-local input value', () => {
        const date = new Date('2024-02-03T04:05:00');
        expect(formatDatetimeLocal(date)).toBe('2024-02-03T04:05');
    });
});
