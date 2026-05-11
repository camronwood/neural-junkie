import React from 'react';
import { useToastStore } from '../stores/toastStore';

interface ToastProps {
  toast: {
    id: string;
    type: 'success' | 'error' | 'warning' | 'info';
    title: string;
    message?: string;
    action?: {
      label: string;
      onClick: () => void;
    };
  };
}

const Toast: React.FC<ToastProps> = ({ toast }) => {
  const { removeToast } = useToastStore();

  const getToastStyles = () => {
    const baseStyles =
      'flex items-start gap-3 p-3 rounded-lg shadow-xl border max-w-sm w-full backdrop-blur-sm bg-slack-bgHover/95';

    switch (toast.type) {
      case 'success':
        return `${baseStyles} border-emerald-600/60 text-slack-text`;
      case 'error':
        return `${baseStyles} border-red-500/70 text-red-200`;
      case 'warning':
        return `${baseStyles} border-amber-500/60 text-amber-100`;
      case 'info':
        return `${baseStyles} border-slack-accent/70 text-slack-text`;
      default:
        return `${baseStyles} border-slack-border text-slack-text`;
    }
  };

  const getIcon = () => {
    switch (toast.type) {
      case 'success':
        return '✅';
      case 'error':
        return '❌';
      case 'warning':
        return '⚠️';
      case 'info':
        return 'ℹ️';
      default:
        return '📢';
    }
  };

  return (
    <div className={getToastStyles()} role="status">
      <div className="flex-shrink-0 pt-0.5" aria-hidden>
        <span className="text-base leading-none">{getIcon()}</span>
      </div>

      <div className="flex-1 min-w-0">
        <div className="font-medium text-sm leading-snug">{toast.title}</div>
        {toast.message && (
          <div className="mt-1 text-xs text-slack-textMuted leading-relaxed">{toast.message}</div>
        )}
        {toast.action && (
          <div className="mt-2">
            <button
              type="button"
              onClick={toast.action.onClick}
              className="text-xs font-medium text-slack-accent hover:text-white underline-offset-2 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent rounded"
            >
              {toast.action.label}
            </button>
          </div>
        )}
      </div>

      <div className="flex-shrink-0">
        <button
          type="button"
          onClick={() => removeToast(toast.id)}
          className="text-slack-textMuted hover:text-slack-text rounded p-1 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-slack-accent"
        >
          <span className="sr-only">Dismiss notification</span>
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20" aria-hidden>
            <path
              fillRule="evenodd"
              d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
              clipRule="evenodd"
            />
          </svg>
        </button>
      </div>
    </div>
  );
};

export const ToastContainer: React.FC = () => {
  const { toasts } = useToastStore();

  if (toasts.length === 0) return null;

  return (
    <div
      className="fixed top-4 right-4 z-[60] flex max-h-[calc(100vh-2rem)] flex-col gap-2 overflow-y-auto"
      aria-live="polite"
      aria-relevant="additions"
    >
      {toasts.map(toast => (
        <Toast key={toast.id} toast={toast} />
      ))}
    </div>
  );
};
