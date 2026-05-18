import { describe, expect, it } from 'vitest';
import {
  WORKSPACE_FILE_DRAG_MIME,
  parseWorkspaceFileDrag,
  setWorkspaceFileDragData,
} from './workspaceFileDrag';

describe('workspaceFileDrag', () => {
  it('round-trips payload via DataTransfer', () => {
    const store: Record<string, string> = {};
    const dt = {
      setData(type: string, value: string) {
        store[type] = value;
      },
      getData(type: string) {
        return store[type] ?? '';
      },
      effectAllowed: '',
    } as unknown as DataTransfer;

    setWorkspaceFileDragData(dt, { workspaceId: 'ws-1', path: 'src/main.go' });
    expect(dt.getData(WORKSPACE_FILE_DRAG_MIME)).toContain('ws-1');
    const parsed = parseWorkspaceFileDrag(dt);
    expect(parsed).toHaveLength(1);
    expect(parsed[0]).toEqual({ workspaceId: 'ws-1', path: 'src/main.go' });
  });
});
