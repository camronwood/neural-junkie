import { useCallback, useState } from 'react';
import type { ModelStoreView } from './types';

export function useModelStoreNavigation(initialView: ModelStoreView = 'grid') {
  const [view, setView] = useState<ModelStoreView>(initialView);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const openDetail = useCallback((id: string) => {
    setSelectedId(id);
    setView('detail');
  }, []);

  const closeDetail = useCallback(() => {
    setView('grid');
    setSelectedId(null);
  }, []);

  const reset = useCallback(() => {
    setView('grid');
    setSelectedId(null);
  }, []);

  return {
    view,
    selectedId,
    openDetail,
    closeDetail,
    reset,
    isDetail: view === 'detail',
  };
}
