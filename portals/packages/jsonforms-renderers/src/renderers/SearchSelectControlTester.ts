import { rankWith, schemaMatches } from '@jsonforms/core'
import type { JsonSchema } from '@jsonforms/core'

export const SearchSelectControlTester = rankWith(
  3,
  schemaMatches((schema: JsonSchema) => {
    const xs = (schema as Record<string, unknown>)['x-search']
    return schema.type === 'string' && typeof xs === 'object' && xs !== null
  }),
)
