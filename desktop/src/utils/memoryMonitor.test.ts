import { describe, expect, it } from 'vitest';
import {
  formatMemoryBytes,
  memoryPressureLevel,
  shortModelTag,
} from './memoryMonitor';

describe('formatMemoryBytes', () => {
  it('formats gigabytes', () => {
    expect(formatMemoryBytes(17 * 1024 ** 3)).toBe('17 GB');
  });

  it('formats megabytes', () => {
    expect(formatMemoryBytes(512 * 1024 ** 2)).toBe('512 MB');
  });
});

describe('memoryPressureLevel', () => {
  it('classifies pressure bands', () => {
    expect(memoryPressureLevel(50)).toBe('ok');
    expect(memoryPressureLevel(75)).toBe('warn');
    expect(memoryPressureLevel(90)).toBe('critical');
  });
});

describe('shortModelTag', () => {
  it('shortens long model names', () => {
    expect(shortModelTag('qwen3.5:27b')).toBe('qwen3.5:27b');
    expect(shortModelTag('registry.example.com/org/very-long-model-name-tag:latest')).toContain('…');
  });
});
