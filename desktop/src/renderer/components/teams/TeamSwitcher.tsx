/**
 * TeamSwitcher — dropdown for selecting the active team, shown in the top bar
 * when the user is signed in and has at least one team.
 *
 * Supports inline up/down arrows to reorder teams.
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { toast } from '../../ui/toast';

type TeamSwitcherProps = {
  teams: TeamSummary[];
  activeTeamId: string | null;
  onSelect: (team: TeamSummary) => void;
  /** Called after a successful reorder so the parent can reload. */
  onReorder?: () => void;
};

export default function TeamSwitcher({ teams, activeTeamId, onSelect, onReorder }: TeamSwitcherProps) {
  const active = teams.find((t) => t.id === activeTeamId);
  const label = active?.name ?? 'Select team';

  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // Local optimistic order
  const [localOrder, setLocalOrder] = useState<TeamSummary[]>(teams);
  useEffect(() => { setLocalOrder(teams); }, [teams]);

  const openMenu = useCallback(() => {
    if (!triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    setPos({ top: rect.bottom + 4, left: rect.left });
    setOpen(true);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: PointerEvent) => {
      if (!menuRef.current?.contains(e.target as Node) && !triggerRef.current?.contains(e.target as Node)) {
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

  const handleSelect = useCallback((team: TeamSummary) => {
    onSelect(team);
    setOpen(false);
  }, [onSelect]);

  const handleMove = useCallback(async (teamId: string, direction: 'up' | 'down') => {
    const idx = localOrder.findIndex((t) => t.id === teamId);
    if (idx === -1) return;
    const next = direction === 'up' ? idx - 1 : idx + 1;
    if (next < 0 || next >= localOrder.length) return;

    const reordered = [...localOrder];
    const temp = reordered[idx]!;
    reordered[idx] = reordered[next]!;
    reordered[next] = temp;
    setLocalOrder(reordered);

    try {
      await window.sshthing.teamsReorder(reordered.map((t) => t.id));
      onReorder?.();
    } catch (err) {
      setLocalOrder(localOrder);
      const e = err as Error;
      toast.error(e.message ?? 'Failed to reorder teams');
    }
  }, [localOrder, onReorder]);

  if (teams.length === 0) return null;

  return (
    <>
      <div
        ref={triggerRef}
        onClick={(e) => { e.stopPropagation(); openMenu(); }}
        style={{ display: 'inline-flex' }}
      >
        <button
          type="button"
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            background: 'var(--paper-2)',
            border: '1px solid var(--line)',
            borderRadius: 4,
            padding: '4px 10px',
            fontSize: 12,
            fontWeight: 600,
            color: 'var(--ink)',
            cursor: 'pointer',
            whiteSpace: 'nowrap',
            maxWidth: 180,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {label}
          <span style={{ color: 'var(--muted)', fontSize: 10 }}>▾</span>
        </button>
      </div>

      {open && createPortal(
        <div
          ref={menuRef}
          className="dropdown-menu"
          role="menu"
          style={{ position: 'fixed', top: pos.top, left: pos.left, minWidth: 200 }}
        >
          {localOrder.map((team, idx) => (
            <div
              key={team.id}
              role="menuitem"
              className={`dropdown-menu__item${team.id === activeTeamId ? ' dropdown-menu__item--active' : ''}`}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                paddingRight: 4,
                cursor: 'pointer',
              }}
              onClick={() => handleSelect(team)}
            >
              <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {team.name}
              </span>
              <button
                type="button"
                title="Move up"
                disabled={idx === 0}
                onClick={(e) => { e.stopPropagation(); void handleMove(team.id, 'up'); }}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: idx === 0 ? 'default' : 'pointer',
                  opacity: idx === 0 ? 0.25 : 0.6,
                  padding: '0 2px',
                  fontSize: 10,
                  color: 'var(--ink)',
                  lineHeight: 1,
                }}
                aria-label="Move team up"
              >
                ▲
              </button>
              <button
                type="button"
                title="Move down"
                disabled={idx === localOrder.length - 1}
                onClick={(e) => { e.stopPropagation(); void handleMove(team.id, 'down'); }}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: idx === localOrder.length - 1 ? 'default' : 'pointer',
                  opacity: idx === localOrder.length - 1 ? 0.25 : 0.6,
                  padding: '0 2px',
                  fontSize: 10,
                  color: 'var(--ink)',
                  lineHeight: 1,
                }}
                aria-label="Move team down"
              >
                ▼
              </button>
            </div>
          ))}
        </div>,
        document.body,
      )}
    </>
  );
}
