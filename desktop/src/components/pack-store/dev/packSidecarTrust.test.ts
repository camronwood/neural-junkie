import { describe, expect, it } from 'vitest';
import {
  PACK_SIDECAR_TRUST_COPY,
  validationMentionsOfficialCollision,
  yamlDeclaresSidecar,
} from './packSidecarTrust';

describe('packSidecarTrust', () => {
  it('detects hub-sidecar and mcp-sidecar in yaml', () => {
    expect(yamlDeclaresSidecar('capabilities:\n  - hub-sidecar')).toBe(true);
    expect(yamlDeclaresSidecar('kind: mcp-sidecar')).toBe(true);
    expect(yamlDeclaresSidecar('pack_kind: customer\nid: my-lab')).toBe(false);
  });

  it('detects official catalog collision messages', () => {
    expect(
      validationMentionsOfficialCollision([
        'pack id "life-sciences" collides with an official catalog pack',
      ]),
    ).toBe(true);
    expect(validationMentionsOfficialCollision(['ok'])).toBe(false);
  });

  it('exports non-empty trust copy', () => {
    expect(PACK_SIDECAR_TRUST_COPY).toMatch(/OS user/);
    expect(PACK_SIDECAR_TRUST_COPY).toMatch(/Official pack ids/);
  });
});
