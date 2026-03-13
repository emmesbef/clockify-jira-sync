import { describe, expect, it } from 'vitest';

import { normalizeLogRoundingMin, normalizeSummaryWordLimit } from './settings.js';

describe('normalizeSummaryWordLimit', () => {
    it('clamps values to the supported 0..5 range', () => {
        expect(normalizeSummaryWordLimit(-3)).toBe(0);
        expect(normalizeSummaryWordLimit(0)).toBe(0);
        expect(normalizeSummaryWordLimit(2)).toBe(2);
        expect(normalizeSummaryWordLimit(5)).toBe(5);
        expect(normalizeSummaryWordLimit(8)).toBe(5);
    });

    it('handles non-integer and non-finite inputs', () => {
        expect(normalizeSummaryWordLimit(3.9)).toBe(3);
        expect(normalizeSummaryWordLimit(Number.NaN)).toBe(0);
        expect(normalizeSummaryWordLimit(Number.POSITIVE_INFINITY)).toBe(0);
    });
});

describe('normalizeLogRoundingMin', () => {
    it('keeps only supported rounding values', () => {
        expect(normalizeLogRoundingMin(0)).toBe(0);
        expect(normalizeLogRoundingMin(5)).toBe(5);
        expect(normalizeLogRoundingMin(10)).toBe(10);
        expect(normalizeLogRoundingMin(15)).toBe(15);
        expect(normalizeLogRoundingMin(30)).toBe(30);
        expect(normalizeLogRoundingMin(60)).toBe(60);
    });

    it('normalizes unsupported values to 0', () => {
        expect(normalizeLogRoundingMin(-1)).toBe(0);
        expect(normalizeLogRoundingMin(7)).toBe(0);
        expect(normalizeLogRoundingMin(14.9)).toBe(0);
        expect(normalizeLogRoundingMin(Number.NaN)).toBe(0);
    });
});
