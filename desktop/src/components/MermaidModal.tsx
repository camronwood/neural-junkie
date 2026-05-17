import { useEffect } from 'react';
import { MermaidCanvas } from './MermaidCanvas';

interface MermaidModalProps {
  isOpen: boolean;
  onClose: () => void;
  content: string;
}

export function MermaidModal({ isOpen, onClose, content }: MermaidModalProps) {
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm"
      onClick={onClose}
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
        className="relative w-[95vw] h-[95vh] overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <MermaidCanvas content={content} active={isOpen} className="w-full h-full" />
      </div>
    </div>
  );
}
