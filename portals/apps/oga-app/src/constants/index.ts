import { getRequiredEnv } from '../runtimeConfig'

export const API_BASE_URL = getRequiredEnv('VITE_API_BASE_URL')