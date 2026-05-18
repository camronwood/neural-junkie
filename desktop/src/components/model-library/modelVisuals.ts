export interface ModelVisual {
  monogram: string;
  gradientFrom: string;
  gradientTo: string;
  categoryLabel: string;
}

const ICON_GRADIENTS: Record<string, { from: string; to: string; label: string }> = {
  qwen: { from: '#7c3aed', to: '#4f46e5', label: 'Qwen' },
  llama: { from: '#2563eb', to: '#1d4ed8', label: 'Llama' },
  mistral: { from: '#ea580c', to: '#c2410c', label: 'Mistral' },
  deepseek: { from: '#0891b2', to: '#0e7490', label: 'DeepSeek' },
  phi: { from: '#ca8a04', to: '#a16207', label: 'Phi' },
  gemma: { from: '#db2777', to: '#be185d', label: 'Gemma' },
  codellama: { from: '#059669', to: '#047857', label: 'Code Llama' },
  llava: { from: '#9333ea', to: '#7e22ce', label: 'LLaVA' },
  granite: { from: '#475569', to: '#334155', label: 'Granite' },
  starcoder: { from: '#16a34a', to: '#15803d', label: 'StarCoder' },
  embedding: { from: '#64748b', to: '#475569', label: 'Embedding' },
};

const TAG_GRADIENTS: Record<string, { from: string; to: string; label: string }> = {
  code: { from: '#059669', to: '#047857', label: 'Code' },
  vision: { from: '#9333ea', to: '#7e22ce', label: 'Vision' },
  embedding: { from: '#64748b', to: '#475569', label: 'Embedding' },
  reasoning: { from: '#0891b2', to: '#0e7490', label: 'Reasoning' },
  general: { from: '#d97706', to: '#b45309', label: 'General' },
  large: { from: '#dc2626', to: '#b91c1c', label: 'Large' },
  small: { from: '#65a30d', to: '#4d7c0f', label: 'Compact' },
  recommended: { from: '#ca8a04', to: '#a16207', label: 'Featured' },
};

function monogramFromTitle(title: string): string {
  const words = title.trim().split(/\s+/).filter(Boolean);
  if (words.length >= 2) {
    return (words[0][0] + words[1][0]).toUpperCase();
  }
  return title.slice(0, 2).toUpperCase() || '?';
}

export function modelVisualForItem(opts: {
  title: string;
  tags?: string[];
  iconKey?: string;
}): ModelVisual {
  const key = opts.iconKey?.toLowerCase();
  if (key && ICON_GRADIENTS[key]) {
    const g = ICON_GRADIENTS[key];
    return {
      monogram: monogramFromTitle(opts.title),
      gradientFrom: g.from,
      gradientTo: g.to,
      categoryLabel: g.label,
    };
  }
  for (const tag of opts.tags ?? []) {
    const t = tag.toLowerCase();
    if (TAG_GRADIENTS[t]) {
      const g = TAG_GRADIENTS[t];
      return {
        monogram: monogramFromTitle(opts.title),
        gradientFrom: g.from,
        gradientTo: g.to,
        categoryLabel: g.label,
      };
    }
  }
  return {
    monogram: monogramFromTitle(opts.title),
    gradientFrom: '#475569',
    gradientTo: '#334155',
    categoryLabel: 'Model',
  };
}

export function statusPillClass(status: string): string {
  switch (status) {
    case 'installed':
    case 'on_disk':
      return 'bg-green-900/40 text-green-300';
    case 'cloud':
      return 'bg-sky-900/40 text-sky-300';
    default:
      return 'bg-slate-800 text-slate-400';
  }
}

export function defaultStatusLabel(status: string): string {
  switch (status) {
    case 'installed':
      return 'Installed';
    case 'on_disk':
      return 'On disk';
    case 'cloud':
      return 'Cloud';
    default:
      return 'Available';
  }
}
