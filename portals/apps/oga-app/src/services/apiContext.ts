import { createContext } from 'react'
import type { ApiClient } from '../api'

export const ApiContext = createContext<ApiClient | null>(null)
