import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  WORKSPACE_FILE_DRAG_MIME,
  WORKSPACE_FILE_DROP_EVENT,
  WORKSPACE_FILE_DROPZONE_ATTR,
  clearWorkspaceFileDragData,
  dispatchWorkspaceFileDropEventAtPoint,
  parseWorkspaceFileDrag,
  scheduleWorkspaceFileDragClear,
  setActiveWorkspaceFileDropZone,
  setWorkspaceFileDragData,
} from './workspaceFileDrag';

afterEach(() => {
  vi.useRealTimers();
  clearWorkspaceFileDragData();
});

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

  it('falls back when WebKit drops the custom MIME payload', () => {
    const sourceStore: Record<string, string> = {};
    const source = {
      setData(type: string, value: string) {
        sourceStore[type] = value;
      },
      getData(type: string) {
        return sourceStore[type] ?? '';
      },
      effectAllowed: '',
    } as unknown as DataTransfer;

    setWorkspaceFileDragData(source, { workspaceId: 'ws-1', path: 'src/main.go' });

    const drop = {
      getData(type: string) {
        return type === 'text/plain' ? 'src/main.go' : '';
      },
    } as unknown as DataTransfer;

    expect(parseWorkspaceFileDrag(drop)).toEqual([{ workspaceId: 'ws-1', path: 'src/main.go' }]);
    clearWorkspaceFileDragData();
    expect(parseWorkspaceFileDrag(drop)).toEqual([]);
  });

  it('keeps the fallback briefly if dragend fires before drop', () => {
    vi.useFakeTimers();
    const source = {
      setData() {
        /* WebKit may drop this before the drop event. */
      },
      getData() {
        return '';
      },
      effectAllowed: '',
    } as unknown as DataTransfer;

    setWorkspaceFileDragData(source, { workspaceId: 'ws-1', path: 'src/main.go' });
    scheduleWorkspaceFileDragClear();

    const drop = {
      getData() {
        return '';
      },
    } as unknown as DataTransfer;

    expect(parseWorkspaceFileDrag(drop)).toEqual([{ workspaceId: 'ws-1', path: 'src/main.go' }]);

    vi.advanceTimersByTime(1_500);
    expect(parseWorkspaceFileDrag(drop)).toEqual([]);
  });

  it('falls back when DataTransfer.getData throws for custom drag types', () => {
    const source = {
      setData() {
        /* ignore */
      },
      getData() {
        throw new Error('unavailable');
      },
      effectAllowed: '',
    } as unknown as DataTransfer;

    setWorkspaceFileDragData(source, { workspaceId: 'ws-1', path: 'src/main.go' });

    const drop = {
      getData() {
        throw new Error('unavailable');
      },
    } as unknown as DataTransfer;

    expect(parseWorkspaceFileDrag(drop)).toEqual([{ workspaceId: 'ws-1', path: 'src/main.go' }]);
  });

  it('dispatches an explicit drop event to the composer under the drag end point', () => {
    const source = {
      setData() {
        /* ignore */
      },
      getData() {
        return '';
      },
      effectAllowed: '',
    } as unknown as DataTransfer;
    setWorkspaceFileDragData(source, { workspaceId: 'ws-1', path: 'src/main.go' });

    const dropZone = document.createElement('div');
    dropZone.setAttribute(WORKSPACE_FILE_DROPZONE_ATTR, 'true');
    const child = document.createElement('div');
    dropZone.appendChild(child);
    document.body.appendChild(dropZone);
    const received: unknown[] = [];
    dropZone.addEventListener(WORKSPACE_FILE_DROP_EVENT, (event) => {
      received.push((event as CustomEvent).detail);
    });
    const originalElementFromPoint = document.elementFromPoint;
    document.elementFromPoint = vi.fn(() => child);

    expect(dispatchWorkspaceFileDropEventAtPoint(10, 20)).toBe(true);
    expect(received).toEqual([
      {
        payload: { workspaceId: 'ws-1', path: 'src/main.go' },
        payloads: [{ workspaceId: 'ws-1', path: 'src/main.go' }],
      },
    ]);

    document.elementFromPoint = originalElementFromPoint;
    dropZone.remove();
  });

  it('dispatches to the active drop zone when dragend coordinates miss', () => {
    const source = {
      setData() {
        /* ignore */
      },
      getData() {
        return '';
      },
      effectAllowed: '',
    } as unknown as DataTransfer;
    setWorkspaceFileDragData(source, { workspaceId: 'ws-1', path: 'run/A1' });

    const dropZone = document.createElement('div');
    dropZone.setAttribute(WORKSPACE_FILE_DROPZONE_ATTR, 'true');
    document.body.appendChild(dropZone);
    setActiveWorkspaceFileDropZone(dropZone);

    const received: unknown[] = [];
    dropZone.addEventListener(WORKSPACE_FILE_DROP_EVENT, (event) => {
      received.push((event as CustomEvent).detail);
    });
    const originalElementFromPoint = document.elementFromPoint;
    document.elementFromPoint = vi.fn(() => null);

    expect(dispatchWorkspaceFileDropEventAtPoint(0, 0)).toBe(true);
    expect(received).toEqual([{ payload: { workspaceId: 'ws-1', path: 'run/A1' }, payloads: [{ workspaceId: 'ws-1', path: 'run/A1' }] }]);

    document.elementFromPoint = originalElementFromPoint;
    dropZone.remove();
  });

  it('round-trips multi-file payload via DataTransfer', () => {
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

    setWorkspaceFileDragData(dt, [
      { workspaceId: 'ws-1', path: 'src/a.ts' },
      { workspaceId: 'ws-1', path: 'src/b.ts' },
    ]);
    expect(parseWorkspaceFileDrag(dt)).toEqual([
      { workspaceId: 'ws-1', path: 'src/a.ts' },
      { workspaceId: 'ws-1', path: 'src/b.ts' },
    ]);
  });

  it('still parses legacy single-object MIME payloads', () => {
    const dt = {
      getData(type: string) {
        if (type === WORKSPACE_FILE_DRAG_MIME) {
          return JSON.stringify({ workspaceId: 'ws-1', path: 'src/main.go' });
        }
        return '';
      },
    } as unknown as DataTransfer;
    expect(parseWorkspaceFileDrag(dt)).toEqual([{ workspaceId: 'ws-1', path: 'src/main.go' }]);
  });
});
