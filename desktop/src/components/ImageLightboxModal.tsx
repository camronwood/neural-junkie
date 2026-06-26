import { useEffect, useState } from 'react';
import { useShortcutOverlay } from '../shortcuts/useShortcutOverlay';

interface ImageLightboxModalProps {
  isOpen: boolean;
  onClose: () => void;
  src: string;
  alt?: string;
}

export function ImageLightboxModal({ isOpen, onClose, src, alt = '' }: ImageLightboxModalProps) {
  useShortcutOverlay('imageLightbox', isOpen, onClose);

  useEffect(() => {
    if (!isOpen) return;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = 'unset';
    };
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label="Image preview"
    >
      <button
        type="button"
        onClick={onClose}
        className="absolute top-4 right-4 z-10 p-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded-full transition-colors"
        title="Close (ESC)"
      >
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      <div
        className="relative max-w-[95vw] max-h-[95vh] p-4"
        onClick={(e) => e.stopPropagation()}
      >
        <img
          src={src}
          alt={alt}
          className="max-w-full max-h-[90vh] object-contain rounded border border-slack-border/40 shadow-2xl"
        />
      </div>
    </div>
  );
}

interface ChatClickableImageProps {
  src: string;
  alt?: string;
  className?: string;
  loading?: 'lazy' | 'eager';
}

/** Chat thumbnail that opens a fullscreen lightbox on click. */
export function ChatClickableImage({ src, alt = '', className = '', loading }: ChatClickableImageProps) {
  const [lightboxOpen, setLightboxOpen] = useState(false);
  return (
    <>
      <img
        src={src}
        alt={alt}
        loading={loading}
        className={`cursor-pointer hover:opacity-90 transition-opacity ${className}`}
        onClick={() => setLightboxOpen(true)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            setLightboxOpen(true);
          }
        }}
        role="button"
        tabIndex={0}
        title="Click to enlarge"
      />
      <ImageLightboxModal
        isOpen={lightboxOpen}
        onClose={() => setLightboxOpen(false)}
        src={src}
        alt={alt}
      />
    </>
  );
}
