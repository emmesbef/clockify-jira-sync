const maxSummaryWords = 5;
const supportedLogRoundingMinutes = new Set([0, 5, 10, 15, 30, 60]);

export function normalizeSummaryWordLimit(limit) {
    const parsed = Number.isFinite(limit) ? Math.trunc(limit) : 0;
    if (parsed < 0) return 0;
    if (parsed > maxSummaryWords) return maxSummaryWords;
    return parsed;
}

export function normalizeLogRoundingMin(value) {
    const parsed = Number.isFinite(value) ? Math.trunc(value) : 0;
    return supportedLogRoundingMinutes.has(parsed) ? parsed : 0;
}
