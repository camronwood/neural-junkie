import { lazy } from 'react';

export const LazyCadWorkbench = lazy(() =>
  import('./CadWorkbench').then((m) => ({ default: m.CadWorkbench }))
);
export const LazyStructureWorkbench = lazy(() =>
  import('./StructureWorkbench').then((m) => ({ default: m.StructureWorkbench }))
);
export const LazyHtmlBrowserWorkbench = lazy(() =>
  import('./HtmlBrowserWorkbench').then((m) => ({ default: m.HtmlBrowserWorkbench }))
);
export const LazyMusicWorkbench = lazy(() =>
  import('./MusicWorkbench').then((m) => ({ default: m.MusicWorkbench }))
);
export const LazyArenaWorkbench = lazy(() =>
  import('./ArenaWorkbench').then((m) => ({ default: m.ArenaWorkbench }))
);
export const LazyKnowledgeGraphWorkbench = lazy(() =>
  import('./knowledge-graph/KnowledgeGraphWorkbench').then((m) => ({
    default: m.KnowledgeGraphWorkbench,
  }))
);
export const LazyScanAnalysisViewer = lazy(() =>
  import('./ScanAnalysisViewer').then((m) => ({ default: m.ScanAnalysisViewer }))
);
export const LazyComparatorAnalysisViewer = lazy(() =>
  import('./ComparatorAnalysisViewer').then((m) => ({ default: m.ComparatorAnalysisViewer }))
);
export const LazyNeuralCanvasTab = lazy(() =>
  import('./neural-canvas').then((m) => ({ default: m.NeuralCanvasTab }))
);
