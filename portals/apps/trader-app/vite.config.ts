import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import * as path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import fs from 'node:fs'
import type { Plugin } from 'vite'

function deepMerge(base: Record<string, unknown>, override: Record<string, unknown>): Record<string, unknown> {
  const result = { ...base }
  for (const key of Object.keys(override)) {
    if (
      override[key] &&
      typeof override[key] === 'object' &&
      !Array.isArray(override[key]) &&
      base[key] &&
      typeof base[key] === 'object' &&
      !Array.isArray(base[key])
    ) {
      result[key] = deepMerge(base[key] as Record<string, unknown>, override[key] as Record<string, unknown>)
    } else {
      result[key] = override[key]
    }
  }
  return result
}

function brandingConfigPlugin(): Plugin {
  const defaultJsonPath = path.resolve(import.meta.dirname, 'src/configs/default.json')
  const brandingPath = process.env.VITE_BRANDING_PATH
  const customJsonPath = brandingPath ? path.resolve(import.meta.dirname, brandingPath) : null
  const watchedPaths = [defaultJsonPath, ...(customJsonPath ? [customJsonPath] : [])]

  function readJson(filePath: string): Record<string, unknown> {
    const content = fs.readFileSync(filePath, 'utf8')
    try {
      return JSON.parse(content) as Record<string, unknown>
    } catch {
      throw new Error(
        `[branding-config] "${filePath}" is not valid JSON.\n` +
          `Content preview: ${content.slice(0, 120)}\n` +
          `Tip: if VITE_BRANDING_PATH still points to a .yaml file, update it to .json in your .env file.`,
      )
    }
  }

  function loadMerged(): Record<string, unknown> {
    console.log(`[branding-config] default: ${defaultJsonPath}`)
    if (customJsonPath) console.log(`[branding-config] custom:  ${customJsonPath}`)
    let merged = readJson(defaultJsonPath)
    if (customJsonPath && fs.existsSync(customJsonPath)) {
      merged = deepMerge(merged, readJson(customJsonPath))
    }
    return merged
  }

  let currentConfig = loadMerged()

  return {
    name: 'branding-config',
    transform(code) {
      if (!code.includes('__BRANDING_CONFIG__')) return null
      return {
        code: code.replace(/__BRANDING_CONFIG__/g, () => JSON.stringify(currentConfig)),
        map: null,
      }
    },
    configureServer(server) {
      server.watcher.add(watchedPaths)
      server.watcher.on('change', (file) => {
        if (!watchedPaths.map(path.normalize).includes(path.normalize(file))) return
        currentConfig = loadMerged()
        server.moduleGraph.invalidateAll()
        server.ws.send({ type: 'full-reload' })
      })
    },
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss(), brandingConfigPlugin()],
  resolve: {
    alias: {
      '@opennsw/ui': path.resolve(import.meta.dirname, '../../packages/ui/src'),
      '@opennsw/jsonforms-renderers': path.resolve(import.meta.dirname, '../../packages/jsonforms-renderers/src'),
    },
  },
})
