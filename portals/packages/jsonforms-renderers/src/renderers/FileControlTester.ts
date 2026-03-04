import { rankWith, formatIs, or } from '@jsonforms/core';

export const FileControlTester = rankWith(
    3, // Rank
    or(formatIs('data-url'), formatIs('file'))
);