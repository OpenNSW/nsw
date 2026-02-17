import type {UIConfig} from "./configs/types.ts";


const configModules = import.meta.glob<{ config: UIConfig }>(
  './configs/*.config.ts',
  { eager: true }
);

function loadConfig(): UIConfig {
  const instance = import.meta.env.VITE_INSTANCE_CONFIG as string;

  if (!instance) {
    throw new Error(
      'VITE_INSTANCE environment variable is not set. ' +
      'Please specify which instance to build (e.g., client-a, client-b)'
    );
  }

  const configPath = `./configs/${instance}.config.ts`;


  const configModule = configModules[configPath];

  if (!configModule) {
    throw new Error(
      `Config not found for environment: ${instance}. Available: ${Object.keys(configModules).join(', ')}`
    );
  }

  return configModule.config;
}

export const appConfig = loadConfig();