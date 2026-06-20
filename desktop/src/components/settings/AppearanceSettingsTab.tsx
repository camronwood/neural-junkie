import { useSettingsStore, type ColorTheme, type FontSizeScope } from '../../stores/settingsStore';
import type { SettingsTabProps } from './settingsShared';

export function AppearanceSettingsTab({ isActive }: SettingsTabProps) {
  const { settings, updateFontSize, updateFontSizeScope, updateColorTheme } = useSettingsStore();

  if (!isActive) return null;

  const handleFontSizeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    updateFontSize(parseInt(e.target.value));
  };

  const handleScopeChange = (scope: FontSizeScope) => {
    updateFontSizeScope(scope);
  };

  const handleColorThemeChange = (theme: ColorTheme) => {
    updateColorTheme(theme);
  };

  const activeColorTheme: ColorTheme = settings.colorTheme ?? 'slack';

  return (
<div className="space-y-6">
    {/* Color theme */}
    <div>
      <label className="block text-sm font-medium text-slack-text mb-3">
        Color theme
      </label>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {(
          [
            {
              id: 'slack' as const,
              label: 'Slack',
              description: 'Classic blue-accent dark UI',
              previewBg: '#1a1d21',
              previewAccent: '#1164a3',
            },
            {
              id: 'flat' as const,
              label: 'Flat',
              description: 'Navy base, purple accent, mint status',
              previewBg: '#0d0d1a',
              previewAccent: '#a78bfa',
            },
            {
              id: 'roving' as const,
              label: 'Roving',
              description: 'Warm cream, peach & lavender light UI',
              previewBg: '#f9f7f2',
              previewAccent: '#d8c9de',
            },
            {
              id: 'brand' as const,
              label: 'Brand',
              description: 'Website coral on charcoal dark UI',
              previewBg: '#1a161a',
              previewAccent: '#f44a69',
            },
          ] as const
        ).map((option) => (
          <label
            key={option.id}
            className={`flex cursor-pointer flex-col rounded-lg border p-3 transition-colors ${
              activeColorTheme === option.id
                ? 'border-slack-accent bg-slack-accent/10 ring-1 ring-slack-accent'
                : 'border-slack-border bg-slack-bgHover hover:border-slack-textMuted'
            }`}
          >
            <input
              type="radio"
              name="colorTheme"
              value={option.id}
              checked={activeColorTheme === option.id}
              onChange={() => handleColorThemeChange(option.id)}
              className="sr-only"
            />
            <div className="mb-2 flex items-center gap-2">
              <span
                className="h-8 w-8 shrink-0 rounded border border-white/10"
                style={{ backgroundColor: option.previewBg }}
                aria-hidden
              />
              <span
                className="h-8 flex-1 rounded"
                style={{ backgroundColor: option.previewAccent }}
                aria-hidden
              />
            </div>
            <div className="text-sm font-medium text-slack-text">{option.label}</div>
            <div className="text-xs text-slack-textMuted mt-0.5">{option.description}</div>
          </label>
        ))}
      </div>
    </div>

    {/* Font Size */}
    <div>
      <label className="block text-sm font-medium text-slack-text mb-3">
        Font Size: {settings.fontSize}px
      </label>
      <input
        type="range"
        min="12"
        max="24"
        value={settings.fontSize}
        onChange={handleFontSizeChange}
        className="w-full h-2 bg-slack-bgHover rounded-lg appearance-none cursor-pointer slider"
      />
      <div className="flex justify-between text-xs text-slack-textMuted mt-1">
        <span>12px</span>
        <span>24px</span>
      </div>
    </div>

    {/* Font Size Scope */}
    <div>
      <label className="block text-sm font-medium text-slack-text mb-3">
        Apply font size to:
      </label>
      <div className="space-y-2">
        {[
          { value: 'messages', label: 'Messages only', description: 'Only message content' },
          { value: 'input', label: 'Messages & Input', description: 'Chat messages and input field' },
          { value: 'global', label: 'Global', description: 'Entire application' },
        ].map((option) => (
          <label key={option.value} className="flex items-start space-x-3 cursor-pointer">
            <input
              type="radio"
              name="fontScope"
              value={option.value}
              checked={settings.fontSizeScope === option.value}
              onChange={() => handleScopeChange(option.value as FontSizeScope)}
              className="mt-1 text-slack-accent focus:ring-slack-accent"
            />
            <div>
              <div className="text-sm font-medium text-slack-text">{option.label}</div>
              <div className="text-xs text-slack-textMuted">{option.description}</div>
            </div>
          </label>
        ))}
      </div>
    </div>

    {/* Preview */}
    <div>
      <label className="block text-sm font-medium text-slack-text mb-3">
        Preview:
      </label>
      <div 
        className="p-4 bg-slack-bgHover rounded-lg border border-slack-border"
        style={{ fontSize: `${settings.fontSize}px` }}
      >
        <div className="text-slack-text">
          This is how your messages will look with the current font size.
        </div>
        <div className="text-slack-textMuted text-sm mt-2">
          Sample message content with different text styles.
        </div>
      </div>
    </div>
  </div>

  );
}
