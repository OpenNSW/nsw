import { createAjv } from '@jsonforms/core'

// Shared Ajv instance for every JsonForms usage. createAjv applies JsonForms'
// own defaults (format validators etc.); useDefaults: true additionally lets
// Ajv populate schema `default` values into the data during validation, so
// defaulted fields are present without the user touching them.
export const ajv = createAjv({ useDefaults: true })
