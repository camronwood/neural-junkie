import { describe, expect, it } from 'vitest';
import {
  detectHubDataAccessNeeds,
  hasGrantedHubDataAccess,
} from './hubDataAccess';
import { GRANTED_HUB_DATA_ACCESS_KEY } from '../constants/promptMetadata';

describe('detectHubDataAccessNeeds', () => {
  it('detects session file mentions', () => {
    const opts = detectHubDataAccessNeeds('please review last-session.json');
    expect(opts.some((o) => o.relativePath === 'last-session.json' && o.kind === 'file')).toBe(
      true
    );
  });

  it('detects whole hub directory mentions', () => {
    const opts = detectHubDataAccessNeeds('list everything in ~/.neural-junkie');
    expect(opts.some((o) => o.kind === 'directory' && o.relativePath === '')).toBe(true);
  });

  it('detects specific files under hub path', () => {
    const opts = detectHubDataAccessNeeds('open ~/.neural-junkie/config.json');
    expect(opts.some((o) => o.relativePath === 'config.json')).toBe(true);
  });

  it('detects full filesystem paths to hub files', () => {
    const opts = detectHubDataAccessNeeds(
      'review /Users/camronwood/.neural-junkie/last-session.json'
    );
    expect(opts.some((o) => o.relativePath === 'last-session.json')).toBe(true);
  });

  it('dedupes file and directory when both match', () => {
    const opts = detectHubDataAccessNeeds(
      'scan ~/.neural-junkie and read last-session.json'
    );
    const keys = opts.map((o) => `${o.kind}:${o.relativePath}`);
    expect(new Set(keys).size).toBe(keys.length);
  });
});

describe('hasGrantedHubDataAccess', () => {
  it('is false without grant metadata', () => {
    expect(hasGrantedHubDataAccess({})).toBe(false);
  });

  it('is true when grant payload has entries', () => {
    expect(
      hasGrantedHubDataAccess({
        [GRANTED_HUB_DATA_ACCESS_KEY]: { entries: [{ path: 'x' }] },
      })
    ).toBe(true);
  });
});
