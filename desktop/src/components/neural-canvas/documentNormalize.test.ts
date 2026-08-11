import { describe, expect, it } from 'vitest';
import {
  compileMarkdown,
  documentFromModelOutput,
  unwrapDocument,
} from './documentNormalize';

describe('compileMarkdown', () => {
  it('lifts headings, lists, tables, mermaid, and images', () => {
    const doc = compileMarkdown(`# Trip plan

Welcome.

- Tokyo
- Kyoto

| Name | Role |
| --- | --- |
| Ada | Architect |

\`\`\`mermaid
flowchart LR
  A --> B
\`\`\`

![Skyline](/api/artifacts/a1/assets/embed.png)
`);
    expect(doc.blocks.map((block) => block.type)).toEqual([
      'heading',
      'markdown',
      'list',
      'table',
      'mermaid',
      'image',
    ]);
    expect(doc.blocks[0]).toMatchObject({ level: 1, text: 'Trip plan' });
    expect(doc.blocks[2].items).toEqual(['Tokyo', 'Kyoto']);
    expect(doc.blocks[3].rows?.[0]).toMatchObject({ name: 'Ada' });
    expect(doc.blocks[4].source).toContain('flowchart LR');
    expect(doc.blocks[5]).toMatchObject({
      alt: 'Skyline',
      src: '/api/artifacts/a1/assets/embed.png',
    });
  });
});

describe('unwrapDocument', () => {
  it('wraps a legacy markdown string', () => {
    const doc = unwrapDocument('# Canvas\n\n- one\n');
    expect(doc.blocks[0]).toMatchObject({ type: 'heading', text: 'Canvas' });
    expect(doc.blocks[1]).toMatchObject({ type: 'list', items: ['one'] });
  });

  it('unwraps { content } objects', () => {
    const doc = unwrapDocument({ content: '## Notes\n\nhello' });
    expect(doc.blocks[0]).toMatchObject({ type: 'heading', text: 'Notes' });
  });

  it('passes through a document object', () => {
    const doc = unwrapDocument({
      schema_version: 1,
      blocks: [{ type: 'callout', tone: 'info', body: 'Hi' }],
    });
    expect(doc.blocks[0]).toMatchObject({ type: 'callout', body: 'Hi' });
  });
});

describe('documentFromModelOutput', () => {
  it('parses fenced document JSON', () => {
    const doc = documentFromModelOutput('```json\n{"schema_version":1,"blocks":[{"type":"heading","level":1,"text":"Hi"}]}\n```');
    expect(doc.blocks[0]).toMatchObject({ type: 'heading', text: 'Hi' });
  });

  it('compiles markdown when JSON is invalid', () => {
    const doc = documentFromModelOutput('| A | B |\n| --- | --- |\n| 1 | 2 |\n');
    expect(doc.blocks[0]?.type).toBe('table');
    expect(doc.blocks[0]?.rows?.[0]).toMatchObject({ a: '1' });
  });
});
