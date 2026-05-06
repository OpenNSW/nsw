import { validateConfig } from './configs/types'
import type { UIConfig } from './configs/types'

declare const __BRANDING_CONFIG__: unknown

function loadConfig(): UIConfig {
  // Use runtime override if available, otherwise fallback to build-time constant
  const runtimeConfig = (window as any).__BRANDING_CONFIG__
  const config = runtimeConfig || __BRANDING_CONFIG__
  return validateConfig(config, 'VITE_BRANDING_PATH')
}

export const appConfig = loadConfig()
