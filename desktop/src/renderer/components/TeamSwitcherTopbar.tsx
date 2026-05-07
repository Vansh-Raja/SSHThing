/**
 * TeamSwitcherTopbar — compact team picker for the AppShell topbar.
 *
 * Shows the active team name with a dropdown listing all teams in order.
 * Supports:
 *   - switching teams (updates TeamContext)
 *   - reordering via up/down arrows in the dropdown
 *   - "Create team" action that opens a modal
 *   - "Manage teams" link that navigates to /teams
 *
 * Only rendered when signed in and at least one team exists.
 */
import { useCallback, useRef, useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { useNavigate } from 'react-router-dom';
import { useTeamContext } from '../contexts/TeamContext';
import Modal from '../ui/Modal';
import { toast } from '../ui/toast';

type TeamSwitcherTopbarProps = {
  teams: TeamSummary[];
  onTeamsChange?: () => void;
};

export default function TeamSwitcherTopbar({ teams, onTeamsChange }: TeamSwitcherTopbarProps) {
  const navigate = useNavigate();
  const { activeTeamId, setActiveTeamId } = useTeamContext();
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // Create team modal state
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState('');
  const [creating, setCreating] = useState(false);

  // Reorder optimistic local state (mirrors teams prop until server confirms)
  const [localOrder, setLocalOrder] = useState<TeamSummary[]>(teams);
  useEffect(() => { setLocalOrder(teams); }, [teams]);

  const sorted = [...localOrder].sort((a, b) => a.displayOrder - b.displayOrder);
  // activeTeamId === null means "Personal" mode (no team selected)
  const active = activeTeamId === null ? null : (sorted.find((t) => t.id === activeTeamId) ?? sorted[0] ?? null);

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
    setActiveTeamId(team.id);
    setOpen(false);
    navigate('/teams');
  }, [setActiveTeamId, navigate]);

  const handleReorder = useCallback(async (teamId: string, direction: 'up' | 'down') => {
    const idx = localOrder.findIndex((t) => t.id === teamId);
    if (idx === -1) return;
    const next = direction === 'up' ? idx - 1 : idx + 1;
    if (next < 0 || next >= localOrder.length) return;

    // Optimistic update
    const reordered = [...localOrder];
    const temp = reordered[idx]!;
    reordered[idx] = reordered[next]!;
    reordered[next] = temp;
    setLocalOrder(reordered);

    try {
      await window.sshthing.teamsReorder(reordered.map((t) => t.id));
      onTeamsChange?.();
    } catch (err) {
      // Revert on failure
      setLocalOrder(localOrder);
      const e = err as Error;
      toast.error(e.message ?? 'Failed to reorder teams');
    }
  }, [localOrder, onTeamsChange]);

  const handleCreateTeam = useCallback(async () => {
    const name = createName.trim();
    if (!name) return;
    setCreating(true);
    try {
      const team = await window.sshthing.teamsCreate(name);
      toast.success(`Team "${team.name}" created`);
      setActiveTeamId(team.id);
      setCreateOpen(false);
      setCreateName('');
      onTeamsChange?.();
      navigate('/teams');
    } catch (err) {
      const e = err as Error;
      toast.error(e.message ?? 'Failed to create team');
    } finally {
      setCreating(false);
    }
  }, [createName, setActiveTeamId, onTeamsChange, navigate]);

  if (sorted.length === 0) {
    return (
      <button
        type="button"
        className="btn btn--ghost"
        style={{ fontSize: 11, padding: '3px 8px' }}
        onClick={() => setCreateOpen(true)}
      >
        + New team
      </button>
    );
  }

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
            gap: 5,
            background: 'transparent',
            border: '1px solid var(--line)',
            borderRadius: 4,
            padding: '3px 8px',
            fontSize: 12,
            fontWeight: 600,
            color: 'var(--ink)',
            cursor: 'pointer',
            whiteSpace: 'nowrap',
            maxWidth: 160,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {activeTeamId === null ? 'Personal' : (active?.name ?? 'Teams')}
          </span>
          <span style={{ color: 'var(--muted)', fontSize: 10, flexShrink: 0 }}>▾</span>
        </button>
      </div>

      {open && createPortal(
        <div
          ref={menuRef}
          className="dropdown-menu"
          role="menu"
          style={{ position: 'fixed', top: pos.top, left: pos.left, minWidth: 200 }}
        >
          {sorted.map((team, idx) => (
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
                onClick={(e) => { e.stopPropagation(); void handleReorder(team.id, 'up'); }}
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
                disabled={idx === sorted.length - 1}
                onClick={(e) => { e.stopPropagation(); void handleReorder(team.id, 'down'); }}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: idx === sorted.length - 1 ? 'default' : 'pointer',
                  opacity: idx === sorted.length - 1 ? 0.25 : 0.6,
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
          <div className="dropdown-menu__separator" role="separator" />
          <button
            type="button"
            role="menuitem"
            className="dropdown-menu__item"
            style={activeTeamId === null ? { fontWeight: 600 } : undefined}
            onClick={() => { setOpen(false); setActiveTeamId(null); navigate('/hosts'); }}
          >
            Personal
          </button>
          <div className="dropdown-menu__separator" role="separator" />
          <button
            type="button"
            role="menuitem"
            className="dropdown-menu__item"
            onClick={() => { setOpen(false); setCreateOpen(true); }}
          >
            + Create team
          </button>
          <button
            type="button"
            role="menuitem"
            className="dropdown-menu__item"
            onClick={() => { setOpen(false); navigate('/teams'); }}
          >
            Manage teams
          </button>
        </div>,
        document.body,
      )}

      {/* Create team modal */}
      <Modal
        open={createOpen}
        onClose={() => { setCreateOpen(false); setCreateName(''); }}
        title="Create team"
        maxWidth={400}
        footer={
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <button
              type="button"
              className="btn btn--ghost"
              onClick={() => { setCreateOpen(false); setCreateName(''); }}
              disabled={creating}
            >
              Cancel
            </button>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => void handleCreateTeam()}
              disabled={creating || createName.trim() === ''}
            >
              {creating ? 'Creating…' : 'Create'}
            </button>
          </div>
        }
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <label style={{ fontSize: 12, color: 'var(--muted)' }}>
            Team name
          </label>
          <input
            className="input"
            type="text"
            placeholder="e.g. Acme Corp"
            value={createName}
            onChange={(e) => setCreateName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !creating && createName.trim()) {
                void handleCreateTeam();
              }
            }}
            autoFocus
            disabled={creating}
          />
        </div>
      </Modal>
    </>
  );
}
