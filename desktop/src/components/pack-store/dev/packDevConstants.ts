export const OVERLAY_FIELD_DOCS: Array<{ key: string; hint: string }> = [
  { key: 'secondary_analysis_tools_path', hint: 'Pack-relative path to secondary analysis tools' },
  { key: 'python_executable', hint: 'Python binary for biology tools (e.g. python3)' },
  { key: 'cumulative_qc_dir', hint: 'Pack-relative cumulative QC directory' },
  { key: 'default_panel_profile', hint: 'Default biology panel profile name' },
  { key: 'artifacts_dir', hint: 'Pack-relative artifacts directory' },
];

export const MANIFEST_FIELD_HINTS = [
  'id — unique slug (lowercase, hyphens)',
  'pack_kind: customer — marks a private customer pack',
  'capabilities — include customer-pack for generic data/SOP packs',
  'requires_packs — domain packs that must be installed and enabled first',
  'settings_overlay — biology tool defaults applied on enable',
  'assets.workspace_guide — markdown SOP shown in test panel',
  'Customer-specific capabilities (e.g. phoenix-import) belong in that org’s private pack repo — not the dev studio scaffold',
];
