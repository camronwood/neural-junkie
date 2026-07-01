import type { ComposerMode } from '../constants/composerMode';
import { composerModeTitle } from '../constants/composerMode';

const MODES: ComposerMode[] = ['ask', 'plan', 'agent'];

export function ComposerModeControl({
  mode,
  onChange,
  disabled,
}: {
  mode: ComposerMode;
  onChange: (mode: ComposerMode) => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 border-t border-slack-border bg-slack-bg/80 text-xs">
      <span className="text-slack-textMuted shrink-0">Mode</span>
      <div className="inline-flex rounded border border-slack-border overflow-hidden">
        {MODES.map((m) => (
          <button
            key={m}
            type="button"
            disabled={disabled}
            onClick={() => onChange(m)}
            className={`px-2.5 py-0.5 capitalize ${
              mode === m
                ? 'bg-teal-600 text-white'
                : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
            } disabled:opacity-50`}
            title={composerModeTitle(m)}
          >
            {m}
          </button>
        ))}
      </div>
    </div>
  );
}
