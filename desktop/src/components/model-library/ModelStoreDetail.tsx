import type { StoreModelItem } from './types';
import { ModelIconTile } from './ModelIconTile';
import { defaultStatusLabel, modelVisualForItem, statusPillClass } from './modelVisuals';

interface ModelStoreDetailProps {
  item: StoreModelItem;
  onBack: () => void;
  showBackButton?: boolean;
}

function actionButtonClass(variant: string | undefined): string {
  const base = 'px-4 py-2 text-sm font-medium rounded-lg disabled:opacity-40';
  if (variant === 'danger') {
    return `${base} bg-red-900/50 text-red-200 hover:bg-red-800/60`;
  }
  if (variant === 'secondary') {
    return `${base} bg-gray-700 text-gray-200 hover:bg-gray-600`;
  }
  if (variant === 'primary') {
    return `${base} bg-emerald-700/80 text-white hover:bg-emerald-600`;
  }
  return `${base} bg-blue-600 text-white hover:bg-blue-500`;
}

export function ModelStoreDetail({ item, onBack, showBackButton = true }: ModelStoreDetailProps) {
  const visual = modelVisualForItem({
    title: item.title,
    tags: item.tags,
    iconKey: item.iconKey,
  });
  const statusLabel = item.statusLabel ?? defaultStatusLabel(item.status);

  return (
    <div className="space-y-4 max-h-[min(520px,60vh)] overflow-y-auto">
      {showBackButton && (
        <button
          type="button"
          onClick={onBack}
          className="text-sm text-amber-400 hover:text-amber-300 flex items-center gap-1"
        >
          ← Back to browse
        </button>
      )}

      <div className="flex flex-col sm:flex-row gap-6">
        <div className="flex flex-col items-center sm:items-start gap-2 shrink-0">
          <ModelIconTile title={item.title} tags={item.tags} iconKey={item.iconKey} size="hero" />
          <span className="text-xs text-gray-500">{visual.categoryLabel}</span>
        </div>

        <div className="flex-1 min-w-0 space-y-3">
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-lg font-semibold text-white">{item.title}</h3>
              <span
                className={`text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded ${statusPillClass(item.status)}`}
              >
                {statusLabel}
              </span>
            </div>
            {item.publisher && (
              <p className="text-xs text-gray-500 mt-0.5">{item.publisher}</p>
            )}
            <p className="text-sm text-gray-500 font-mono mt-1 break-all">{item.subtitle}</p>
            {item.sizeHint && (
              <p className="text-xs text-gray-500 mt-1">{item.sizeHint}</p>
            )}
          </div>

          <p className="text-sm text-gray-300 leading-relaxed">{item.description}</p>

          {item.tags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {item.tags.map((t) => (
                <span
                  key={t}
                  className="text-[10px] px-2 py-0.5 rounded-full bg-gray-800 text-gray-400"
                >
                  {t}
                </span>
              ))}
            </div>
          )}

          {item.detailRows && item.detailRows.length > 0 && (
            <dl className="space-y-2 border-t border-gray-800 pt-3">
              {item.detailRows.map((row) => (
                <div key={row.label} className="flex flex-col sm:flex-row sm:gap-4 text-xs">
                  <dt className="text-gray-500 shrink-0 sm:w-28">{row.label}</dt>
                  <dd className="text-gray-300 font-mono break-all">{row.value}</dd>
                </div>
              ))}
            </dl>
          )}

          {item.externalUrl && (
            <a
              href={item.externalUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-block text-sm text-blue-400 hover:text-blue-300"
              onClick={(e) => e.stopPropagation()}
            >
              View on Hugging Face ↗
            </a>
          )}
        </div>
      </div>

      {item.detailActions && item.detailActions.length > 0 && (
        <div className="sticky bottom-0 pt-3 border-t border-gray-800 bg-slack-bg/95 flex flex-wrap gap-2">
          {item.detailActions.map((action) => (
            <button
              key={action.id}
              type="button"
              disabled={action.disabled}
              onClick={action.onClick}
              className={actionButtonClass(action.variant)}
            >
              {action.busyLabel ?? action.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
