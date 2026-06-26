import { describe, expect, it } from 'vitest';
import {
  looksLikeBlockMarkdown,
  normalizeProseMarkdownBlocks,
  promoteStandaloneImageFilePaths,
} from './markdownNormalize';
import { renderChatMarkdown } from './markdownRenderer';

describe('normalizeProseMarkdownBlocks', () => {
  it('breaks inline horizontal rules and headings', () => {
    const input =
      'professional and engaging manner. --- ### Introducing Neural Junkie: Revolutionizing Collaboration';
    const out = normalizeProseMarkdownBlocks(input);
    expect(out).toContain('\n---\n');
    expect(out).toContain('\n### Introducing Neural Junkie');
  });

  it('breaks numbered lists and sub-bullets', () => {
    const input = '#### Key Features 1. Real-Time Collaboration: - Integrated Workspaces: Neural Junkie';
    const out = normalizeProseMarkdownBlocks(input);
    expect(out).toContain('#### Key Features');
    expect(out).toContain('\n1. Real-Time Collaboration');
    expect(out).toContain(':\n\n- Integrated Workspaces');
  });

  it('renders article-style markdown to headings and lists', () => {
    const normalized = normalizeProseMarkdownBlocks(
      'Intro text. --- ### Title Here #### Section 1. First item: - Detail one 2. Second item: - Detail two'
    );
    const html = renderChatMarkdown(normalized);
    expect(html).toContain('<h3');
    expect(html).toContain('<h4');
    expect(html).toContain('<ol');
    expect(html).toContain('<ul');
    expect(html).toContain('<hr');
  });

  it('breaks inline MFA-style numbered lists with sub-bullets', () => {
    const input =
      "I'll walk you through the MFA setup process using Google Workspace: 1. Enforce MFA in Google Workspace Admin Console: - Log into the Google Workspace admin console (admin.google.com) - Navigate to Security 2. Set up GitHub Organization MFA Policies: - In GitHub Enterprise 3. Google Workspace + GitHub Integration: - Use Google Authenticator";
    const normalized = normalizeProseMarkdownBlocks(input);
    expect(normalized).toContain('Google Workspace:\n\n1. Enforce MFA');
    expect(normalized).toContain('Admin Console:\n\n- Log into');
    expect(normalized).toContain('(admin.google.com)\n\n- Navigate');
    expect(normalized).not.toContain('Security 2. Set up');
    expect(normalized).toContain('\n\n2. Set up GitHub');
    expect(normalized).toContain('\n\n3. Google Workspace');

    const html = renderChatMarkdown(normalized);
    expect(html).toContain('<ol');
    expect(html).toContain('<ul');
    expect(html).toContain('<li');
  });

  it('detects inline numbered lists before normalization', () => {
    expect(looksLikeBlockMarkdown('Here is the plan: 1. First step 2. Second step')).toBe(true);
  });

  it('breaks agent review sections and parenthesized enumerations', () => {
    const input =
      "Review of FrontendEngineer's Response What went wrong: The agent failed to analyze errors: (1) missing default export in SettingsButton.tsx, (2) unused imports in Header.tsx, and (3) missing react-bootstrap dependency. Worse, they submitted an empty FILE_CHANGE block. What could have been better: Before suggesting edits, the agent should have used read_file. Recommendation: Always read error output before proposing fixes.";
    const normalized = normalizeProseMarkdownBlocks(input);
    expect(normalized).toMatch(/^## Review of FrontendEngineer's Response/);
    expect(normalized).toContain('### What went wrong');
    expect(normalized).toContain('### What could have been better');
    expect(normalized).toContain('### Recommendation');
    expect(normalized).toContain('\n\n1. missing default export');
    expect(normalized).toContain('\n\n2. unused imports');
    expect(normalized).toContain('\n\n3. missing react-bootstrap');

    const html = renderChatMarkdown(normalized);
    expect(html).toContain('<h2');
    expect(html).toContain('<h3');
    expect(html).toContain('<ol');
    expect(html).toContain('<li');
  });

  it('does not alter fenced code blocks', () => {
    const input = '```go\nfmt.Println(" --- ### not a header ")\n```';
    expect(normalizeProseMarkdownBlocks(input)).toBe(input);
  });
});

describe('looksLikeBlockMarkdown', () => {
  it('detects headings and lists', () => {
    expect(looksLikeBlockMarkdown('### Hello')).toBe(true);
    expect(looksLikeBlockMarkdown('plain chat reply')).toBe(false);
  });
});

describe('promoteStandaloneImageFilePaths', () => {
  it('rewrites a lone absolute macOS image path to markdown image', () => {
    const input = 'Saved to:\n\n/Users/me/project/out.png\n\nDone.';
    expect(promoteStandaloneImageFilePaths(input)).toBe(
      'Saved to:\n\n![](/Users/me/project/out.png)\n\nDone.'
    );
  });

  it('rewrites Windows-style absolute paths', () => {
    const input = 'C:\\Users\\me\\x.JPEG';
    expect(promoteStandaloneImageFilePaths(input)).toBe('![](C:\\Users\\me\\x.JPEG)');
  });

  it('does not rewrite paths with spaces (likely prose)', () => {
    const input = '/Users/me/my photos/out.png';
    expect(promoteStandaloneImageFilePaths(input)).toBe(input);
  });

  it('does not rewrite lines containing backticks', () => {
    const input = '`/Users/me/out.png`';
    expect(promoteStandaloneImageFilePaths(input)).toBe(input);
  });

  it('does not rewrite existing markdown images', () => {
    const input = '![](/Users/me/out.png)';
    expect(promoteStandaloneImageFilePaths(input)).toBe(input);
  });
});
