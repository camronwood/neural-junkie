import { modelVisualForItem } from './modelVisuals';

interface ModelIconTileProps {
  title: string;
  tags?: string[];
  iconKey?: string;
  size?: 'card' | 'hero';
}

export function ModelIconTile({ title, tags, iconKey, size = 'card' }: ModelIconTileProps) {
  const visual = modelVisualForItem({ title, tags, iconKey });
  const dim = size === 'hero' ? 'h-28 w-28 text-3xl' : 'h-20 w-20 text-xl';
  return (
    <div
      className={`${dim} shrink-0 rounded-2xl flex items-center justify-center font-bold text-white shadow-lg`}
      style={{
        background: `linear-gradient(135deg, ${visual.gradientFrom}, ${visual.gradientTo})`,
      }}
      aria-hidden
    >
      {visual.monogram}
    </div>
  );
}
