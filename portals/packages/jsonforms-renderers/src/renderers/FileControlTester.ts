// renderers/fileControlTester.ts
import { rankWith, and, schemaMatches} from '@jsonforms/core';
import type { JsonSchema } from '@jsonforms/core';

export const FileControlTester = rankWith(
    10, // high rank to beat default array renderer
    and(
        // Match both single-file (type: string) and multi-file (type: array)
        schemaMatches((schema: JsonSchema) => {
            if (schema.type === 'string' && schema.format === 'file') return true;
            if (schema.type === 'array') {
                const items = schema.items as JsonSchema | undefined;
                return items?.type === 'string' && items?.format === 'file';
            }
            return false;
        })
    )
);