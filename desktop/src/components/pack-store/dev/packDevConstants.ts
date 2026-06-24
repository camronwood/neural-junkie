/** Overlay keys for generic custom packs created in Pack dev studio (workspace/SOP packs). */
export const GENERIC_OVERLAY_FIELD_DOCS: Array<{ key: string; hint: string }> = [];

/**
 * Overlay keys for custom packs with `secondary-analysis-*` capabilities — edit pack.yaml
 * in your private pack repo (e.g. brightest-bio-lab), not the generic scaffold wizard.
 */
export const PRIVATE_SECONDARY_OVERLAY_KEYS = [
  'secondary_analysis_tools_path',
  'python_executable',
  'cumulative_qc_dir',
  'default_panel_profile',
] as const;

export const MANIFEST_FIELD_HINTS = [
  'id — unique slug (lowercase, hyphens)',
  'pack_kind: customer — marks a private custom pack (internal token; UI says “custom pack”)',
  'capabilities — include customer-pack for generic workspace/SOP packs',
  'requires_packs — domain packs that must be installed and enabled first',
  'settings_overlay — optional; generic packs usually omit biology tool paths',
  'assets.workspace_guide — markdown SOP shown in test panel',
  'capability_defs.<id>.ui.toolbar — sidebar chip: label (≤3 letters) and/or icon (pack-relative path)',
  'capability_defs.<id>.ui.modal — desktop modal id opened when the chip is clicked (e.g. phoenix-import in brightest-bio-lab)',
  'Advanced lab capabilities (phoenix-import, secondary-analysis-*, scan viewers) belong in your org pack repo — see brightest-bio-lab',
];
