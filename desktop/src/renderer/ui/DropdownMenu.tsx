import {
  useState,
  useRef,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react';
import { createPortal } from 'react-dom';

export type MenuItemDef =
  | { kind: 'item'; label: string; onClick: () => void; danger?: boolean; disabled?: boolean }
  | { kind: 'separator' };

type DropdownMenuProps = {
  trigger: ReactNode;
  items: MenuItemDef[];
};

export default function DropdownMenu({ trigger, items }: DropdownMenuProps) {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const openMenu = useCallback(() => {
    if (!triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    setPos({ top: rect.bottom + 4, left: rect.left });
    setOpen(true);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: PointerEvent) => {
      if (!menuRef.current?.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('pointerdown', onPointerDown);
    return () => document.removeEventListener('pointerdown', onPointerDown);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open]);

  return (
    <>
      <div
        ref={triggerRef}
        onClick={(e) => { e.stopPropagation(); openMenu(); }}
        style={{ display: 'inline-flex' }}
      >
        {trigger}
      </div>
      {open &&
        createPortal(
          <div
            ref={menuRef}
            className="dropdown-menu"
            role="menu"
            style={{ position: 'fixed', top: pos.top, left: pos.left }}
          >
            {items.map((item, i) => {
              if (item.kind === 'separator') {
                return <div key={i} className="dropdown-menu__separator" role="separator" />;
              }
              return (
                <button
                  key={i}
                  type="button"
                  role="menuitem"
                  className={`dropdown-menu__item${item.danger ? ' dropdown-menu__item--danger' : ''}`}
                  disabled={item.disabled}
                  onClick={(e) => {
                    e.stopPropagation();
                    setOpen(false);
                    item.onClick();
                  }}
                >
                  {item.label}
                </button>
              );
            })}
          </div>,
          document.body,
        )}
    </>
  );
}
