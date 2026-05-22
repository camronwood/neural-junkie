import { DiffEditor } from '@monaco-editor/react';

interface GitDiffViewProps {
  path: string;
  original: string;
  modified: string;
  staged: boolean;
  onBack: () => void;
}

export function GitDiffView({ path, original, modified, staged, onBack }: GitDiffViewProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <button
          type="button"
          onClick={onBack}
          className="text-sm text-amber-400 hover:text-amber-300"
        >
          ← Back
        </button>
        <span className="font-mono text-xs text-slack-text truncate" title={path}>
          {path} {staged ? '(staged)' : '(unstaged)'}
        </span>
      </div>
      <div className="h-[min(50vh,400px)] border border-slack-border rounded-md overflow-hidden">
        <DiffEditor
          original={original}
          modified={modified}
          theme="vs-dark"
          options={{
            readOnly: true,
            renderSideBySide: true,
            automaticLayout: true,
            minimap: { enabled: false },
          }}
        />
      </div>
    </div>
  );
}
