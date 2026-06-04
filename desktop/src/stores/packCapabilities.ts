/** Capability tokens declared in pack manifests (hub + desktop). */
export const PACK_CAP = {
  IDE_V2: 'ide-v2',
  IDE_V3_COMPOSER: 'ide-v3-composer',
  GIT_REST: 'git-rest',
  INLINE_COMPLETION: 'inline-completion',
  SCAN_SUMMARY_API: 'scan-summary-api',
  SCAN_SUMMARY_VIEWER: 'scan-summary-viewer',
  SCAN_ANALYSIS_VIEWER: 'scan-analysis-viewer',
  SECONDARY_ANALYSIS_API: 'secondary-analysis-api',
  SECONDARY_ANALYSIS_VIEWER: 'secondary-analysis-viewer',
  SECONDARY_ANALYSIS_PYTHON: 'secondary-analysis-python',
  CAD_API: 'cad-api',
  CAD_VIEWER: 'cad-viewer',
  CAD_WORKBENCH: 'cad-workbench',
  LORA_TRAINING: 'lora-training',
  LORA_COMPOSE: 'lora-compose',
  LORA_ADAPTERS: 'lora-adapters',
  PERSONAL_LEARNING: 'personal-learning',
  CUSTOMER_PACK: 'customer-pack',
  PHOENIX_IMPORT: 'phoenix-import',
} as const;

export type PackCapability = (typeof PACK_CAP)[keyof typeof PACK_CAP];
