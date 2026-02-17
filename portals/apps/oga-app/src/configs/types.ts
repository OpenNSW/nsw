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