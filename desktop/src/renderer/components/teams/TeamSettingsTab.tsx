/**
 * TeamSettingsTab — team-level settings: rename, delete, leave.
 *
 * Actions by role:
 *   owner  → rename + delete team (requires typing team name to confirm)
 *   admin  → leave team
 *   member → leave team
 *
 * Transfer ownership is stubbed "Coming soon" — no web API route exists.
 */
import { useCallback, useState } from 'react';
import Modal from '../../ui/Modal';
import { toast } from '../../ui/toast';
import { useTeamContext } from '../../contexts/TeamContext';

type TeamSettingsTabProps = {
  team: TeamSummary;
  viewerRole: TeamRole;
  /** Called after a successful rename/delete/leave so the parent can reload. */
  onRefresh?: () => void;
};

export default function TeamSettingsTab({ team, viewerRole, onRefresh }: TeamSettingsTabProps) {
  const { setActiveTeamId } = useTeamContext();
  const isOwner = viewerRole === 'owner';

  // ── Rename ────────────────────────────────────────────────────────────────
  const [renameName, setRenameName] = useState(team.name);
  const [renaming, setRenaming] = useState(false);

  const handleRename = useCallback(async () => {
    const name = renameName.trim();
    if (!name || name === team.name) return;
    setRenaming(true);
    try {
      await window.sshthing.teamsRename(team.id, name);
      toast.success('Team renamed');
      onRefresh?.();
    } catch (err) {
      const e = err as Error;
      toast.error(e.message ?? 'Failed to rename team');
    } finally {
      setRenaming(false);
    }
  }, [renameName, team.id, team.name, onRefresh]);

  // ── Delete ────────────────────────────────────────────────────────────────
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState('');
  const [deleting, setDeleting] = useState(false);

  const handleDelete = useCallback(async () => {
    if (deleteConfirm !== team.name) return;
    setDeleting(true);
    try {
      await window.sshthing.teamsDelete(team.id);
      toast.success(`Team "${team.name}" deleted`);
      setDeleteOpen(false);
      setActiveTeamId(null);
      onRefresh?.();
    } catch (err) {
      const e = err as Error;
      toast.error(e.message ?? 'Failed to delete team');
    } finally {
      setDeleting(false);
    }
  }, [deleteConfirm, team.id, team.name, onRefresh]);

  // ── Leave ─────────────────────────────────────────────────────────────────
  const [leaveOpen, setLeaveOpen] = useState(false);
  const [leaving, setLeaving] = useState(false);

  const handleLeave = useCallback(async () => {
    setLeaving(true);
    try {
      await window.sshthing.teamsLeave(team.id);
      toast.success(`Left team "${team.name}"`);
      setLeaveOpen(false);
      setActiveTeamId(null);
      onRefresh?.();
    } catch (err) {
      const e = err as Error;
      toast.error(e.message ?? 'Failed to leave team');
    } finally {
      setLeaving(false);
    }
  }, [team.id, team.name, onRefresh]);

  return (
    <div style={{ padding: '24px 20px', maxWidth: 480 }}>
      {/* ── Team info ── */}
      <section style={{ marginBottom: 28 }}>
        <h3 style={{ fontSize: 11, fontWeight: 800, letterSpacing: '0.12em', textTransform: 'uppercase', color: 'var(--muted)', margin: '0 0 12px 0' }}>
          Team info
        </h3>
        <dl style={{ display: 'grid', gridTemplateColumns: 'max-content 1fr', gap: '6px 16px', margin: 0 }}>
          <dt style={{ fontSize: 12, color: 'var(--muted)' }}>Name</dt>
          <dd style={{ fontSize: 13, margin: 0, fontWeight: 500 }}>{team.name}</dd>
          <dt style={{ fontSize: 12, color: 'var(--muted)' }}>Slug</dt>
          <dd style={{ fontSize: 13, margin: 0, fontFamily: 'var(--font-mono)' }}>{team.slug}</dd>
          <dt style={{ fontSize: 12, color: 'var(--muted)' }}>Your role</dt>
          <dd style={{ fontSize: 13, margin: 0 }}>
            <span className="chip" style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.06em' }}>
              {viewerRole}
            </span>
          </dd>
        </dl>
      </section>

      {/* ── Rename (owner only) ── */}
      {isOwner && (
        <section style={{ marginBottom: 28 }}>
          <h3 style={{ fontSize: 11, fontWeight: 800, letterSpacing: '0.12em', textTransform: 'uppercase', color: 'var(--muted)', margin: '0 0 12px 0' }}>
            Rename team
          </h3>
          <div style={{ display: 'flex', gap: 8 }}>
            <input
              className="input"
              type="text"
              value={renameName}
              onChange={(e) => setRenameName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !renaming && renameName.trim() && renameName.trim() !== team.name) {
                  void handleRename();
                }
              }}
              disabled={renaming}
              style={{ flex: 1 }}
            />
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => void handleRename()}
              disabled={renaming || !renameName.trim() || renameName.trim() === team.name}
            >
              {renaming ? 'Saving…' : 'Save'}
            </button>
          </div>
        </section>
      )}

      {/* ── Danger zone ── */}
      <section>
        <h3 style={{ fontSize: 11, fontWeight: 800, letterSpacing: '0.12em', textTransform: 'uppercase', color: 'var(--danger)', margin: '0 0 12px 0' }}>
          Danger zone
        </h3>
        <div style={{ border: '1px solid var(--line)', borderRadius: 6, overflow: 'hidden' }}>
          {/* Transfer ownership — stub (no API route exists) */}
          {isOwner && (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 14px', borderBottom: '1px solid var(--line)' }}>
              <div>
                <div style={{ fontSize: 13, fontWeight: 500 }}>Transfer ownership</div>
                <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>Transfer ownership to another member</div>
              </div>
              <button type="button" className="btn btn--ghost" disabled style={{ fontSize: 12, opacity: 0.5 }}>
                Coming soon
              </button>
            </div>
          )}

          {/* Delete (owner) / Leave (others) */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 14px' }}>
            <div>
              <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--danger)' }}>
                {isOwner ? 'Delete team' : 'Leave team'}
              </div>
              <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>
                {isOwner
                  ? 'Permanently delete this team and all its hosts'
                  : 'Remove yourself from this team'}
              </div>
            </div>
            <button
              type="button"
              className="btn btn--danger"
              onClick={() => isOwner ? setDeleteOpen(true) : setLeaveOpen(true)}
            >
              {isOwner ? 'Delete…' : 'Leave…'}
            </button>
          </div>
        </div>
      </section>

      {/* ── Delete confirm modal ── */}
      <Modal
        open={deleteOpen}
        onClose={() => { setDeleteOpen(false); setDeleteConfirm(''); }}
        title="Delete team"
        maxWidth={420}
        footer={
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <button type="button" className="btn btn--ghost" onClick={() => { setDeleteOpen(false); setDeleteConfirm(''); }} disabled={deleting}>
              Cancel
            </button>
            <button type="button" className="btn btn--danger" onClick={() => void handleDelete()} disabled={deleting || deleteConfirm !== team.name}>
              {deleting ? 'Deleting…' : 'Delete team'}
            </button>
          </div>
        }
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <p style={{ margin: 0, fontSize: 13, color: 'var(--ink)' }}>
            This will permanently delete <strong>{team.name}</strong> and all its hosts. This action cannot be undone.
          </p>
          <label style={{ fontSize: 12, color: 'var(--muted)' }}>
            Type <strong>{team.name}</strong> to confirm
          </label>
          <input
            className="input"
            type="text"
            placeholder={team.name}
            value={deleteConfirm}
            onChange={(e) => setDeleteConfirm(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !deleting && deleteConfirm === team.name) {
                void handleDelete();
              }
            }}
            autoFocus
            disabled={deleting}
          />
        </div>
      </Modal>

      {/* ── Leave confirm modal ── */}
      <Modal
        open={leaveOpen}
        onClose={() => setLeaveOpen(false)}
        title="Leave team"
        maxWidth={380}
        footer={
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <button type="button" className="btn btn--ghost" onClick={() => setLeaveOpen(false)} disabled={leaving}>
              Cancel
            </button>
            <button type="button" className="btn btn--danger" onClick={() => void handleLeave()} disabled={leaving}>
              {leaving ? 'Leaving…' : 'Leave team'}
            </button>
          </div>
        }
      >
        <p style={{ margin: 0, fontSize: 13, color: 'var(--ink)' }}>
          You will lose access to <strong>{team.name}</strong> and all its hosts. You can only rejoin if re-invited.
        </p>
      </Modal>
    </div>
  );
}
