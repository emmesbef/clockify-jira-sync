import { defineConfig } from 'vitest/config';

export default defineConfig({
    test: {
        include: ['src/**/*.test.js'],
        coverage: {
            provider: 'v8',
            reporter: ['text', 'lcov', 'json-summary'],
            reportsDirectory: 'coverage',
            include: ['src/utils/**/*.js'],
            exclude: ['src/**/*.test.js'],
            all: true,
        },
    },
});
