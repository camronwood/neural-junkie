import type { CSSProperties } from 'react';

export function taskActionBtnStyle(color: string): CSSProperties {
  return {
    padding: '4px 8px',
    borderRadius: 4,
    border: '1px solid var(--border-color, #444)',
    background: 'transparent',
    color,
    fontSize: 11,
    cursor: 'pointer',
  };
}
