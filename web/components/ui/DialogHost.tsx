"use client";

import { useEffect, useState } from "react";

import ChoiceDialog from "./ChoiceDialog";
import ConfirmDialog from "./ConfirmDialog";
import PromptDialog from "./PromptDialog";
import { resolveDialog, subscribeDialogs, type DialogRequest } from "./dialogs";

export default function DialogHost() {
  const [requests, setRequests] = useState<DialogRequest[]>([]);

  useEffect(() => {
    const unsubscribe = subscribeDialogs(setRequests);
    return unsubscribe;
  }, []);

  // Only render the top-most request to avoid stacking modals.
  const current = requests[requests.length - 1] ?? null;

  if (!current) return null;

  if (current.kind === "confirm") {
    return (
      <ConfirmDialog
        open
        title={current.options.title}
        message={current.options.message}
        confirmLabel={current.options.confirmLabel}
        cancelLabel={current.options.cancelLabel}
        variant={current.options.variant}
        onConfirm={() => resolveDialog(current.id, true)}
        onCancel={() => resolveDialog(current.id, false)}
      />
    );
  }

  if (current.kind === "choice") {
    return (
      <ChoiceDialog
        open
        title={current.options.title}
        message={current.options.message}
        options={current.options.options}
        onSelect={(label) => resolveDialog(current.id, label)}
        onCancel={() => resolveDialog(current.id, null)}
      />
    );
  }

  return (
    <PromptDialog
      open
      title={current.options.title}
      label={current.options.label}
      message={current.options.message}
      placeholder={current.options.placeholder}
      defaultValue={current.options.defaultValue}
      confirmLabel={current.options.confirmLabel}
      cancelLabel={current.options.cancelLabel}
      validate={current.options.validate}
      onConfirm={(value) => resolveDialog(current.id, value)}
      onCancel={() => resolveDialog(current.id, null)}
    />
  );
}
