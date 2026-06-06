import type { JsonSchema, UISchemaElement } from '@jsonforms/core'

export type Fixture = {
  id: string
  name: string
  schema: JsonSchema
  uischema: UISchemaElement
  data?: Record<string, unknown>
}

// One fixture per renderer/component. Selecting a fixture loads its schema +
// uischema into the editors; both are live-editable from there.
export const fixtures: Fixture[] = [
  {
    id: 'text',
    name: 'Text',
    schema: {
      type: 'object',
      properties: {
        name: { type: 'string', description: 'A plain text field' },
        bio: { type: 'string', description: 'Multi-line via options.multi' },
      },
      required: ['name'],
    },
    uischema: {
      type: 'VerticalLayout',
      elements: [
        { type: 'Control', scope: '#/properties/name' },
        { type: 'Control', scope: '#/properties/bio', options: { multi: true } },
      ],
    } as UISchemaElement,
  },
  {
    id: 'number',
    name: 'Number',
    schema: {
      type: 'object',
      properties: {
        price: { type: 'number', description: 'Decimal value' },
        quantity: { type: 'integer', minimum: 0, description: 'Whole number ≥ 0' },
      },
    },
    uischema: {
      type: 'VerticalLayout',
      elements: [
        { type: 'Control', scope: '#/properties/price' },
        { type: 'Control', scope: '#/properties/quantity' },
      ],
    } as UISchemaElement,
  },
  {
    id: 'boolean',
    name: 'Boolean',
    schema: {
      type: 'object',
      properties: {
        agree: { type: 'boolean', description: 'Terms & conditions' },
      },
      required: ['agree'],
    },
    uischema: {
      type: 'VerticalLayout',
      elements: [{ type: 'Control', scope: '#/properties/agree' }],
    } as UISchemaElement,
  },
  {
    id: 'radio',
    name: 'Radio',
    schema: {
      type: 'object',
      properties: {
        size: { type: 'string', enum: ['Small', 'Medium', 'Large'], description: 'Pick one' },
      },
    },
    uischema: {
      type: 'VerticalLayout',
      elements: [{ type: 'Control', scope: '#/properties/size', options: { format: 'radio' } }],
    } as UISchemaElement,
  },
  {
    id: 'select',
    name: 'Select',
    schema: {
      type: 'object',
      properties: {
        country: { type: 'string', enum: ['Sri Lanka', 'India', 'Maldives'], description: 'Dropdown' },
      },
    },
    uischema: {
      type: 'VerticalLayout',
      elements: [{ type: 'Control', scope: '#/properties/country' }],
    } as UISchemaElement,
  },
  {
    id: 'date',
    name: 'Date / Time',
    schema: {
      type: 'object',
      properties: {
        eventDate: { type: 'string', format: 'date', description: 'Date only (yyyy-MM-dd)' },
        appointment: { type: 'string', format: 'date-time', description: 'Date + time (RFC 3339)' },
        openingTime: { type: 'string', format: 'time', description: 'Time only (native picker)' },
      },
      required: ['eventDate'],
    },
    uischema: {
      type: 'VerticalLayout',
      elements: [
        { type: 'Control', scope: '#/properties/eventDate' },
        { type: 'Control', scope: '#/properties/appointment' },
        { type: 'Control', scope: '#/properties/openingTime' },
      ],
    } as UISchemaElement,
  },
  {
    id: 'file',
    name: 'File',
    schema: {
      type: 'object',
      properties: {
        avatar: { type: 'string', format: 'file', description: 'Single file' },
        attachments: {
          type: 'array',
          items: { type: 'string', format: 'file' },
          description: 'Multiple files',
        },
      },
    },
    uischema: {
      type: 'VerticalLayout',
      elements: [
        { type: 'Control', scope: '#/properties/avatar' },
        { type: 'Control', scope: '#/properties/attachments' },
      ],
    } as UISchemaElement,
  },
  {
    id: 'array',
    name: 'Array (objects)',
    schema: {
      type: 'object',
      properties: {
        items: {
          type: 'array',
          description: 'List of line items',
          items: {
            type: 'object',
            properties: {
              description: { type: 'string' },
              qty: { type: 'integer', minimum: 1 },
            },
            required: ['description'],
          },
        },
      },
    },
    uischema: {
      type: 'VerticalLayout',
      elements: [{ type: 'Control', scope: '#/properties/items' }],
    } as UISchemaElement,
    data: { items: [{ description: 'First item', qty: 1 }] },
  },
  {
    id: 'horizontal',
    name: 'Horizontal layout',
    schema: {
      type: 'object',
      properties: {
        firstName: { type: 'string' },
        lastName: { type: 'string' },
      },
    },
    uischema: {
      type: 'HorizontalLayout',
      elements: [
        { type: 'Control', scope: '#/properties/firstName' },
        { type: 'Control', scope: '#/properties/lastName' },
      ],
    } as UISchemaElement,
  },
  {
    id: 'group',
    name: 'Group layout',
    schema: {
      type: 'object',
      properties: {
        street: { type: 'string' },
        city: { type: 'string' },
      },
    },
    uischema: {
      type: 'Group',
      label: 'Address',
      elements: [
        { type: 'Control', scope: '#/properties/street' },
        { type: 'Control', scope: '#/properties/city' },
      ],
    } as UISchemaElement,
  },
  {
    id: 'categorization',
    name: 'Categorization',
    schema: {
      type: 'object',
      properties: {
        name: { type: 'string' },
        email: { type: 'string' },
        notes: { type: 'string' },
      },
    },
    uischema: {
      type: 'Categorization',
      elements: [
        {
          type: 'Category',
          label: 'Personal',
          elements: [
            { type: 'Control', scope: '#/properties/name' },
            { type: 'Control', scope: '#/properties/email' },
          ],
        },
        {
          type: 'Category',
          label: 'More',
          elements: [{ type: 'Control', scope: '#/properties/notes', options: { multi: true } }],
        },
      ],
    } as UISchemaElement,
  },
  {
    id: 'label',
    name: 'Label',
    schema: {
      type: 'object',
      properties: {
        name: { type: 'string' },
      },
    },
    uischema: {
      type: 'VerticalLayout',
      elements: [
        { type: 'Label', text: 'Section heading via Label renderer' },
        { type: 'Control', scope: '#/properties/name' },
      ],
    } as UISchemaElement,
  },
]
