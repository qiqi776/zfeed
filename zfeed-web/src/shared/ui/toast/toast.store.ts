import { create } from "zustand";

export type ToastTone = "info" | "success" | "error";

export type ToastInput = {
  title: string;
  description?: string;
  tone?: ToastTone;
  durationMs?: number;
};

export type ToastItem = ToastInput & {
  id: string;
};

type ToastState = {
  toasts: ToastItem[];
  showToast: (input: ToastInput) => string;
  dismissToast: (id: string) => void;
  clearToasts: () => void;
};

let toastSequence = 0;

function nextToastId() {
  toastSequence += 1;
  return `toast-${toastSequence}`;
}

export const useToastStore = create<ToastState>()((set) => ({
  toasts: [],
  showToast: ({ title, description, tone = "info", durationMs = 3600 }) => {
    const id = nextToastId();
    const nextToast: ToastItem = {
      id,
      title,
      description,
      tone,
      durationMs,
    };

    set((state) => ({
      toasts: [...state.toasts, nextToast].slice(-4),
    }));

    return id;
  },
  dismissToast: (id) => {
    set((state) => ({
      toasts: state.toasts.filter((toast) => toast.id !== id),
    }));
  },
  clearToasts: () => {
    set({ toasts: [] });
  },
}));

export function useToast() {
  const showToast = useToastStore((state) => state.showToast);
  const dismissToast = useToastStore((state) => state.dismissToast);

  return { showToast, dismissToast };
}
