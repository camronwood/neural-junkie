/** Capability tokens declared in pack manifests (hub + desktop). */
export const PACK_CAP = {
  IDE_V2: 'ide-v2',
  IDE_V3_COMPOSER: 'ide-v3-composer',
  GIT_REST: 'git-rest',
  INLINE_COMPLETION: 'inline-completion',
  SCAN_SUMMARY_API: 'scan-summary-api',
  SCAN_SUMMARY_VIEWER: 'scan-summary-viewer',
  SCAN_ANALYSIS_VIEWER: 'scan-analysis-viewer',
} as const;

export type PackCapability = (typeof PACK_CAP)[keyof typeof PACK_CAP];
