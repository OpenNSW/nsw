type RuntimeConfigValue = string | undefined

type RuntimeConfigMap = Record<string, RuntimeConfigValue>

export interface IdpRoleGroupConfig {
  traderGroupName: string
  chaGroupName: string
}

const DEFAULT_TRADER_GROUP_NAME = 'Traders'
const DEFAULT_CHA_GROUP_NAME = 'CHA'

declare global {
  interface Window {
    __APP_CONFIG__?: RuntimeConfigMap
  }
}

function resolveRuntimeConfig(): RuntimeConfigMap {
  if (typeof window === 'undefined') {
    return {}
  }

  return window.__APP_CONFIG__ ?? {}
}

export function getEnv(name: string, fallback?: string): string | undefined {
  const runtimeValue = resolveRuntimeConfig()[name]
  if (runtimeValue && runtimeValue.trim() !== '') {
    return runtimeValue
  }

  const buildValue = (import.meta.env as Record<string, string | undefined>)[name]
  if (buildValue && buildValue.trim() !== '') {
    return buildValue
  }

  return fallback
}

export function getBooleanEnv(name: string, fallback = false): boolean {
  const value = getEnv(name)
  if (!value) {
    return fallback
  }

  return value.toLowerCase() === 'true'
}

export function getIdpRoleGroupConfig(): IdpRoleGroupConfig {
  return {
    traderGroupName:
      getEnv('VITE_IDP_TRADER_GROUP_NAME', DEFAULT_TRADER_GROUP_NAME) ?? DEFAULT_TRADER_GROUP_NAME,
    chaGroupName: getEnv('VITE_IDP_CHA_GROUP_NAME', DEFAULT_CHA_GROUP_NAME) ?? DEFAULT_CHA_GROUP_NAME,
  }
}
