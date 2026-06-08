/** Overlay keys for generic customer packs created in Pack dev studio (workspace/SOP packs). */
export const GENERIC_OVERLAY_FIELD_DOCS: Array<{ key: string; hint: string }> = [];

/**
 * Overlay keys for customer packs with `secondary-analysis-*` capabilities — edit pack.yaml
 * in the customer repo, not the generic scaffold wizard.
 */
export const PRIVATE_SECONDARY_OVERLAY_KEYS = [
  'secondary_analysis_tools_path',
  'python_executable',
  'cumulative_qc_dir',
  'default_panel_profile',
] as const;

export const MANIFEST_FIELD_HINTS = [
  'id — unique slug (lowercase, hyphens)',
  'pack_kind: customer — marks a private customer pack',
  'capabilities — include customer-pack for generic data/SOP packs',
  'requires_packs — domain packs that must be installed and enabled first',
  'settings_overlay — optional; generic packs usually omit biology tool paths',
  'assets.workspace_guide — markdown SOP shown in test panel',
  'Customer pack: phoenix-import, secondary-analysis-api/viewer/python + overlay keys (tools path, python, panel profile, cumulative_qc_dir)',
];
