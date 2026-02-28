import { rankWith, and, uiTypeIs, schemaTypeIs } from '@jsonforms/core';

export const ArrayControlTester = rankWith(
    2,
    and(uiTypeIs('Control'), schemaTypeIs('array'))
);
