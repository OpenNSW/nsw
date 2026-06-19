# `@opennsw/jsonforms-renderers`

Custom [JSON Schema Form (JSON Forms)](https://jsonforms.io/) renderers for React, styled with [Radix UI Themes](https://www.radix-ui.com/themes).

This package provides a collection of JSON Forms controls and layout renderers tailored to match the visual styling, theme, and UX of the OpenNSW portal applications.

## Features

- **Radix UI Integration**: All components are constructed using `@radix-ui/themes` and layout elements (`Box`, `Flex`, `Text`, etc.).
- **Native Date & Time Inputs**: Relies completely on native browser-supported picker interfaces (no heavy external date picker libraries like `react-day-picker` or Popover wrappers), ensuring a fast, lightweight, and responsive experience on desktop and mobile.
- **Custom File Uploader Control**: A rich, interactive file picker component (`FileControl`) supporting upload progression and state.
- **Pre-configured Tester Rankings**: Simplifies registration by bundling ready-to-use ranks and testers.

## Available Renderers

### Controls

- **[DateControl](src/renderers/DateControl.tsx)**: Formats `date`, `time`, and `date-time` inputs. Combines native inputs that split space equally in `date-time` combination mode and expand to `100%` container width otherwise.
- **[TextControl](src/renderers/TextControl.tsx)**: Handles standard string schemas with optional placeholders.
- **[NumberControl](src/renderers/NumberControl.tsx)**: Handles numeric and integer inputs.
- **[BooleanControl](src/renderers/BooleanControl.tsx)**: Toggle switch control for boolean values.
- **[RadioControl](src/renderers/RadioControl.tsx)**: Option picker rendered as radio group buttons.
- **[SelectControl](src/renderers/SelectControl.tsx)**: Select drop-down control for `enum` or `oneOf` schemas.
- **[FileControl](src/renderers/FileControl.tsx)**: File picker control supporting custom attachments.

### Layouts

- **Vertical & Horizontal Layouts**: Renders standard JSON Forms layouts (`VerticalLayout`, `HorizontalLayout`).
- **Group Layout**: Standard boxed grouping for forms sections.
- **Categorization Layout**: Renders category schemas using tabbed views powered by Radix `Tabs.Root`.

---

## Installation & Setup

To use the custom renderers, register them inside the `<JsonForms>` provider in your React application:

```tsx
import { JsonForms } from '@jsonforms/react'
import { 
  materialCells, // fallback cells 
} from '@jsonforms/material-renderers' // or other fallbacks
import openNswRenderers from '@opennsw/jsonforms-renderers'

function App() {
  const [data, setData] = useState({})
  
  return (
    <JsonForms
      schema={schema}
      uischema={uischema}
      data={data}
      renderers={openNswRenderers}
      onChange={({ data }) => setData(data)}
    />
  )
}
```

## Development

All standard commands should be executed from the portals monorepo root workspace using `pnpm`:

```bash
# Build the renderers package
pnpm --filter @opennsw/jsonforms-renderers build

# Run TypeScript type check
pnpm --filter @opennsw/jsonforms-renderers type-check

# Lint files
pnpm --filter @opennsw/jsonforms-renderers lint
```
