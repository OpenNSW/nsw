import { parse as parseYaml } from 'yaml';
import { validateConfig } from './configs/types';
import type { UIConfig } from './configs/types';
import { getEnv } from './runtimeConfig';

const yamlModules = import.meta.glob<string>(
  './configs/*.yaml',
  { query: '?raw', import: 'default', eager: true }
);

function loadConfig(): UIConfig {
  const instance = getEnv('VITE_INSTANCE_CONFIG');

  if (!instance) {
    throw new Error(
      'VITE_INSTANCE_CONFIG environment variable is not set. ' +
      'Please specify which instance to load (e.g., npqs, fcau, ird)'
    );
  }

  // Load and parse default config
  const defaultYaml = yamlModules['./configs/default.yaml'];
  const defaults = defaultYaml ? parseYaml(defaultYaml) as Record<string, unknown> : {};

  // Load and parse instance config
  const instancePath = `./configs/${instance}.yaml`;
  const instanceYaml = yamlModules[instancePath];

  if (!instanceYaml) {
    throw new Error(
      `Config not found for instance: ${instance}. ` +
      `Available: ${Object.keys(yamlModules).filter(k => k !== './configs/default.yaml' && k !== './configs/example.yaml').join(', ')}`
    );
  }

  const instanceConfig = parseYaml(instanceYaml) as Record<string, unknown>;

  // Deep merge: instance overrides defaults
  const merged = deepMerge(defaults, instanceConfig);
  return validateConfig(merged, instance);
}

function deepMerge(base: Record<string, unknown>, override: Record<string, unknown>): Record<string, unknown> {
  const result = { ...base };
  for (const key of Object.keys(override)) {
    if (
      override[key] && typeof override[key] === 'object' && !Array.isArray(override[key]) &&
      base[key] && typeof base[key] === 'object' && !Array.isArray(base[key])
    ) {
      result[key] = deepMerge(base[key] as Record<string, unknown>, override[key] as Record<string, unknown>);
    } else {
      result[key] = override[key];
    }
  }
  return result;
}

export const appConfig = loadConfig();