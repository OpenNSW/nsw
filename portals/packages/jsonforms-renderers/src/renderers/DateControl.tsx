import {
  type ControlProps,
  isDateControl,
  isDateTimeControl,
  isTimeControl,
  or,
  type RankedTester,
  rankWith,
} from '@jsonforms/core'
import { withJsonFormsControlProps } from '@jsonforms/react'
import { TextField, Text, Flex, Box } from '@radix-ui/themes'
import { useEffect, type ReactNode } from 'react'
import dayjs from 'dayjs'

import { getErrorMessage } from '../utils/error'

// dayjs().format() defaults to RFC 3339 (e.g. 2026-06-05T12:30:00+05:30) with
// seconds + local offset, which ajv's strict "date-time" format check requires.
const toISODateTime = (dateStr: string, timeStr: string) => dayjs(`${dateStr}T${timeStr}`).format()

type ShellProps = Pick<ControlProps, 'path' | 'label' | 'required' | 'errors'> & {
  description?: string
  children: ReactNode
}

const FieldShell = ({ path, label, required, errors, description, children }: ShellProps) => {
  const isValid = errors.length === 0
  return (
    <Box mb="4">
      <Flex direction="column" gap="1">
        <Text as="label" size="2" weight="bold" htmlFor={path}>
          {label} {required && <Text color="red">*</Text>}
        </Text>
        {children}
        {!isValid && (
          <Text color="red" size="1">
            {getErrorMessage(errors, label)}
          </Text>
        )}
        {description && (
          <Text size="1" color="gray">
            {description}
          </Text>
        )}
      </Flex>
    </Box>
  )
}

export const DateControl = ({
  data,
  handleChange,
  path,
  label,
  required,
  errors,
  schema,
  enabled,
  visible = true,
}: ControlProps) => {
  useEffect(() => {
    if (visible === false) {
      handleChange(path, undefined)
    }
  }, [visible, path, handleChange])

  if (visible === false) {
    return null
  }
  const isValid = errors.length === 0
  const value: string = typeof data === 'string' ? data : ''

  const shell = (children: ReactNode) => (
    <FieldShell path={path} label={label} required={required} errors={errors} description={schema.description}>
      {children}
    </FieldShell>
  )

  // Time-only: native picker is plenty.
  if (schema.format === 'time') {
    return shell(
      <TextField.Root
        type="time"
        value={value}
        onChange={(e) => handleChange(path, e.target.value)}
        disabled={!enabled}
        color={!isValid ? 'red' : undefined}
        id={path}
      />,
    )
  }

  // date and date-time share the same date input; date-time appends a native time input.
  const hasTime = schema.format === 'date-time'
  const [datePart = '', timeRaw = ''] = value.split('T')
  // The native <input type="time"> only understands HH:MM(:SS); strip any
  // timezone suffix from the stored RFC 3339 value before feeding it back.
  const timeForInput = timeRaw.slice(0, 5)

  const commit = (nextDate: string, nextTime: string) => {
    if (!nextDate) {
      handleChange(path, undefined)
      return
    }
    handleChange(path, hasTime ? toISODateTime(nextDate, nextTime || '00:00') : nextDate)
  }

  return shell(
    <Flex gap="2" align="center">
      <TextField.Root
        type="date"
        value={datePart}
        disabled={!enabled}
        color={!isValid ? 'red' : undefined}
        onChange={(e) => commit(e.target.value, timeForInput)}
        id={path}
        style={{ flex: 1 }}
      />
      {hasTime && (
        <TextField.Root
          type="time"
          value={timeForInput}
          disabled={!enabled || !datePart}
          color={!isValid ? 'red' : undefined}
          onChange={(e) => commit(datePart, e.target.value)}
          style={{ flex: 1 }}
        />
      )}
    </Flex>,
  )
}

export const DateControlTester: RankedTester = rankWith(2, or(isDateControl, isDateTimeControl, isTimeControl))

const JsonFormsDateControl = withJsonFormsControlProps(DateControl)
export default JsonFormsDateControl
