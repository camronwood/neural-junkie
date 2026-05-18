import { describe, expect, it } from 'vitest';
import { normalizeMermaidSource } from './mermaidNormalize';

describe('normalizeMermaidSource', () => {
  it('quotes flowchart node labels containing @', () => {
    const input = `flowchart LR
  T1[@ReactExpert - label]
  T2 --> T3[@RustExpert]`;
    const out = normalizeMermaidSource(input);
    expect(out).toContain('T1["@ReactExpert - label"]');
    expect(out).toContain('T3["@RustExpert"]');
  });

  it('leaves already-quoted labels unchanged', () => {
    const input = 'flowchart LR\n  A["@Already quoted"]';
    expect(normalizeMermaidSource(input)).toBe(input);
  });

  it('does not mutate sequence diagrams', () => {
    const input = 'sequenceDiagram\n  Alice->>Bob: hello @user';
    expect(normalizeMermaidSource(input)).toBe(input);
  });

  it('quotes round nodes with @', () => {
    const input = 'flowchart TD\n  A(@ReactExpert task)';
    const out = normalizeMermaidSource(input);
    expect(out).toContain('A("@ReactExpert task")');
  });
});
