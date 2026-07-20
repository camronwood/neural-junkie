import { useEffect, useState } from 'react';
import type { PackToolbarAction } from '../stores/packCapabilityRegistry';
import { hubAuthHeaders, hubSessionHeaders } from '../config/hubUrl';

const iconBtn =
  'w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2';

interface CustomPackToolbarChipProps {
  action: PackToolbarAction;
  onClick: () => void;
}

/**
 * Pack chip icons are hub HTTP URLs. Tauri release CSP blocks those in <img src>,
 * so we fetch via connect-src (allowed) and render a blob/data URL instead.
 */
export function CustomPackToolbarChip({ action, onClick }: CustomPackToolbarChipProps) {
  const [iconSrc, setIconSrc] = useState<string | null>(null);

  useEffect(() => {
    const url = action.iconUrl?.trim();
    if (!url) {
      setIconSrc(null);
      return;
    }

    // https icons are allowed by CSP img-src directly.
    if (/^https:\/\//i.test(url)) {
      setIconSrc(url);
      return;
    }

    let cancelled = false;
    let objectUrl: string | null = null;

    void (async () => {
      try {
        const res = await fetch(url, {
          headers: { ...hubAuthHeaders(), ...hubSessionHeaders() },
        });
        if (!res.ok) throw new Error(`icon ${res.status}`);
        const blob = await res.blob();
        objectUrl = URL.createObjectURL(blob);
        if (cancelled) {
          URL.revokeObjectURL(objectUrl);
          objectUrl = null;
          return;
        }
        setIconSrc(objectUrl);
      } catch {
        if (!cancelled) setIconSrc(null);
      }
    })();

    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [action.iconUrl]);

  const showImg = Boolean(iconSrc);
  const label = action.label || '?';

  return (
    <button
      type="button"
      onClick={onClick}
      className={`${iconBtn} bg-teal-700 hover:bg-teal-600 text-white text-[10px] font-bold focus-visible:outline-teal-400 overflow-hidden`}
      title={action.title}
      aria-label={action.title}
    >
      {showImg ? (
        <img src={iconSrc!} alt="" className="h-5 w-5 object-contain" draggable={false} />
      ) : (
        label
      )}
    </button>
  );
}
