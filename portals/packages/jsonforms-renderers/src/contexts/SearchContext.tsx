import { createContext, useContext, useMemo, type ReactNode } from 'react'

export type GetSearchHeaders = () => Promise<HeadersInit>

export interface SearchContextValue {
  baseUrl: string
  getHeaders: GetSearchHeaders
}

const SearchContext = createContext<SearchContextValue | null>(null)

export function SearchProvider({
  children,
  baseUrl,
  getHeaders,
}: {
  children: ReactNode
  baseUrl: string
  getHeaders: GetSearchHeaders
}) {
  const value = useMemo(() => ({ baseUrl, getHeaders }), [baseUrl, getHeaders])
  return <SearchContext.Provider value={value}>{children}</SearchContext.Provider>
}

export function useSearchContext(): SearchContextValue | null {
  return useContext(SearchContext)
}
