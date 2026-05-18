import { useEffect } from 'react';
import type { StoreModelItem } from './types';
import { ModelStoreGrid } from './ModelStoreGrid';
import { ModelStoreDetail } from './ModelStoreDetail';
import { useModelStoreNavigation } from './useModelStoreNavigation';

interface ModelStoreBrowseProps {
  items: StoreModelItem[];
  query: string;
  onQueryChange: (q: string) => void;
  searchPlaceholder?: string;
  headerRight?: React.ReactNode;
  emptyMessage?: string;
  /** Called when grid/detail depth changes (for modal header back button) */
  onViewChange?: (view: 'grid' | 'detail') => void;
  /** Parent requests closing detail (e.g. tab switch) */
  resetDetailSignal?: number;
  footer?: React.ReactNode;
  banner?: React.ReactNode;
}

export function ModelStoreBrowse({
  items,
  query,
  onQueryChange,
  searchPlaceholder,
  headerRight,
  emptyMessage,
  onViewChange,
  resetDetailSignal,
  footer,
  banner,
}: ModelStoreBrowseProps) {
  const nav = useModelStoreNavigation();

  const { isDetail, reset } = nav;

  useEffect(() => {
    onViewChange?.(isDetail ? 'detail' : 'grid');
  }, [isDetail, onViewChange]);

  useEffect(() => {
    if (resetDetailSignal !== undefined && resetDetailSignal > 0) {
      reset();
    }
  }, [resetDetailSignal, reset]);

  const selected = nav.selectedId ? items.find((i) => i.id === nav.selectedId) : undefined;

  if (nav.isDetail && selected) {
    return (
      <div className="space-y-4">
        {banner}
        <ModelStoreDetail item={selected} onBack={nav.closeDetail} showBackButton={false} />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {banner}
      <ModelStoreGrid
        items={items}
        query={query}
        onQueryChange={onQueryChange}
        onOpenDetail={nav.openDetail}
        searchPlaceholder={searchPlaceholder}
        headerRight={headerRight}
        emptyMessage={emptyMessage}
      />
      {footer}
    </div>
  );
}

export { useModelStoreNavigation };
