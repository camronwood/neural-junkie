import type { StoreModelItem } from './types';
import { ModelIconTile } from './ModelIconTile';
import { defaultStatusLabel, statusPillClass } from './modelVisuals';

interface ModelStoreCardProps {
  item: StoreModelItem;
  onOpenDetail: (id: string) => void;
}

function actionButtonClass(variant: string | undefined): string {
  const base = 'px-3 py-1.5 text-xs font-medium rounded-lg disabled:opacity-40';
  if (variant === 'danger') {
    return `${base} bg-red-900/50 text-red-200 hover:bg-red-800/60`;
  }
  if (variant === 'secondary') {
    return `${base} bg-gray-700 text-gray-200 hover:bg-gray-600`;
  }
  return `${base} bg-blue-600 text-white hover:bg-blue-500`;
}

export function ModelStoreCard({ item, onOpenDetail }: ModelStoreCardProps) {
  const action = item.primaryAction;
  const statusLabel = item.statusLabel ?? defaultStatusLabel(item.status);
  const showRecommended = item.tags.some((t) => t.toLowerCase() === 'recommended');

  return (
    <article
      role="button"
      tabIndex={0}
      onClick={() => onOpenDetail(item.id)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onOpenDetail(item.id);
        }
      }}
      className="group flex flex-col rounded-xl border border-slack-border bg-slack-bgHover/50 p-4 cursor-pointer transition hover:ring-2 hover:ring-amber-500/30 focus:outline-none focus:ring-2 focus:ring-amber-500/40"
    >
      <div className="flex flex-col items-center text-center gap-3 flex-1">
        <ModelIconTile title={item.title} tags={item.tags} iconKey={item.iconKey} />
        <div className="min-w-0 w-full space-y-1">
          <h4 className="text-sm font-semibold text-white truncate">{item.title}</h4>
          {item.publisher && (
            <p className="text-[10px] uppercase tracking-wide text-gray-500">{item.publisher}</p>
          )}
          <p className="text-xs text-gray-500 font-mono truncate">{item.subtitle}</p>
        </div>
        <div className="flex flex-wrap gap-1 justify-center">
          <span className={`text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded ${statusPillClass(item.status)}`}>
            {statusLabel}
          </span>
          {showRecommended && (
            <span className="text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded bg-amber-900/40 text-amber-300">
              Recommended
            </span>
          )}
        </div>
      </div>
      {action && (
        <button
          type="button"
          disabled={action.disabled}
          onClick={(e) => {
            e.stopPropagation();
            action.onClick();
          }}
          className={`mt-3 w-full ${actionButtonClass(action.variant)}`}
        >
          {action.busyLabel ?? action.label}
        </button>
      )}
    </article>
  );
}
