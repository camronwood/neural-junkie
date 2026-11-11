/** Detect sidecar / high-trust pack signals from YAML or validation errors. */

const SIDECAR_MARKERS = [
  'hub-sidecar',
  'mcp-sidecar',
  'kind: hub-sidecar',
  'kind: mcp-sidecar',
  'kind:hub-sidecar',
  'kind:mcp-sidecar',
];

export function yamlDeclaresSidecar(yaml: string): boolean {
  const lower = yaml.toLowerCase();
  return SIDECAR_MARKERS.some((m) => lower.includes(m));
}

export function validationMentionsOfficialCollision(messages: string[]): boolean {
  return messages.some((m) => /official catalog pack/i.test(m));
}

export const PACK_SIDECAR_TRUST_COPY =
  'Enabling a pack with hub or MCP sidecars runs local code as your OS user. Official pack ids are reserved — sideload zips that reuse them are rejected. Installing a zip overwrites ~/.neural-junkie/packs/<id>/.';
