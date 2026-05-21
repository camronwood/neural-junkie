import { describe, expect, it } from 'vitest';
import {
  isScanSummaryDirListing,
  isScanSummaryMetadataPath,
  isScanSummaryWellPath,
  isScanSummaryWorkspaceRoot,
  parseScanSummaryMetadata,
  scanSummaryDirForFilePath,
  scanSummaryDirFromMetadataPath,
  WELL_ID_PATTERN,
} from './scanSummary';

describe('scanSummary', () => {
  it('detects metadata path', () => {
    expect(isScanSummaryMetadataPath('run/imageMetadata.json')).toBe(true);
    expect(isScanSummaryMetadataPath('A1')).toBe(false);
  });

  it('detects dir listing', () => {
    expect(
      isScanSummaryDirListing([
        { name: 'imageMetadata.json', is_dir: false },
        { name: 'A1', is_dir: false },
      ])
    ).toBe(true);
    expect(isScanSummaryDirListing([{ name: 'readme.txt', is_dir: false }])).toBe(false);
  });

  it('parses metadata and indexes by well', () => {
    const raw = JSON.stringify({
      metadata: [
        { imageName: 'A1', spots: [{ analyte: 'IL-6', row: '1', column: '1', x_px: 1, y_px: 2 }] },
        { imageName: 'B2', spots: [] },
      ],
    });
    const data = parseScanSummaryMetadata(raw);
    expect(data.metadata).toHaveLength(2);
    expect(data.byWell.get('A1')?.spots[0].analyte).toBe('IL-6');
    expect(scanSummaryDirFromMetadataPath('folder/imageMetadata.json')).toBe('folder');
  });

  it('validates well ids', () => {
    expect(WELL_ID_PATTERN.test('A1')).toBe(true);
    expect(WELL_ID_PATTERN.test('H12')).toBe(true);
    expect(WELL_ID_PATTERN.test('I1')).toBe(false);
  });

  it('detects well paths and summary dir', () => {
    expect(isScanSummaryWellPath('A1')).toBe(true);
    expect(isScanSummaryWellPath('folder/A12')).toBe(true);
    expect(scanSummaryDirForFilePath('A1')).toBe('');
    expect(scanSummaryDirForFilePath('run/A3')).toBe('run');
    expect(isScanSummaryWorkspaceRoot([{ name: 'imageMetadata.json', is_dir: false }])).toBe(true);
  });
});
