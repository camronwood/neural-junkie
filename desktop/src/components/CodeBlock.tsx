import { memo, useMemo } from 'react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { mapHighlighterLanguage } from '../utils/markdownNormalize';

const CODE_BLOCK_CUSTOM_STYLE: React.CSSProperties = {
  margin: 0,
  padding: '0.75rem 1rem',
  background: 'var(--nj-prose-pre-bg)',
  fontSize: '0.8125rem',
  lineHeight: 1.55,
};

const CODE_TAG_PROPS = {
  style: {
    whiteSpace: 'pre-wrap' as const,
    wordBreak: 'break-word' as const,
  },
};

export interface CodeBlockProps {
  content: string;
  language?: string;
  /** When false, hide the language header bar (e.g. short inline paths). */
  showHeader?: boolean;
}

export const CodeBlock = memo(function CodeBlockImpl({
  content,
  language = 'text',
  showHeader = true,
}: CodeBlockProps) {
  const hl = mapHighlighterLanguage(language);
  const showLineNumbers = useMemo(() => content.split('\n').length <= 40, [content]);
  return (
    <div className="overflow-hidden rounded-md border border-slack-border shadow-sm">
      {showHeader ? (
        <div className="border-b border-slack-border bg-slack-bgHover px-3 py-1.5 text-xs font-mono text-slack-textMuted">
          {language}
        </div>
      ) : null}
      <SyntaxHighlighter
        language={hl}
        style={vscDarkPlus}
        customStyle={CODE_BLOCK_CUSTOM_STYLE}
        codeTagProps={CODE_TAG_PROPS}
        showLineNumbers={showLineNumbers}
        wrapLongLines
      >
        {content}
      </SyntaxHighlighter>
    </div>
  );
});
