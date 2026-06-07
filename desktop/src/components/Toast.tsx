import React from 'react';
import { useToastStore, type Toast as ToastModel } from '../stores/toastStore';
import { SlackIcon } from './icons/SlackIcon';

interface ToastProps {
  toast: ToastModel;
}

function ToastIcon({ type, variant }: { type: ToastModel['type']; variant?: ToastModel['variant'] }) {
  if (variant === 'slack') {
    return (
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-[#4A154B]/30 ring-1 ring-[#E01E5A]/40">
        <SlackIcon className="h-4 w-4 text-[#E8DEE8]" />
      </div>
    );
  }

  const base = 'flex h-8 w-8 shrink-0 items-center justify-center rounded-md ring-1';
  switch (type) {
    case 'success':
      return (
        <div className={`${base} bg-emerald-950/50 ring-emerald-500/40 text-emerald-300`}>
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
            <path
              fillRule="evenodd"
              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
              clipRule="evenodd"
            />
          </svg>
        </div>
      );
    case 'error':
      return (
        <div className={`${base} bg-red-950/50 ring-red-500/40 text-red-300`}>
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
            <path
              fillRule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
              clipRule="evenodd"
            />
          </svg>
        </div>
      );
    case 'warning':
      return (
        <div className={`${base} bg-amber-950/40 ring-amber-500/40 text-amber-200`}>
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
            <path
              fillRule="evenodd"
              d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
        </div>
      );
    default:
      return (
        <div className={`${base} bg-slack-accent/20 ring-slack-accent/50 text-sky-200`}>
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
            <path
              fillRule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
              clipRule="evenodd"
            />
          </svg>
        </div>
      );
  }
}

const Toast: React.FC<ToastProps> = ({ toast }) => {
  const { removeToast } = useToastStore();
  const isSlack = toast.variant === 'slack';
  const clickable = !!toast.action;

  const shellClass = isSlack
    ? 'border-slack-border bg-slack-bg shadow-[0_8px_32px_rgba(0,0,0,0.45)] ring-1 ring-[#E01E5A]/20'
    : 'border-slack-border bg-slack-bg shadow-[0_8px_32px_rgba(0,0,0,0.45)] ring-1 ring-white/5';

  const handleActivate = () => {
    toast.action?.onClick();
    removeToast(toast.id);
  };

  return (
    <div
      className={`flex max-w-md w-full items-start gap-3 rounded-lg border p-3 backdrop-blur-md ${shellClass} ${
        clickable ? 'cursor-pointer hover:bg-slack-bgHover/90 transition-colors' : ''
      }`}
      role={clickable ? 'button' : 'status'}
      tabIndex={clickable ? 0 : undefined}
      onClick={clickable ? handleActivate : undefined}
      onKeyDown={
        clickable
          ? (e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                handleActivate();
              }
            }
          : undefined
      }
    >
      <ToastIcon type={toast.type} variant={toast.variant} />

      <div className="min-w-0 flex-1">
        <div className="text-sm font-semibold leading-snug text-slack-text">{toast.title}</div>
        {toast.message && (
          <div className="mt-1 line-clamp-3 text-xs leading-relaxed text-slack-textMuted">{toast.message}</div>
        )}
        {toast.action && (
          <div className="mt-2.5">
            <span className="inline-flex items-center rounded-md bg-slack-accent px-2.5 py-1 text-xs font-medium text-white shadow-sm">
              {toast.action.label}
            </span>
          </div>
        )}
      </div>

      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          removeToast(toast.id);
        }}
        className="shrink-0 rounded p-1 text-slack-textMuted transition-colors hover:bg-white/10 hover:text-slack-text focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-slack-accent"
        aria-label="Dismiss notification"
      >
        <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20" aria-hidden>
          <path
            fillRule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
      </button>
    </div>
  );
};

export const ToastContainer: React.FC = () => {
  const { toasts } = useToastStore();

  if (toasts.length === 0) return null;

  return (
    <div
      className="pointer-events-none fixed bottom-4 right-4 z-[60] flex max-h-[min(70vh,28rem)] w-[min(100vw-2rem,24rem)] flex-col-reverse gap-2 overflow-y-auto"
      aria-live="polite"
      aria-relevant="additions"
    >
      {toasts.map((toast) => (
        <div key={toast.id} className="pointer-events-auto">
          <Toast toast={toast} />
        </div>
      ))}
    </div>
  );
};
