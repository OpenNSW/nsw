import { rankWith, formatIs, type RankedTester } from '@jsonforms/core';

export const FileControlTester: RankedTester = rankWith(
    3,
    formatIs('data-url')
);
