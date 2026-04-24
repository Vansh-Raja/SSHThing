"use client";

import type { ReactNode } from "react";

import Modal from "./Modal";

export type ConfirmDialogProps = {
  open: boolean;
  title: ReactNode;
  message?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "default" | "danger";
  onConfirm: () => void;
  onCancel: () => void;
};

export default function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  variant = "default",
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  return (
    <Modal
      open={open}
      onClose={onCancel}
      title={title}
      footer={
        <div className="modal__actions">
          <button
            type="button"
            className="btn"
            onClick={onCancel}
            autoFocus={variant === "danger"}
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            className={`btn ${variant === "danger" ? "btn--danger" : "btn--primary"}`}
            onClick={onConfirm}
            autoFocus={variant !== "danger"}
          >
            {confirmLabel}
          </button>
        </div>
      }
    >
      {message ? <p className="modal__message">{message}</p> : null}
    </Modal>
  );
}
