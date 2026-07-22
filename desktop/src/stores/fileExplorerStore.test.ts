import { describe, expect, it, beforeEach } from 'vitest';
import { useFileExplorerStore } from './fileExplorerStore';

describe('fileExplorerStore multi-select', () => {
  beforeEach(() => {
    useFileExplorerStore.setState({
      selectedPath: null,
      selectedPaths: [],
    });
  });

  it('toggleSelectedPath adds and removes paths in extend mode', () => {
    const store = useFileExplorerStore.getState();
    store.toggleSelectedPath('src/a.ts', true);
    store.toggleSelectedPath('src/b.ts', true);
    expect(useFileExplorerStore.getState().selectedPaths).toEqual(['src/a.ts', 'src/b.ts']);
    store.toggleSelectedPath('src/a.ts', true);
    expect(useFileExplorerStore.getState().selectedPaths).toEqual(['src/b.ts']);
  });

  it('setSelectedPath replaces selection', () => {
    const store = useFileExplorerStore.getState();
    store.toggleSelectedPath('src/a.ts', true);
    store.setSelectedPath('src/c.ts');
    expect(useFileExplorerStore.getState().selectedPaths).toEqual(['src/c.ts']);
  });

  it('documents that setSelectedPath after async open collapses multi-select', () => {
    const store = useFileExplorerStore.getState();
    store.setSelectedPath('src/a.ts');
    store.toggleSelectedPath('src/b.ts', true);
    expect(useFileExplorerStore.getState().selectedPaths).toEqual(['src/a.ts', 'src/b.ts']);
    // Old FileExplorerPanel race: finishing open called setSelectedPath and wiped B.
    store.setSelectedPath('src/a.ts');
    expect(useFileExplorerStore.getState().selectedPaths).toEqual(['src/a.ts']);
  });
});
