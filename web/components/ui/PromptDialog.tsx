"use client";

import { useEffect, useState, type FormEvent, type ReactNode } from "react";

import Modal from "./Modal";

export type PromptDialogProps = {
  open: boolean;
  title: ReactNode;
  label?: ReactNode;
  message?: ReactNode;
  placeholder?: string;
  defaultValue?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  validate?: (value: string) => string | null;
  onConfirm: (value: string) => void;
  onCancel: () => void;
};

export default function PromptDialog({
  open,
  title,
  label,
  message,
  placeholder,
  defaultValue = "",
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  validate,
  onConfirm,
  onCancel,
}: PromptDialogProps) {
  const [value, setValue] = useState(defaultValue);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setValue(defaultValue);
      setError(null);
    }
  }, [open, defaultValue]);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = value.trim();
    if (validate) {
      const validationError = validate(trimmed);
      if (validationError) {
        setError(validationError);
        return;
      }
    }
    onConfirm(trimmed);
  }

  return (
    <Modal
      open={open}
      onClose={onCancel}
      title={title}
      footer={
        <div className="modal__actions">
          <button type="button" className="btn" onClick={onCancel}>
            {cancelLabel}
          </button>
          <button
            type="submit"
            form="prompt-dialog-form"
            className="btn btn--primary"
          >
            {confirmLabel}
          </button>
        </div>
      }
    >
      {message ? <p className="modal__message">{message}</p> : null}
      <form id="prompt-dialog-form" className="stack" style={{ gap: 10 }} onSubmit={handleSubmit}>
        <label className="field">
          {label ? <span className="field__label">{label}</span> : null}
          <input
            className="field__input"
            value={value}
            autoFocus
            placeholder={placeholder}
            onChange={(e) => setValue(e.target.value)}
          />
        </label>
        {error ? <p className="modal__error">{error}</p> : null}
      </form>
    </Modal>
  );
}
