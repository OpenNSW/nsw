import { StrictMode, useMemo, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { Theme, Flex, Box, Heading, Card, Text, Separator, TextArea, Badge } from '@radix-ui/themes'
import '@radix-ui/themes/styles.css'
import { JsonForms } from '@jsonforms/react'
import type { JsonSchema, UISchemaElement } from '@jsonforms/core'
import { radixRenderers } from '../src'
import { ajv } from './ajv'
import { fixtures, type Fixture } from './fixtures'

// Parse JSON text, returning the parsed value or an error message — never throws.
const parse = (text: string): { value?: unknown; error?: string } => {
  try {
    return { value: JSON.parse(text) }
  } catch (e) {
    return { error: e instanceof Error ? e.message : String(e) }
  }
}

type EditorProps = {
  label: string
  text: string
  error?: string
  onChange: (next: string) => void
}

const JsonEditor = ({ label, text, error, onChange }: EditorProps) => (
  <Box>
    <Flex justify="between" align="center" mb="1">
      <Text size="2" weight="bold">
        {label}
      </Text>
      {error ? <Badge color="red">invalid JSON</Badge> : <Badge color="green">ok</Badge>}
    </Flex>
    <TextArea
      value={text}
      onChange={(e) => onChange(e.target.value)}
      resize="vertical"
      style={{ fontFamily: 'monospace', fontSize: 12, minHeight: 220 }}
      color={error ? 'red' : undefined}
    />
    {error && (
      <Text size="1" color="red">
        {error}
      </Text>
    )}
  </Box>
)

function Playground() {
  const [selectedId, setSelectedId] = useState<string>(fixtures[0].id)

  // Editable JSON text for schema + uischema. Data is live read-only output.
  const [schemaText, setSchemaText] = useState(() => JSON.stringify(fixtures[0].schema, null, 2))
  const [uiText, setUiText] = useState(() => JSON.stringify(fixtures[0].uischema, null, 2))
  const [data, setData] = useState<Record<string, unknown>>(() => fixtures[0].data ?? {})

  // Last successfully parsed schema/uischema drive the form; a bad edit keeps
  // the previous good render and just surfaces the parse error.
  const [schema, setSchema] = useState<JsonSchema>(() => fixtures[0].schema)
  const [uischema, setUischema] = useState<UISchemaElement>(() => fixtures[0].uischema)
  const [schemaError, setSchemaError] = useState<string>()
  const [uiError, setUiError] = useState<string>()

  const loadFixture = (f: Fixture) => {
    setSelectedId(f.id)
    setSchemaText(JSON.stringify(f.schema, null, 2))
    setUiText(JSON.stringify(f.uischema, null, 2))
    setSchema(f.schema)
    setUischema(f.uischema)
    setData(f.data ?? {})
    setSchemaError(undefined)
    setUiError(undefined)
  }

  const onSchemaText = (next: string) => {
    setSchemaText(next)
    const { value, error } = parse(next)
    if (error) setSchemaError(error)
    else {
      setSchema(value as JsonSchema)
      setSchemaError(undefined)
    }
  }

  const onUiText = (next: string) => {
    setUiText(next)
    const { value, error } = parse(next)
    if (error) setUiError(error)
    else {
      setUischema(value as UISchemaElement)
      setUiError(undefined)
    }
  }

  // Remount JsonForms when switching fixtures so it doesn't hold onto stale internal state.
  const formKey = selectedId

  return (
    <Flex align="start" style={{ minHeight: '100vh' }}>
      {/* Sidebar: one entry per component fixture */}
      <Box
        p="3"
        style={{ width: 200, flexShrink: 0, borderRight: '1px solid var(--gray-5)', position: 'sticky', top: 0 }}
      >
        <Heading size="3" mb="2">
          Components
        </Heading>
        <Flex direction="column" gap="1">
          {fixtures.map((f) => {
            const active = f.id === selectedId
            return (
              <Text
                key={f.id}
                as="div"
                size="2"
                onClick={() => loadFixture(f)}
                style={{
                  cursor: 'pointer',
                  padding: '6px 8px',
                  borderRadius: 6,
                  background: active ? 'var(--accent-4)' : 'transparent',
                  fontWeight: active ? 600 : 400,
                }}
              >
                {f.name}
              </Text>
            )
          })}
        </Flex>
      </Box>

      {/* Main area */}
      <Box p="5" style={{ flex: 1 }}>
        <Heading mb="1">JSONForms renderers — playground</Heading>
        <Text size="2" color="gray">
          Pick a component on the left, then edit its Schema or UI Schema below — the form updates in real time.
        </Text>
        <Separator my="4" size="4" />

        <Flex gap="4" align="start" wrap="wrap">
          {/* Rendered form + live data */}
          <Flex direction="column" gap="4" style={{ flex: '1 1 360px' }}>
            <Card>
              <Text size="2" weight="bold" mb="2" as="div">
                Rendered form
              </Text>
              <JsonForms
                key={formKey}
                schema={schema}
                uischema={uischema}
                data={data}
                renderers={radixRenderers}
                ajv={ajv}
                onChange={({ data }) => setData(data as Record<string, unknown>)}
                validationMode="ValidateAndShow"
              />
            </Card>
            <Card>
              <Text size="2" weight="bold" as="div">
                Live data
              </Text>
              <pre style={{ margin: 0, fontSize: 13, whiteSpace: 'pre-wrap' }}>{JSON.stringify(data, null, 2)}</pre>
            </Card>
          </Flex>

          {/* Editors */}
          <Flex direction="column" gap="4" style={{ flex: '1 1 360px' }}>
            <JsonEditor label="Schema" text={schemaText} error={schemaError} onChange={onSchemaText} />
            <JsonEditor label="UI Schema" text={uiText} error={uiError} onChange={onUiText} />
          </Flex>
        </Flex>
      </Box>
    </Flex>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Theme>
      <Playground />
    </Theme>
  </StrictMode>,
)
