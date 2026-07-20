import { describe, expect, it } from 'vitest';
import { isExternalHttpUrl } from './openExternalLink';

describe('openExternalLink helpers', () => {
  it('accepts http(s) urls', () => {
    expect(isExternalHttpUrl('https://example.com/docs')).toBe(true);
    expect(isExternalHttpUrl('http://localhost:3000')).toBe(true);
  });

  it('rejects non-http schemes', () => {
    expect(isExternalHttpUrl('javascript:alert(1)')).toBe(false);
    expect(isExternalHttpUrl('data:text/html,hi')).toBe(false);
    expect(isExternalHttpUrl('/relative')).toBe(false);
  });
});
