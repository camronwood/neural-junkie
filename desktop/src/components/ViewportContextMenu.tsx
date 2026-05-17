import { useLayoutEffect, useRef, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

export interface ViewportContextMenuProps {
  x: number;
  y: number;
  onClose: () => void;
  children: ReactNode;
}

/** Renders a context menu in a portal so it is not clipped by panel overflow/transform. */
export function ViewportContextMenu({ x, y, onClose, children }: ViewportContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ x, y });

  useLayoutEffect(() => {
    const el = menuRef.current;
    const pad = 8;
    if (!el) {
      setPos({ x, y });
      return;
    }
    const w = el.offsetWidth;
    const h = el.offsetHeight;
    let nx = x;
    let ny = y;
    if (nx + w > window.innerWidth - pad) {
      nx = Math.max(pad, window.innerWidth - w - pad);
    }
    if (ny + h > window.innerHeight - pad) {
      ny = Math.max(pad, window.innerHeight - h - pad);
    }
    if (nx < pad) nx = pad;
    if (ny < pad) ny = pad;
    setPos({ x: nx, y: ny });
  }, [x, y]);

  return createPortal(
    <>
      <div
        className="fixed inset-0 z-[250]"
        onClick={onClose}
        onContextMenu={(e) => {
          e.preventDefault();
          onClose();
        }}
        aria-hidden
      />
      <div
        ref={menuRef}
        role="menu"
        className="fixed z-[251] min-w-[11rem] bg-slack-bg border border-slack-border rounded shadow-lg py-1"
        style={{ left: pos.x, top: pos.y }}
      >
        {children}
      </div>
    </>,
    document.body
  );
}
