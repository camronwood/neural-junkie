import { useState } from 'react';
import type { CommandOutput as CommandOutputType } from '../types/protocol';

interface CommandOutputProps {
  output: CommandOutputType;
}

export function CommandOutput({ output }: CommandOutputProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  // Format duration from nanoseconds to human-readable
  const formatDuration = (nanoseconds: number): string => {
    const milliseconds = nanoseconds / 1000000;
    if (milliseconds < 1000) {
      return `${milliseconds.toFixed(0)}ms`;
    }
    return `${(milliseconds / 1000).toFixed(2)}s`;
  };

  // Truncate long output
  const truncateOutput = (text: string, maxLength: number = 500): string => {
    if (text.length <= maxLength) {
      return text;
    }
    const truncated = text.substring(0, maxLength);
    const remaining = text.length - maxLength;
    return `${truncated}\n\n... (${remaining} bytes truncated)`;
  };

  const stdout = output.stdout.trim();
  const stderr = output.stderr.trim();
  const hasOutput = stdout.length > 0 || stderr.length > 0;

  return (
    <div className="my-2 border border-slack-border rounded-lg overflow-hidden bg-slack-bgHover/30">
      {/* Header */}
      <div
        className={`px-4 py-2 flex items-center justify-between cursor-pointer ${
          output.success
            ? 'bg-green-500/10 border-b border-green-500/20'
            : 'bg-red-500/10 border-b border-red-500/20'
        }`}
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-2">
          <span className="text-lg">
            {output.success ? '✅' : '❌'}
          </span>
          <span className="font-semibold text-sm">
            {output.success ? 'Command executed successfully' : 'Command failed'}
          </span>
          <span className="text-xs text-slack-textMuted font-mono">
            ({formatDuration(output.duration)})
          </span>
        </div>
        <button
          className="text-slack-textMuted hover:text-slack-text transition-colors"
          onClick={(e) => {
            e.stopPropagation();
            setIsExpanded(!isExpanded);
          }}
        >
          <svg
            className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 9l-7 7-7-7"
            />
          </svg>
        </button>
      </div>

      {/* Command */}
      {isExpanded && (
        <>
          <div className="px-4 py-2 bg-black/20">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-xs font-semibold text-slack-textMuted uppercase">
                Command
              </span>
              <span className="text-xs px-2 py-0.5 rounded bg-blue-500/20 text-blue-400">
                {output.plugin}
              </span>
            </div>
            <code className="text-sm font-mono text-slack-text">
              {output.command}
            </code>
          </div>

          {/* Output */}
          {hasOutput && (
            <div className="border-t border-slack-border">
              {stdout && (
                <div className="px-4 py-3 bg-black/10">
                  <div className="text-xs font-semibold text-slack-textMuted uppercase mb-2">
                    Output
                  </div>
                  <pre className="text-xs font-mono text-slack-text whitespace-pre-wrap overflow-x-auto">
                    {isExpanded ? stdout : truncateOutput(stdout)}
                  </pre>
                </div>
              )}

              {stderr && (
                <div className="px-4 py-3 bg-red-500/5 border-t border-red-500/20">
                  <div className="text-xs font-semibold text-red-400 uppercase mb-2">
                    Errors
                  </div>
                  <pre className="text-xs font-mono text-red-300 whitespace-pre-wrap overflow-x-auto">
                    {stderr}
                  </pre>
                </div>
              )}
            </div>
          )}

          {/* No output message */}
          {!hasOutput && (
            <div className="px-4 py-3 text-xs text-slack-textMuted italic">
              (no output)
            </div>
          )}

          {/* Exit code */}
          {!output.success && (
            <div className="px-4 py-2 bg-black/20 border-t border-slack-border">
              <span className="text-xs text-slack-textMuted">
                Exit code: <span className="font-mono text-red-400">{output.exit_code}</span>
              </span>
            </div>
          )}
        </>
      )}
    </div>
  );
}

