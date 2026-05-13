import * as matchers from '@testing-library/jest-dom/matchers';
import { expect } from 'vitest';

expect.extend(matchers);

/** Zustand chatStore reads `last-channel` at module load; ensure a minimal localStorage in node. */
const memoryStore: Record<string, string> = {};
const ls = {
  getItem: (k: string) => (k in memoryStore ? memoryStore[k] : null),
  setItem: (k: string, v: string) => {
    memoryStore[k] = v;
  },
  removeItem: (k: string) => {
    delete memoryStore[k];
  },
  clear: () => {
    for (const k of Object.keys(memoryStore)) delete memoryStore[k];
  },
  key: (_i: number) => null as string | null,
  get length() {
    return Object.keys(memoryStore).length;
  },
};
Object.defineProperty(globalThis, 'localStorage', { value: ls, writable: true });
