/** NJ platform capability tokens (official domain + customer platform). */
export const PACK_PLATFORM_CAP = {
  CUSTOMER_PACK: 'customer-pack',
  SETTINGS_OVERLAY: 'settings-overlay',
  WORKSPACE_GUIDE: 'workspace-guide',
  IDE_V2: 'ide-v2',
  IDE_V3_COMPOSER: 'ide-v3-composer',
  GIT_REST: 'git-rest',
  INLINE_COMPLETION: 'inline-completion',
  CAD_API: 'cad-api',
  CAD_VIEWER: 'cad-viewer',
  CAD_WORKBENCH: 'cad-workbench',
  LORA_TRAINING: 'lora-training',
  LORA_COMPOSE: 'lora-compose',
  LORA_ADAPTERS: 'lora-adapters',
  PERSONAL_LEARNING: 'personal-learning',
  AWS_API: 'aws-api',
  AWS_SSO: 'aws-sso',
  INCIDENT_API: 'incident-api',
  JIRA_INTEGRATION: 'jira-integration',
  INCIDENT_TRIAGE: 'incident-triage',
  WEB_BROWSER: 'web-browser',
  WEB_PREVIEW: 'web-preview',
  WEB_BROWSER_WORKBENCH: 'web-browser-workbench',
} as const;

/**
 * Common pack-local capability ids (defined in custom pack capability_defs, not NJ platform).
 * Prefer registryHasCapability / capability_registry from the hub for runtime checks.
 */
export const PACK_LOCAL_CAP = {
  PHOENIX_IMPORT: 'phoenix-import',
  SCAN_SUMMARY_API: 'scan-summary-api',
  SCAN_SUMMARY_VIEWER: 'scan-summary-viewer',
  SCAN_ANALYSIS_VIEWER: 'scan-analysis-viewer',
  SECONDARY_ANALYSIS_API: 'secondary-analysis-api',
  SECONDARY_ANALYSIS_VIEWER: 'secondary-analysis-viewer',
  SECONDARY_ANALYSIS_PYTHON: 'secondary-analysis-python',
} as const;

/** @deprecated use PACK_PLATFORM_CAP or PACK_LOCAL_CAP */
export const PACK_CAP = { ...PACK_PLATFORM_CAP, ...PACK_LOCAL_CAP } as const;

export type PackPlatformCapability = (typeof PACK_PLATFORM_CAP)[keyof typeof PACK_PLATFORM_CAP];
export type PackLocalCapability = (typeof PACK_LOCAL_CAP)[keyof typeof PACK_LOCAL_CAP];
export type PackCapability = (typeof PACK_CAP)[keyof typeof PACK_CAP];
