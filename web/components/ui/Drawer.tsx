"use client";

import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";

type DrawerProps = {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  /** When false, backdrop click + Esc do nothing. Default true. */
  dismissible?: boolean;
  width?: number;
  children: ReactNode;
  /** Optional footer rendered below the body (sticky bottom). */
  footer?: ReactNode;
};

function focusableIn(root: HTMLElement): HTMLElement[] {
  const selector = [
    "a[href]",
    "button:not([disabled])",
    "input:not([disabled]):not([type='hidden'])",
    "select:not([disabled])",
    "textarea:not([disabled])",
    "[tabindex]:not([tabindex='-1'])",
  ].join(",");
  return Array.from(root.querySelectorAll<HTMLElement>(selector)).filter(
    (el) => el.offsetParent !== null || el === document.activeElement,
  );
}

export default function Drawer({
  open,
  onClose,
  title,
  dismissible = true,
  width = 520,
  children,
  footer,
}: DrawerProps) {
  const [mounted, setMounted] = useState(false);
  /** `entered` drives the CSS transition from off-screen to on-screen. */
  const [entered, setEntered] = useState(false);
  const panelRef = useRef<HTMLDivElement | null>(null);
  const previousActive = useRef<HTMLElement | null>(null);
  const titleId = useId();

  useEffect(() => {
    setMounted(true);
  }, []);

  // Body scroll lock.
  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prev;
    };
  }, [open]);

  // Remember previously-focused element and restore on close.
  useEffect(() => {
    if (open) {
      previousActive.current = document.activeElement as HTMLElement | null;
    } else if (previousActive.current) {
      previousActive.current.focus();
      previousActive.current = null;
    }
  }, [open]);

  // Drive enter transition after mount so CSS interpolates.
  useEffect(() => {
    if (!open) {
      setEntered(false);
      return;
    }
    const raf = requestAnimationFrame(() => setEntered(true));
    return () => cancelAnimationFrame(raf);
  }, [open]);

  // Focus trap + Esc.
  useEffect(() => {
    if (!open) return;

    // Focus first focusable inside the panel, skipping the header close
    // button so the caret lands on the first real input instead.
    const raf = requestAnimationFrame(() => {
      const panel = panelRef.current;
      if (!panel) return;
      const focusables = focusableIn(panel);
      const first =
        focusables.find((el) => !el.classList.contains("drawer__close")) ??
        focusables[0] ??
        panel;
      first.focus();
    });

    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && dismissible) {
        e.stopPropagation();
        onClose();
        return;
      }
      if (e.key === "Tab") {
        const panel = panelRef.current;
        if (!panel) return;
        const focusables = focusableIn(panel);
        if (focusables.length === 0) {
          e.preventDefault();
          panel.focus();
          return;
        }
        const first = focusables[0];
        const last = focusables[focusables.length - 1];
        const active = document.activeElement as HTMLElement | null;
        if (e.shiftKey && active === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && active === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    document.addEventListener("keydown", onKey);
    return () => {
      cancelAnimationFrame(raf);
      document.removeEventListener("keydown", onKey);
    };
  }, [open, dismissible, onClose]);

  const onBackdropClick = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (e.target !== e.currentTarget) return;
      if (!dismissible) return;
      onClose();
    },
    [dismissible, onClose],
  );

  if (!mounted || !open) return null;

  return createPortal(
    <div
      className={`drawer${entered ? " drawer--entered" : ""}`}
      role="presentation"
      onMouseDown={onBackdropClick}
    >
      <div className="drawer__backdrop" aria-hidden="true" />
      <div
        ref={panelRef}
        className="drawer__panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        style={{ width: `min(${width}px, 92vw)` }}
      >
        <div className="drawer__header">
          <h2 id={titleId} className="drawer__title">
            {title}
          </h2>
          <button
            type="button"
            className="drawer__close"
            aria-label="Close"
            onClick={onClose}
          >
            ×
          </button>
        </div>
        <div className="drawer__body">{children}</div>
        {footer ? <div className="drawer__footer">{footer}</div> : null}
      </div>
    </div>,
    document.body,
  );
}
