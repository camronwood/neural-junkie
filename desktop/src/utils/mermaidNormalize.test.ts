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

  it('quotes path-like labels that break Mermaid lexing', () => {
    const input = `flowchart TD
  A[packages/package-lock.json] --> B[public/assets]
  C[src] --> D[ok]`;
    const out = normalizeMermaidSource(input);
    expect(out).toContain('A["packages/package-lock.json"]');
    expect(out).toContain('B["public/assets"]');
    expect(out).toContain('D[ok]');
  });

  it('quotes labels after %%{init}%% frontmatter', () => {
    const input = `%%{init: {'theme':'base'}}%%
flowchart TD
  A[User / Client] --> B[src-tauri]`;
    const out = normalizeMermaidSource(input);
    expect(out).toContain('A["User / Client"]');
    expect(out).toContain('B[src-tauri]');
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
