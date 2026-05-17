import { useState, useEffect } from 'react';
import { useToastStore } from '../stores/toastStore';

interface EditorImagePreviewProps {
  src: string;
  alt: string;
  reloadKey: number;
}

export function EditorImagePreview({ src, alt, reloadKey }: EditorImagePreviewProps) {
  const { addToast } = useToastStore();
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setFailed(false);
  }, [reloadKey, src]);

  const handleError = () => {
    setFailed(true);
    addToast({
      type: 'error',
      title: 'Could not load image',
      message: `Failed to load preview for ${alt}.`,
    });
  };

  if (failed) {
    return (
      <div className="flex items-center justify-center h-full text-slack-textMuted p-6">
        <div className="text-center max-w-sm">
          <div className="text-4xl mb-3">🖼️</div>
          <div className="text-sm font-medium mb-1">Image preview unavailable</div>
          <div className="text-xs break-all">{alt}</div>
        </div>
      </div>
    );
  }

  return (
    <div
      className="h-full w-full overflow-auto flex items-center justify-center p-4"
      style={{
        backgroundImage:
          'linear-gradient(45deg, #2a2d31 25%, transparent 25%), linear-gradient(-45deg, #2a2d31 25%, transparent 25%), linear-gradient(45deg, transparent 75%, #2a2d31 75%), linear-gradient(-45deg, transparent 75%, #2a2d31 75%)',
        backgroundSize: '20px 20px',
        backgroundPosition: '0 0, 0 10px, 10px -10px, -10px 0',
        backgroundColor: '#1a1d21',
      }}
    >
      <img
        key={reloadKey}
        src={src}
        alt={alt}
        onError={handleError}
        className="max-w-full max-h-full object-contain shadow-lg"
      />
    </div>
  );
}
