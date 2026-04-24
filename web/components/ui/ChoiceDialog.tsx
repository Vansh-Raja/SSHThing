"use client";

import { useEffect, useRef, type ReactNode } from "react";

import Modal from "./Modal";
import type { ChoiceOption } from "./dialogs";

export type ChoiceDialogProps = {
  open: boolean;
  title: ReactNode;
  message?: ReactNode;
  options: ChoiceOption[];
  onSelect: (label: string) => void;
  onCancel: () => void;
};

function classForVariant(variant: ChoiceOption["variant"]) {
  if (variant === "danger") return "btn btn--danger";
  if (variant === "primary") return "btn btn--primary";
  return "btn";
}

export default function ChoiceDialog({
  open,
  title,
  message,
  options,
  onSelect,
  onCancel,
}: ChoiceDialogProps) {
  const primaryRef = useRef<HTMLButtonElement | null>(null);

  // Enter triggers the last (primary-by-convention) option.
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Enter") return;
      const target = e.target as HTMLElement | null;
      // Don't hijack Enter inside textareas / multiline inputs.
      if (target && target.tagName === "TEXTAREA") return;
      e.preventDefault();
      const last = options[options.length - 1];
      if (last) onSelect(last.label);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, options, onSelect]);

  return (
    <Modal
      open={open}
      onClose={onCancel}
      title={title}
      footer={
        <div className="modal__actions">
          {options.map((opt, idx) => {
            const isPrimary = idx === options.length - 1;
            return (
              <button
                key={opt.label}
                ref={isPrimary ? primaryRef : undefined}
                type="button"
                className={classForVariant(opt.variant)}
                onClick={() => onSelect(opt.label)}
                autoFocus={isPrimary}
              >
                {opt.label}
              </button>
            );
          })}
        </div>
      }
    >
      {message ? <p className="modal__message">{message}</p> : null}
    </Modal>
  );
}
