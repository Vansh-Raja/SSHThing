"use client";

import type { ReactNode } from "react";

export type ConfirmOptions = {
  title: ReactNode;
  message?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "default" | "danger";
};

export type PromptOptions = {
  title: ReactNode;
  label?: ReactNode;
  message?: ReactNode;
  placeholder?: string;
  defaultValue?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  validate?: (value: string) => string | null;
};

export type ChoiceOption = {
  label: string;
  variant?: "primary" | "default" | "danger";
};

export type ChoiceOptions = {
  title: ReactNode;
  message?: ReactNode;
  /** Rendered in order, left to right. Enter triggers the last (primary) option. */
  options: ChoiceOption[];
};

type ConfirmRequest = {
  kind: "confirm";
  id: number;
  options: ConfirmOptions;
  resolve: (ok: boolean) => void;
};

type PromptRequest = {
  kind: "prompt";
  id: number;
  options: PromptOptions;
  resolve: (value: string | null) => void;
};

type ChoiceRequest = {
  kind: "choice";
  id: number;
  options: ChoiceOptions;
  resolve: (label: string | null) => void;
};

export type DialogRequest = ConfirmRequest | PromptRequest | ChoiceRequest;

type Listener = (requests: DialogRequest[]) => void;

let nextId = 1;
let queue: DialogRequest[] = [];
const listeners = new Set<Listener>();

function emit() {
  const snapshot = queue.slice();
  for (const listener of listeners) {
    listener(snapshot);
  }
}

export function subscribeDialogs(listener: Listener): () => void {
  listeners.add(listener);
  listener(queue.slice());
  return () => {
    listeners.delete(listener);
  };
}

function push(request: DialogRequest) {
  queue = [...queue, request];
  emit();
}

export function resolveDialog(id: number, value: boolean | string | null) {
  const request = queue.find((r) => r.id === id);
  if (!request) return;
  queue = queue.filter((r) => r.id !== id);
  emit();
  if (request.kind === "confirm") {
    request.resolve(Boolean(value));
  } else {
    // prompt + choice both resolve to string | null
    request.resolve((value as string | null) ?? null);
  }
}

export function confirmDialog(options: ConfirmOptions): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    push({ kind: "confirm", id: nextId++, options, resolve });
  });
}

export function promptDialog(options: PromptOptions): Promise<string | null> {
  return new Promise<string | null>((resolve) => {
    push({ kind: "prompt", id: nextId++, options, resolve });
  });
}

/**
 * Three-way (or N-way) choice dialog. Resolves to the selected option's
 * `label`, or `null` on Esc / backdrop click / × button.
 */
export function choiceDialog(options: ChoiceOptions): Promise<string | null> {
  return new Promise<string | null>((resolve) => {
    push({ kind: "choice", id: nextId++, options, resolve });
  });
}
