export interface UIConfig {
  branding: {
    appName: string;
    logoUrl?: string;
    favicon?: string;
  },
  theme?: {
    fontFamily: string;
    borderRadius: string;
  },
  features?: {
    preConsignment: boolean;
    consignmentManagement: boolean;
    reportingDashboard: boolean;
  },
}

export function validateConfig(parsed: unknown, instanceId: string): UIConfig {
  if (!parsed || typeof parsed !== 'object') {
    throw new Error(`Config for "${instanceId}" is not a valid object`);
  }

  const obj = parsed as Record<string, unknown>;

  if (!obj.branding || typeof obj.branding !== 'object') {
    throw new Error(`Config for "${instanceId}" is missing required "branding" section`);
  }

  const branding = obj.branding as Record<string, unknown>;

  if (typeof branding.appName !== 'string' || branding.appName.trim() === '') {
    throw new Error(`Config for "${instanceId}": branding.appName must be a non-empty string`);
  }

  return {
    branding: {
      appName: branding.appName,
      logoUrl: typeof branding.logoUrl === 'string' ? branding.logoUrl : undefined,
      favicon: typeof branding.favicon === 'string' ? branding.favicon : undefined,
    },
    theme: obj.theme as UIConfig['theme'],
    features: obj.features as UIConfig['features'],
  };
}