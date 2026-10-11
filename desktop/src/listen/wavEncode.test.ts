import { describe, expect, it } from 'vitest';
import { downsampleMono, encodeWavMono } from './wavEncode';

describe('wavEncode', () => {
  it('writes a valid RIFF/WAVE header', () => {
    const samples = new Float32Array([0, 0.5, -0.5, 1]);
    const buf = encodeWavMono(samples, 16000);
    const bytes = new Uint8Array(buf);
    expect(String.fromCharCode(...bytes.slice(0, 4))).toBe('RIFF');
    expect(String.fromCharCode(...bytes.slice(8, 12))).toBe('WAVE');
    expect(buf.byteLength).toBe(44 + samples.length * 2);
  });

  it('downsamples roughly by ratio', () => {
    const input = new Float32Array(16000);
    input.fill(0.25);
    const out = downsampleMono(input, 16000, 8000);
    expect(out.length).toBe(8000);
  });
});
