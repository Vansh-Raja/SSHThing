"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";

type MenuCtx = {
  close: () => void;
};

const DropdownCtx = createContext<MenuCtx | null>(null);

type Align = "start" | "end";

type DropdownMenuProps = {
  /** Visible content of the trigger button (the menu wraps it in a <button>). */
  trigger: ReactNode;
  /** className applied to the internal <button> trigger. */
  triggerClassName?: string;
  /** aria-label for the trigger button (required when trigger is icon-only). */
  triggerAriaLabel?: string;
  /** Horizontal alignment of the menu relative to the trigger. */
  align?: Align;
  /** The menu items (use DropdownMenu.Item / DropdownMenu.Separator). */
  children: ReactNode;
};

type Position = { top: number; left?: number; right?: number };

function DropdownMenuRoot({
  trigger,
  triggerClassName,
  triggerAriaLabel,
  align = "start",
  children,
}: DropdownMenuProps) {
  const [open, setOpen] = useState(false);
  const [mounted, setMounted] = useState(false);
  const [position, setPosition] = useState<Position | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const menuId = useId();

  useEffect(() => {
    setMounted(true);
  }, []);

  const computePosition = useCallback(() => {
    const trig = triggerRef.current;
    if (!trig) return;
    const rect = trig.getBoundingClientRect();
    if (align === "end") {
      setPosition({
        top: rect.bottom + 4,
        right: Math.max(8, window.innerWidth - rect.right),
      });
    } else {
      setPosition({
        top: rect.bottom + 4,
        left: Math.max(8, rect.left),
      });
    }
  }, [align]);

  useEffect(() => {
    if (!open) {
      setPosition(null);
      return;
    }
    computePosition();
    const onResize = () => computePosition();
    window.addEventListener("resize", onResize);
    window.addEventListener("scroll", onResize, true);
    return () => {
      window.removeEventListener("resize", onResize);
      window.removeEventListener("scroll", onResize, true);
    };
  }, [open, computePosition]);

  useEffect(() => {
    if (!open) return;
    const onPointer = (e: PointerEvent) => {
      const target = e.target as Node;
      if (menuRef.current?.contains(target)) return;
      if (triggerRef.current?.contains(target)) return;
      setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        setOpen(false);
        triggerRef.current?.focus();
      }
    };
    document.addEventListener("pointerdown", onPointer);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onPointer);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  // Focus first item on open.
  useEffect(() => {
    if (!open || !position) return;
    const raf = requestAnimationFrame(() => {
      const first = menuRef.current?.querySelector<HTMLButtonElement>(
        '[role="menuitem"]:not([aria-disabled="true"])',
      );
      first?.focus();
    });
    return () => cancelAnimationFrame(raf);
  }, [open, position]);

  const close = useCallback(() => {
    setOpen(false);
    triggerRef.current?.focus();
  }, []);

  const onTriggerKey = (e: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (e.key === "ArrowDown" || e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      setOpen(true);
    }
  };

  const onMenuKeyDown = (e: ReactKeyboardEvent<HTMLDivElement>) => {
    const menu = menuRef.current;
    if (!menu) return;
    const items = Array.from(
      menu.querySelectorAll<HTMLButtonElement>(
        '[role="menuitem"]:not([aria-disabled="true"])',
      ),
    );
    if (items.length === 0) return;
    const activeIdx = items.findIndex((el) => el === document.activeElement);
    if (e.key === "ArrowDown") {
      e.preventDefault();
      items[(activeIdx + 1) % items.length]?.focus();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      items[(activeIdx - 1 + items.length) % items.length]?.focus();
    } else if (e.key === "Home") {
      e.preventDefault();
      items[0]?.focus();
    } else if (e.key === "End") {
      e.preventDefault();
      items[items.length - 1]?.focus();
    } else if (e.key === "Tab") {
      setOpen(false);
    }
  };

  const ctx = useMemo<MenuCtx>(() => ({ close }), [close]);

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        className={triggerClassName}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={triggerAriaLabel}
        onClick={(e) => {
          e.stopPropagation();
          setOpen((v) => !v);
        }}
        onKeyDown={onTriggerKey}
      >
        {trigger}
      </button>
      {mounted && open && position
        ? createPortal(
            <div
              ref={menuRef}
              id={menuId}
              role="menu"
              className="dropdown-menu"
              style={{
                position: "fixed",
                top: position.top,
                left: position.left,
                right: position.right,
              }}
              onKeyDown={onMenuKeyDown}
            >
              <DropdownCtx.Provider value={ctx}>{children}</DropdownCtx.Provider>
            </div>,
            document.body,
          )
        : null}
    </>
  );
}

type ItemProps = {
  onSelect: () => void;
  variant?: "default" | "danger";
  disabled?: boolean;
  /** Additional class names, e.g. for active state or layout variants. */
  className?: string;
  children: ReactNode;
};

function Item({
  onSelect,
  variant = "default",
  disabled,
  className,
  children,
}: ItemProps) {
  const ctx = useContext(DropdownCtx);
  const classes = [
    "dropdown-menu__item",
    variant === "danger" ? "dropdown-menu__item--danger" : "",
    className ?? "",
  ]
    .filter(Boolean)
    .join(" ");
  return (
    <button
      type="button"
      role="menuitem"
      className={classes}
      aria-disabled={disabled || undefined}
      disabled={disabled}
      onClick={(e) => {
        e.stopPropagation();
        if (disabled) return;
        ctx?.close();
        onSelect();
      }}
    >
      {children}
    </button>
  );
}

function Separator() {
  return <div className="dropdown-menu__separator" role="separator" aria-orientation="horizontal" />;
}

function Label({ children }: { children: ReactNode }) {
  return <div className="dropdown-menu__label">{children}</div>;
}

const DropdownMenu = Object.assign(DropdownMenuRoot, {
  Item,
  Separator,
  Label,
});

export default DropdownMenu;
