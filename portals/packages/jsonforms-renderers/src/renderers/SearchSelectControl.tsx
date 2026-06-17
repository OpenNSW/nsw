import { type ControlProps, type JsonSchema } from '@jsonforms/core'
import { withJsonFormsControlProps } from '@jsonforms/react'
import { Box, Button, Flex, ScrollArea, Spinner, Text, TextField } from '@radix-ui/themes'
import { ChevronDownIcon } from '@radix-ui/react-icons'
import { useState, useRef, useEffect, useCallback, type KeyboardEvent } from 'react'
import { useSearchContext } from '../contexts/SearchContext'
import { useClearWhenHidden } from '../hooks/useClearWhenHidden'
import { getErrorMessage } from '../utils/error'
import * as React from 'react'

function resolvePath(obj: unknown, path: string): unknown {
  if (!path) return obj
  return path.split('.').reduce<unknown>((acc, key) => {
    if (acc == null || typeof acc !== 'object') return undefined
    return (acc as Record<string, unknown>)[key]
  }, obj)
}

function buildUrl(baseUrl: string, apiPath: string, params: Record<string, string | undefined>): string {
  const url = new URL(apiPath, baseUrl)
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== '') url.searchParams.set(k, v)
  })
  return url.toString()
}

interface SearchOption {
  value: string
  label: string
}

function parseItems(json: unknown, itemsPath: string, valueKey: string, labelKey: string): SearchOption[] {
  const raw = resolvePath(json, itemsPath)
  if (!Array.isArray(raw)) {
    throw new Error(`Response path "${itemsPath}" did not resolve to an array`)
  }
  return raw.map((item) => {
    const value = resolvePath(item, valueKey)
    const label = resolvePath(item, labelKey)
    return {
      value: String(value ?? ''),
      label: String(label ?? value ?? ''),
    }
  })
}

// --- x-search schema config ---

interface XSearchOptions {
  path: string
  valueKey?: string
  labelKey?: string
  pagination?: 'none' | 'offset' | 'cursor'
  pageSize?: number
  itemsPath?: string
  nextCursorPath?: string
  searchParam?: string
  limitParam?: string
  offsetParam?: string
  cursorParam?: string
  loadOnOpen?: boolean // Whether to load options when the dropdown is opened
}

type SearchSelectProps = ControlProps & {
  schema: JsonSchema & { 'x-search'?: XSearchOptions }
}

const SearchSelectControl = ({
  data,
  handleChange,
  path,
  label,
  required,
  errors,
  enabled,
  visible = true,
  schema,
  uischema,
}: SearchSelectProps) => {
  const ctx = useSearchContext()

  const xSearch: XSearchOptions = ((schema as Record<string, unknown>)?.['x-search'] as XSearchOptions) ?? {
    path: '',
  }
  const apiPath = xSearch.path ?? ''
  const pagination = xSearch.pagination ?? 'offset'
  const pageSize = xSearch.pageSize ?? 5
  const valueKey = xSearch.valueKey ?? 'id'
  const labelKey = xSearch.labelKey ?? 'name'
  const searchParam = xSearch.searchParam !== undefined ? xSearch.searchParam : 'q'
  const limitParam = xSearch.limitParam ?? 'limit'
  const offsetParam = xSearch.offsetParam ?? 'offset'
  const cursorParam = xSearch.cursorParam ?? 'cursor'
  const loadOnOpen = xSearch.loadOnOpen ?? false
  const defaultItemsPath = pagination === 'none' ? '' : 'items'
  const itemsPath = xSearch.itemsPath !== undefined ? xSearch.itemsPath : defaultItemsPath
  const nextCursorPath = xSearch.nextCursorPath ?? 'nextCursor'

  const isEnabled = enabled !== false
  const isValid = !errors || errors.length === 0

  const [open, setOpen] = useState(false)
  const [inputValue, setInputValue] = useState('')
  const [options, setOptions] = useState<SearchOption[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedOption, setSelectedOption] = useState<SearchOption | undefined>(undefined)

  const cursorRef = useRef<string | undefined>(undefined)
  const offsetRef = useRef(0)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!data) {
      setSelectedOption(undefined)
      return
    }
    if (selectedOption?.value === data) return
    setSelectedOption({ value: data as string, label: data as string })
  }, [data, selectedOption?.value])

  const runSearch = useCallback(
    async (q: string, isLoadMore = false) => {
      if (!apiPath) return
      if (!ctx) {
        setError('Search service not configured.')
        return
      }

      abortRef.current?.abort()
      const controller = new AbortController()
      abortRef.current = controller

      if (isLoadMore) setLoadingMore(true)
      else {
        setLoading(true)
        setLoadingMore(false)
        setError(null)
      }

      try {
        const params: Record<string, string | undefined> = {}
        if (searchParam) params[searchParam] = q
        if (pagination !== 'none') {
          params[limitParam] = String(pageSize)
          if (pagination === 'offset') {
            params[offsetParam] = String(isLoadMore ? offsetRef.current : 0)
          } else if (pagination === 'cursor' && isLoadMore && cursorRef.current) {
            params[cursorParam] = cursorRef.current
          }
        }

        const url = buildUrl(ctx.baseUrl, apiPath, params)
        const headers = await ctx.getHeaders()
        const res = await fetch(url, { headers, signal: controller.signal })
        if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
        const json: unknown = await res.json()

        const newItems = parseItems(json, itemsPath, valueKey, labelKey)

        if (isLoadMore) {
          setOptions((prev) => [...prev, ...newItems])
          offsetRef.current += newItems.length
        } else {
          setOptions(newItems)
          offsetRef.current = newItems.length
          cursorRef.current = undefined
        }

        if (pagination === 'none') {
          setHasMore(false)
        } else if (pagination === 'cursor') {
          const next = resolvePath(json, nextCursorPath)
          cursorRef.current = typeof next === 'string' && next ? next : undefined
          setHasMore(typeof next === 'string' && next !== '')
        } else {
          setHasMore(newItems.length === pageSize)
        }
      } catch (e) {
        if (e instanceof DOMException && e.name === 'AbortError') return
        setError('Failed to load results. Please try again.')
      } finally {
        if (!controller.signal.aborted) {
          if (isLoadMore) setLoadingMore(false)
          else setLoading(false)
        }
      }
    },
    [
      ctx,
      apiPath,
      pagination,
      pageSize,
      valueKey,
      labelKey,
      searchParam,
      limitParam,
      offsetParam,
      cursorParam,
      itemsPath,
      nextCursorPath,
    ],
  )

  // Close dropdown on outside click.
  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  // Clear list state and abort in-flight request when dropdown closes.
  useEffect(() => {
    if (!open) {
      abortRef.current?.abort()
      setInputValue('')
      setOptions([])
      setHasMore(false)
      setError(null)
      setLoading(false)
      setLoadingMore(false)
    }
  }, [open])

  // Debounced search — fires on input change or immediately on open when loadOnOpen is set.
  useEffect(() => {
    if (!open) return

    if (!inputValue && !loadOnOpen) {
      setOptions([])
      setHasMore(false)
      setError(null)
      offsetRef.current = 0
      cursorRef.current = undefined
      return
    }

    if (debounceRef.current) clearTimeout(debounceRef.current)
    const delay = inputValue ? 300 : 0
    debounceRef.current = setTimeout(() => {
      setOptions([])
      setHasMore(false)
      offsetRef.current = 0
      cursorRef.current = undefined
      void runSearch(inputValue)
    }, delay)

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [inputValue, open, loadOnOpen, runSearch])

  useClearWhenHidden(visible, path, handleChange, null)

  if (visible === false) {
    return null
  }

  const openDropdown = () => {
    setInputValue('')
    setOptions([])
    setHasMore(false)
    setError(null)
    offsetRef.current = 0
    cursorRef.current = undefined
    setOpen(true)
  }

  const onSelect = (option: SearchOption) => {
    handleChange(path, option.value)
    setSelectedOption(option)
    setOpen(false)
  }

  const onClear = (e: React.SyntheticEvent) => {
    e.stopPropagation()
    handleChange(path, null)
    setSelectedOption(undefined)
  }

  const onClearKeyDown = (e: KeyboardEvent<HTMLSpanElement>) => {
    if (e.key === 'Enter' || e.key === ' ') onClear(e)
  }

  const placeholder = (uischema?.options?.placeholder as string | undefined) ?? 'Select an option'

  return (
    <Box mb="4">
      <Flex direction="column" gap="1">
        <Text as="label" size="2" weight="bold" htmlFor={path}>
          {label} {required && <Text color="red">*</Text>}
        </Text>

        <div ref={containerRef} style={{ position: 'relative' }}>
          <TextField.Root
            id={path}
            value={open ? inputValue : (selectedOption?.label ?? '')}
            placeholder={open ? 'Search...' : placeholder}
            disabled={!isEnabled}
            style={!isValid ? { outline: '2px solid var(--red-7)', outlineOffset: '-1px' } : undefined}
            onChange={(e) => {
              setInputValue(e.target.value)
              if (!open) openDropdown()
            }}
            onFocus={() => {
              if (isEnabled && !open) openDropdown()
            }}
            onKeyDown={(e) => {
              if (e.key === 'Escape') setOpen(false)
            }}
          >
            <TextField.Slot side="right">
              <Flex align="center" gap="1">
                {data && isEnabled && (
                  <span
                    role="button"
                    tabIndex={0}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={onClear}
                    onKeyDown={onClearKeyDown}
                    style={{ lineHeight: 1, color: 'var(--gray-9)', padding: '0 2px', cursor: 'pointer' }}
                    aria-label="Clear selection"
                  >
                    ×
                  </span>
                )}
                <ChevronDownIcon
                  style={{ transform: open ? 'rotate(180deg)' : undefined, transition: 'transform 150ms' }}
                />
              </Flex>
            </TextField.Slot>
          </TextField.Root>

          {open && (
            <Box
              style={{
                position: 'absolute',
                top: 'calc(100% + 4px)',
                left: 0,
                right: 0,
                zIndex: 50,
                background: 'var(--color-panel-solid)',
                borderRadius: 'var(--radius-3)',
                boxShadow: 'var(--shadow-5)',
                border: '1px solid var(--gray-a6)',
                overflow: 'hidden',
              }}
            >
              {!apiPath ? (
                <Box p="3">
                  <Text size="2" color="gray">
                    Search not configured.
                  </Text>
                </Box>
              ) : (
                <ScrollArea style={{ maxHeight: 240 }}>
                  <Box p="1">
                    {loading && (
                      <Flex justify="center" p="4">
                        <Spinner />
                      </Flex>
                    )}

                    {!loading && error && (
                      <Box px="3" py="2">
                        <Text size="2" color="red">
                          {error}
                        </Text>
                      </Box>
                    )}

                    {!loading && !error && options.length === 0 && (
                      <Box px="3" py="2">
                        <Text size="2" color="gray">
                          {inputValue || loadOnOpen ? 'No results found.' : 'Type to search…'}
                        </Text>
                      </Box>
                    )}

                    {options.map((opt) => (
                      <Box
                        key={opt.value}
                        px="3"
                        py="2"
                        onMouseDown={(e) => e.preventDefault()}
                        onClick={() => onSelect(opt)}
                        style={{
                          cursor: 'pointer',
                          borderRadius: 'var(--radius-2)',
                          backgroundColor: opt.value === data ? 'var(--accent-3)' : undefined,
                        }}
                        className="hover:bg-(--gray-3)"
                      >
                        <Text size="2">{opt.label}</Text>
                      </Box>
                    ))}

                    {hasMore && !loadingMore && (
                      <Box px="3" py="2">
                        <Button
                          variant="ghost"
                          size="1"
                          style={{ width: '100%' }}
                          onMouseDown={(e) => e.preventDefault()}
                          onClick={() => void runSearch(inputValue, true)}
                        >
                          Load more
                        </Button>
                      </Box>
                    )}

                    {loadingMore && (
                      <Flex justify="center" p="2">
                        <Spinner />
                      </Flex>
                    )}
                  </Box>
                </ScrollArea>
              )}
            </Box>
          )}
        </div>

        {!isValid && (
          <Text color="red" size="1">
            {getErrorMessage(errors, label)}
          </Text>
        )}
        {schema.description && (
          <Text size="1" color="gray">
            {schema.description}
          </Text>
        )}
      </Flex>
    </Box>
  )
}

export default withJsonFormsControlProps(SearchSelectControl)
