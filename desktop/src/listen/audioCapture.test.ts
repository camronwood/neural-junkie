import { describe, expect, it, vi, beforeEach } from 'vitest';
import { listAudioInputDevices, startAudioCapture } from './audioCapture';

describe('listAudioInputDevices', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('returns empty when mediaDevices missing', async () => {
    const orig = navigator.mediaDevices;
    Object.defineProperty(navigator, 'mediaDevices', { value: undefined, configurable: true });
    await expect(listAudioInputDevices()).resolves.toEqual([]);
    Object.defineProperty(navigator, 'mediaDevices', { value: orig, configurable: true });
  });

  it('maps audioinput devices', async () => {
    const stop = vi.fn();
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        getUserMedia: vi.fn().mockResolvedValue({ getTracks: () => [{ stop }] }),
        enumerateDevices: vi.fn().mockResolvedValue([
          { kind: 'audioinput', deviceId: 'a', label: 'Mic A' },
          { kind: 'audiooutput', deviceId: 'b', label: 'Speakers' },
          { kind: 'audioinput', deviceId: 'c', label: '' },
        ]),
      },
    });
    const list = await listAudioInputDevices();
    expect(list).toEqual([
      { deviceId: 'a', label: 'Mic A' },
      { deviceId: 'c', label: 'Microphone 2' },
    ]);
    expect(stop).toHaveBeenCalled();
  });
});

describe('startAudioCapture', () => {
  it('does not proceed when getUserMedia rejects', async () => {
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        getUserMedia: vi.fn().mockRejectedValue(new Error('Permission denied')),
      },
    });
    await expect(
      startAudioCapture(undefined, { onChunk: () => {}, onError: () => {} })
    ).rejects.toThrow(/Microphone permission denied/);
  });

  it('surfaces missing selected device', async () => {
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        getUserMedia: vi.fn().mockRejectedValue(new Error('NotFoundError')),
      },
    });
    await expect(
      startAudioCapture('virtual-device', { onChunk: () => {}, onError: () => {} })
    ).rejects.toThrow(/selected audio input/);
  });
});
