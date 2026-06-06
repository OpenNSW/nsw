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
import { TextField, Text, Flex, Box, Popover, Button } from '@radix-ui/themes'
import { CalendarIcon } from '@radix-ui/react-icons'
import { useState, type ReactNode } from 'react'
import { DayPicker } from 'react-day-picker'
import 'react-day-picker/style.css'
import dayjs from 'dayjs'

const toISODate = (d: Date) => dayjs(d).format('YYYY-MM-DD')

// Parse with an explicit midnight so the string is read as local time — a bare
// 'YYYY-MM-DD' is parsed as UTC and can shift the picked day by one.
const fromISODate = (s?: string) => {
  if (!s) return undefined
  const d = dayjs(`${s}T00:00:00`)
  return d.isValid() ? d.toDate() : undefined
}

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
        {!isValid && errors !== 'is a required property' && (
          <Text color="red" size="1">
            {errors}
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

export const DateControl = ({ data, handleChange, path, label, required, errors, schema, enabled }: ControlProps) => {
  const isValid = errors.length === 0
  const [open, setOpen] = useState(false)
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

  // date and date-time share the calendar; date-time appends a native time input.
  const hasTime = schema.format === 'date-time'
  const [datePart = '', timeRaw = ''] = value.split('T')
  const selected = fromISODate(datePart)
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
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Trigger>
          <Button id={path} type="button" variant="surface" color={!isValid ? 'red' : 'gray'} disabled={!enabled}>
            <CalendarIcon />
            {selected ? selected.toLocaleDateString() : 'Select date'}
          </Button>
        </Popover.Trigger>
        <Popover.Content>
          <DayPicker
            mode="single"
            captionLayout="dropdown"
            selected={selected}
            defaultMonth={selected}
            onSelect={(d) => {
              commit(d ? toISODate(d) : '', timeForInput)
              if (d) setOpen(false)
            }}
          />
        </Popover.Content>
      </Popover.Root>
      {hasTime && (
        <TextField.Root
          type="time"
          value={timeForInput}
          disabled={!enabled || !datePart}
          color={!isValid ? 'red' : undefined}
          onChange={(e) => commit(datePart, e.target.value)}
        />
      )}
    </Flex>,
  )
}

export const DateControlTester: RankedTester = rankWith(2, or(isDateControl, isDateTimeControl, isTimeControl))

export default withJsonFormsControlProps(DateControl)
