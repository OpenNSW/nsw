import { rankWith, formatIs } from '@jsonforms/core';

export const FileControlTester = rankWith(
    3, // Rank
    formatIs('data-url')
);