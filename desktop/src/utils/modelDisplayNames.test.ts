import { describe, expect, it } from 'vitest';
import { formatModelDisplayName, formatModelWithRole } from './modelDisplayNames';

describe('formatModelDisplayName', () => {
  it('labels OpenBio hub tag', () => {
    expect(formatModelDisplayName('koesn/llama3-openbiollm-8b:latest')).toBe('OpenBioLLM 8B');
  });

  it('labels Qwen tool model', () => {
    expect(formatModelDisplayName('qwen2.5:7b')).toBe('Qwen 2.5 7B');
  });

  it('labels Ornith import tag', () => {
    expect(formatModelDisplayName('nj-ornith:9b')).toBe('Ornith 1.0 9B');
  });

  it('falls back to raw tag', () => {
    expect(formatModelDisplayName('custom-model:1b')).toBe('custom-model:1b');
  });
});

describe('formatModelWithRole', () => {
  it('adds chat role suffix', () => {
    expect(formatModelWithRole('koesn/llama3-openbiollm-8b:latest', 'chat')).toBe(
      'OpenBioLLM 8B (chat)'
    );
  });

  it('adds tool role suffix', () => {
    expect(formatModelWithRole('qwen2.5:7b', 'tool')).toBe('Qwen 2.5 7B (tools)');
  });
});
