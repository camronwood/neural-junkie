import { describe, expect, it } from 'vitest';
import { sanitizeMermaidSvg } from './mermaidSvgSanitize';

describe('sanitizeMermaidSvg', () => {
  it('strips script tags from SVG', () => {
    const dirty = '<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><text>ok</text></svg>';
    const clean = sanitizeMermaidSvg(dirty);
    expect(clean).not.toContain('<script');
    expect(clean).toContain('ok');
  });

  it('preserves basic svg structure', () => {
    const svg = '<svg xmlns="http://www.w3.org/2000/svg"><rect width="10" height="10"/></svg>';
    expect(sanitizeMermaidSvg(svg)).toContain('<rect');
  });
});
