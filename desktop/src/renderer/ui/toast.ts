export type ToastVariant = 'success' | 'error' | 'info' | 'warning';

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

export function dismissToast(id: number): void {
  const next = queue.filter((t) => t.id !== id);
  if (next.length === queue.length) return;
  queue = next;
  emit();
}

function pushToast(message: string, variant: ToastVariant): number {
  const id = nextId++;
  queue = [...queue, { id, message, variant }];
  emit();
  window.setTimeout(() => dismissToast(id), AUTO_DISMISS_MS);
  return id;
}

export const toast = {
  success(message: string): number {
    return pushToast(message, 'success');
  },
  error(message: string): number {
    return pushToast(message, 'error');
  },
  info(message: string): number {
    return pushToast(message, 'info');
  },
  warning(message: string): number {
    return pushToast(message, 'warning');
  },
};
