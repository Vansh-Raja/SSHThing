"use client";

export type ToastVariant = "success" | "error" | "info";

export type ToastRecord = {
  id: number;
  message: string;
  variant: ToastVariant;
};

type Listener = (toasts: ToastRecord[]) => void;

const AUTO_DISMISS_MS = 4000;

let nextId = 1;
let queue: ToastRecord[] = [];
const listeners = new Set<Listener>();

function emit() {
  const snapshot = queue.slice();
  for (const listener of listeners) {
    listener(snapshot);
  }
}

export function subscribeToasts(listener: Listener): () => void {
  listeners.add(listener);
  listener(queue.slice());
  return () => {
    listeners.delete(listener);
  };
}

export function dismissToast(id: number) {
  const next = queue.filter((t) => t.id !== id);
  if (next.length === queue.length) return;
  queue = next;
  emit();
}

function pushToast(message: string, variant: ToastVariant): number {
  const id = nextId++;
  queue = [...queue, { id, message, variant }];
  emit();
  if (typeof window !== "undefined") {
    window.setTimeout(() => dismissToast(id), AUTO_DISMISS_MS);
  }
  return id;
}

export const toast = {
  success(message: string) {
    return pushToast(message, "success");
  },
  error(message: string) {
    return pushToast(message, "error");
  },
  info(message: string) {
    return pushToast(message, "info");
  },
};

/** Hook form — rarely needed since `toast.*` works anywhere. */
export function useToast() {
  return toast;
}
