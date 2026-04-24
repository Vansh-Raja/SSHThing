"use client";

import { useEffect, useState } from "react";
import { createPortal } from "react-dom";

import { dismissToast, subscribeToasts, type ToastRecord } from "./toast";

export default function Toaster() {
  const [toasts, setToasts] = useState<ToastRecord[]>([]);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    const unsubscribe = subscribeToasts(setToasts);
    return unsubscribe;
  }, []);

  if (!mounted) return null;

  return createPortal(
    <div className="toaster" aria-live="polite" aria-atomic="false">
      {toasts.map((t) => (
        <button
          key={t.id}
          className={`toast toast--${t.variant}`}
          role={t.variant === "error" ? "alert" : "status"}
          type="button"
          onClick={() => dismissToast(t.id)}
        >
          {t.message}
        </button>
      ))}
    </div>,
    document.body,
  );
}
