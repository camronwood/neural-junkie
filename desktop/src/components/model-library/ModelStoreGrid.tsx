import type { StoreModelItem } from './types';
import { ModelStoreCard } from './ModelStoreCard';

interface ModelStoreGridProps {
  items: StoreModelItem[];
  query: string;
  onQueryChange: (q: string) => void;
  onOpenDetail: (id: string) => void;
  searchPlaceholder?: string;
  headerRight?: React.ReactNode;
  emptyMessage?: string;
}

export function ModelStoreGrid({
  items,
  query,
  onQueryChange,
  onOpenDetail,
  searchPlaceholder = 'Search models…',
  headerRight,
  emptyMessage = 'No models match your search.',
}: ModelStoreGridProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <div className="flex-1 min-w-[200px]">
          <label className="sr-only">Search catalog</label>
          <input
            type="search"
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            placeholder={searchPlaceholder}
            className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-lg text-sm text-white"
          />
        </div>
        {headerRight}
      </div>
      {items.length === 0 ? (
        <div className="py-12 text-center text-sm text-gray-500 border border-gray-700 rounded-xl">
          {emptyMessage}
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-4 max-h-[min(480px,55vh)] overflow-y-auto pr-1">
          {items.map((item) => (
            <ModelStoreCard key={item.id} item={item} onOpenDetail={onOpenDetail} />
          ))}
        </div>
      )}
    </div>
  );
}
