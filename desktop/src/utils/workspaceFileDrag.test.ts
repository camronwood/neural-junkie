import { describe, expect, it } from 'vitest';
import {
  WORKSPACE_FILE_DRAG_MIME,
  parseWorkspaceFileDrag,
  setWorkspaceFileDragData,
} from './workspaceFileDrag';

describe('workspaceFileDrag', () => {
  it('round-trips payload via DataTransfer', () => {
    const dt = {
      _data: {} as Record<string, string>,
      setData(type: string, value: string) {
        this._data[type] = value;
      },
      getData(type: string) {
        return this._data[type] ?? '';
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
