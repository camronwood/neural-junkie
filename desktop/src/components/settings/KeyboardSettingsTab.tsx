import { getShortcutsForDisplay, formatChord } from '../../shortcuts';
import type { SettingsTabProps } from './settingsShared';

export function KeyboardSettingsTab({ isActive }: SettingsTabProps) {
  if (!isActive) return null;

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-semibold text-slack-text mb-2">Keyboard shortcuts</h3>
        <p className="text-sm text-slack-textMuted mb-4">
          Fixed defaults (not customizable yet). Toolbar buttons show the same chords on hover.
        </p>
      </div>
      <div className="border border-slack-border rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slack-bgHover/50 text-left text-slack-textMuted">
            <tr>
              <th className="px-4 py-2 font-medium">Action</th>
              <th className="px-4 py-2 font-medium w-40">Shortcut</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slack-border">
            {getShortcutsForDisplay().map((row) => (
              <tr key={row.id} className="text-slack-text">
                <td className="px-4 py-2">{row.label}</td>
                <td className="px-4 py-2">
                  <kbd className="rounded bg-slack-bgHover px-1.5 py-0.5 font-mono text-xs">
                    {formatChord(row.chord)}
                  </kbd>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
