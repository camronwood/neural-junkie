import { create } from 'zustand';

export interface Toast {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  duration?: number; // Auto-dismiss after this many ms (0 = no auto-dismiss)
  action?: {
    label: string;
    onClick: () => void;
  };
}

interface ToastState {
  toasts: Toast[];
  addToast: (toast: Omit<Toast, 'id'>) => string;
  removeToast: (id: string) => void;
  clearAllToasts: () => void;
}

export const useToastStore = create<ToastState>((set, get) => ({
  toasts: [],
  
  addToast: (toast) => {
    const id = `toast_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const newToast: Toast = {
      id,
      duration: 5000, // Default 5 seconds
      ...toast,
    };
    
    set(state => ({
      toasts: [...state.toasts, newToast],
    }));
    
    // Auto-dismiss if duration is set
    if (newToast.duration && newToast.duration > 0) {
      setTimeout(() => {
        get().removeToast(id);
      }, newToast.duration);
    }
    
    return id;
  },
  
  removeToast: (id) => {
    set(state => ({
      toasts: state.toasts.filter(toast => toast.id !== id),
    }));
  },
  
  clearAllToasts: () => {
    set({ toasts: [] });
  },
}));
