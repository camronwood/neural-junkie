export type ViewportPreset = 'mobile' | 'tablet' | 'desktop' | 'full';

export const VIEWPORT_PRESETS: Record<Exclude<ViewportPreset, 'full'>, { width: number; height: number; label: string }> = {
  mobile: { width: 375, height: 812, label: 'Mobile' },
  tablet: { width: 768, height: 1024, label: 'Tablet' },
  desktop: { width: 1280, height: 800, label: 'Desktop' },
};

interface BrowserResponsiveToolbarProps {
  preset: ViewportPreset;
  onPresetChange: (preset: ViewportPreset) => void;
}

export function BrowserResponsiveToolbar({ preset, onPresetChange }: BrowserResponsiveToolbarProps) {
  return (
    <div className="flex rounded-md border border-slack-border overflow-hidden text-xs">
      {(Object.keys(VIEWPORT_PRESETS) as Array<Exclude<ViewportPreset, 'full'>>).map((key) => (
        <button
          key={key}
          type="button"
          onClick={() => onPresetChange(key)}
          className={`px-3 py-1.5 ${preset === key ? 'bg-teal-600 text-white' : 'bg-slack-bgHover text-slack-textMuted hover:bg-slack-bgHover/80'}`}
        >
          {VIEWPORT_PRESETS[key].label}
        </button>
      ))}
      <button
        type="button"
        onClick={() => onPresetChange('full')}
        className={`px-3 py-1.5 ${preset === 'full' ? 'bg-teal-600 text-white' : 'bg-slack-bgHover text-slack-textMuted hover:bg-slack-bgHover/80'}`}
      >
        Full
      </button>
    </div>
  );
}

export function viewportForPreset(preset: ViewportPreset): { width: number; height: number } | null {
  if (preset === 'full') return null;
  const vp = VIEWPORT_PRESETS[preset];
  return { width: vp.width, height: vp.height };
}
