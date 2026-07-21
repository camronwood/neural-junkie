import { beforeEach, describe, expect, it } from 'vitest';
import { useEditorStore } from './editorStore';
import { parseScanSummaryMetadata, SCAN_SUMMARY_METADATA_FILE } from '../utils/scanSummary';

const sampleData = () =>
  parseScanSummaryMetadata(
    JSON.stringify({
      metadata: [{ imageName: 'A1', spots: [{ analyte: 'IL-6', row: '1', column: '1', x_px: 1, y_px: 2 }] }],
    })
  );

function resetEditorStore(): void {
  useEditorStore.setState({
    tabs: [],
    activeTabId: null,
    saving: false,
    error: null,
  });
}

describe('editorStore openScanSummary', () => {
  beforeEach(() => {
    resetEditorStore();
  });

  it('opens a scan-summary tab with metadata path', () => {
    const data = sampleData();
    useEditorStore.getState().openScanSummary('ws-1', 'run1', data, 'A1');
    const tab = useEditorStore.getState().getActiveTab();
    expect(tab?.viewMode).toBe('scan-summary');
    expect(tab?.scanSummaryDir).toBe('run1');
    expect(tab?.scanSummaryData).toBe(data);
    expect(tab?.scanSummaryInitialWell).toBe('A1');
    expect(tab?.path).toBe(`run1/${SCAN_SUMMARY_METADATA_FILE}`);
  });

  it('reuses existing scan-summary tab for same workspace and dir', () => {
    const data = sampleData();
    useEditorStore.getState().openScanSummary('ws-1', 'run1', data);
    const firstId = useEditorStore.getState().getActiveTab()?.id;
    const updated = parseScanSummaryMetadata(
      JSON.stringify({
        metadata: [{ imageName: 'B2', spots: [] }],
      })
    );
    useEditorStore.getState().openScanSummary('ws-1', 'run1', updated, 'B2');
    expect(useEditorStore.getState().tabs).toHaveLength(1);
    expect(useEditorStore.getState().getActiveTab()?.id).toBe(firstId);
    expect(useEditorStore.getState().getActiveTab()?.scanSummaryInitialWell).toBe('B2');
    expect(useEditorStore.getState().getActiveTab()?.scanSummaryData?.metadata).toHaveLength(1);
  });

  it('replaces stale text tab for metadata json', () => {
    useEditorStore.getState().openFile('ws-1', `run1/${SCAN_SUMMARY_METADATA_FILE}`, '{}', 'json');
    const staleId = useEditorStore.getState().getActiveTab()?.id;
    const data = sampleData();
    useEditorStore.getState().openScanSummary('ws-1', 'run1', data);
    expect(useEditorStore.getState().tabs).toHaveLength(1);
    const tab = useEditorStore.getState().getActiveTab();
    expect(tab?.id).toBe(staleId);
    expect(tab?.viewMode).toBe('scan-summary');
  });

  it('getTabByPath prefers scan-summary tab over text tab at same path', () => {
    const path = `run1/${SCAN_SUMMARY_METADATA_FILE}`;
    useEditorStore.setState({
      tabs: [
        {
          id: 'text-tab',
          workspaceId: 'ws-1',
          path,
          content: '{}',
          isDirty: false,
          viewMode: 'text',
        },
        {
          id: 'scan-tab',
          workspaceId: 'ws-1',
          path,
          content: '',
          isDirty: false,
          viewMode: 'scan-summary',
          scanSummaryDir: 'run1',
          scanSummaryData: sampleData(),
        },
      ],
      activeTabId: 'scan-tab',
    });
    const tab = useEditorStore.getState().getTabByPath('ws-1', path);
    expect(tab?.id).toBe('scan-tab');
    expect(tab?.viewMode).toBe('scan-summary');
  });
});

describe('editorStore openArtifact', () => {
  beforeEach(() => {
    resetEditorStore();
  });

  it('opens and reuses a Neural Canvas tab by artifact id', () => {
    useEditorStore.getState().openArtifact('ws-1', 'artifact-1', 'Latency report');
    const first = useEditorStore.getState().getActiveTab();
    expect(first).toMatchObject({
      workspaceId: 'ws-1',
      path: 'Latency report',
      viewMode: 'neural-canvas',
      artifactId: 'artifact-1',
    });
    useEditorStore.getState().openArtifact('ws-2', 'artifact-1', 'Renamed');
    expect(useEditorStore.getState().tabs).toHaveLength(1);
    expect(useEditorStore.getState().activeTabId).toBe(first?.id);
  });
});
