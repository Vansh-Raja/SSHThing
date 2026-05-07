/**
 * MembersTab — list team members and manage their roles.
 * Admins and owners can invite, update roles, and remove members.
 */
import { useCallback, useState } from 'react';
import { useTeamMembers } from '../../hooks/useTeamMembers';
import DropdownMenu, { type MenuItemDef } from '../../ui/DropdownMenu';
import Dialog from '../../ui/Dialog';
import Drawer from '../../ui/Drawer';
import { toast } from '../../ui/toast';
import EmptyState from '../EmptyState';
import { SkeletonRows } from '../Skeleton';

type MembersTabProps = {
  teamId: string;
  viewerRole: TeamRole;
};

const ROLE_LABELS: Record<TeamRole, string> = {
  owner: 'Owner',
  admin: 'Admin',
  member: 'Member',
};

function canManage(viewerRole: TeamRole): boolean {
  return viewerRole === 'owner' || viewerRole === 'admin';
}

export default function MembersTab({ teamId, viewerRole }: MembersTabProps) {
  const { members, loading, error, reload } = useTeamMembers(teamId);

  // Invite drawer
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState<TeamRole>('member');
  const [inviting, setInviting] = useState(false);

  // Remove confirm
  const [removeTarget, setRemoveTarget] = useState<TeamMember | null>(null);
  const [removing, setRemoving] = useState(false);

  const handleInvite = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteEmail.trim()) {
      toast.error('Email is required');
      return;
    }
    setInviting(true);
    try {
      await window.sshthing.teamsMembersInvite(teamId, inviteEmail.trim(), inviteRole);
      toast.success(`Invite sent to ${inviteEmail}`);
      setInviteOpen(false);
      setInviteEmail('');
      setInviteRole('member');
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to invite member');
    } finally {
      setInviting(false);
    }
  }, [teamId, inviteEmail, inviteRole, reload]);

  const handleUpdateRole = useCallback(async (member: TeamMember, role: TeamRole) => {
    try {
      await window.sshthing.teamsMembersUpdateRole(teamId, member.id, role);
      toast.success(`${member.displayName || member.email} is now ${ROLE_LABELS[role]}`);
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to update role');
    }
  }, [teamId, reload]);

  const handleRemove = useCallback(async () => {
    if (!removeTarget) return;
    setRemoving(true);
    try {
      await window.sshthing.teamsMembersRemove(teamId, removeTarget.id);
      toast.success(`${removeTarget.displayName || removeTarget.email} removed`);
      setRemoveTarget(null);
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to remove member');
    } finally {
      setRemoving(false);
    }
  }, [teamId, removeTarget, reload]);

  if (loading) {
    return (
      <div style={{ padding: '16px 20px' }}>
        <SkeletonRows count={4} />
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 32, display: 'flex', justifyContent: 'center' }}>
        <EmptyState
          title="Failed to load members"
          description={error}
          action={{ label: 'Retry', onClick: reload }}
        />
      </div>
    );
  }

  return (
    <div style={{ padding: '16px 20px' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <span style={{ fontSize: 12, color: 'var(--muted)' }}>
          {members.length} {members.length === 1 ? 'member' : 'members'}
        </span>
        {canManage(viewerRole) && (
          <button
            type="button"
            className="btn btn--primary"
            style={{ fontSize: 12, padding: '5px 12px' }}
            onClick={() => { setInviteEmail(''); setInviteRole('member'); setInviteOpen(true); }}
          >
            Invite member
          </button>
        )}
      </div>

      {/* Member list */}
      {members.length === 0 ? (
        <EmptyState
          title="No members yet"
          description="Invite team members using the button above."
        />
      ) : (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {members.map((m) => {
          const displayName = m.displayName || m.email;
          const menuItems: MenuItemDef[] = [];

          if (canManage(viewerRole) && m.role !== 'owner') {
            if (m.role !== 'admin' && viewerRole === 'owner') {
              menuItems.push({ kind: 'item', label: 'Promote to Admin', onClick: () => void handleUpdateRole(m, 'admin') });
            }
            if (m.role !== 'member') {
              menuItems.push({ kind: 'item', label: 'Set as Member', onClick: () => void handleUpdateRole(m, 'member') });
            }
            menuItems.push({ kind: 'separator' });
            menuItems.push({ kind: 'item', label: 'Remove from team', danger: true, onClick: () => setRemoveTarget(m) });
          }

          return (
            <div
              key={m.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '8px 10px',
                borderRadius: 4,
                background: 'var(--paper-2)',
                border: '1px solid var(--line)',
              }}
            >
              {/* Avatar placeholder */}
              <div
                style={{
                  width: 32,
                  height: 32,
                  borderRadius: '50%',
                  background: 'var(--accent)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 13,
                  fontWeight: 700,
                  color: '#fff',
                  flexShrink: 0,
                }}
              >
                {(displayName[0] ?? '?').toUpperCase()}
              </div>

              {/* Info */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {displayName}
                </div>
                {m.displayName && (
                  <div style={{ fontSize: 11, color: 'var(--muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {m.email}
                  </div>
                )}
              </div>

              {/* Role badge */}
              <span
                className="chip"
                style={{ fontSize: 10, letterSpacing: '0.06em', textTransform: 'uppercase' }}
              >
                {ROLE_LABELS[m.role]}
              </span>

              {/* Status */}
              {m.status !== 'active' && (
                <span style={{ fontSize: 11, color: 'var(--muted)', textTransform: 'capitalize' }}>
                  {m.status}
                </span>
              )}

              {/* Actions menu */}
              {menuItems.length > 0 && (
                <DropdownMenu
                  trigger={
                    <button
                      type="button"
                      style={{
                        background: 'transparent',
                        border: 'none',
                        color: 'var(--muted)',
                        fontSize: 16,
                        cursor: 'pointer',
                        padding: '0 4px',
                        lineHeight: 1,
                      }}
                      title="Member actions"
                    >
                      ⋯
                    </button>
                  }
                  items={menuItems}
                />
              )}
            </div>
          );
        })}
      </div>
      )}

      {/* Invite drawer */}
      <Drawer
        open={inviteOpen}
        onClose={() => setInviteOpen(false)}
        title="Invite member"
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="btn btn--ghost" onClick={() => setInviteOpen(false)} disabled={inviting}>
              Cancel
            </button>
            <button
              type="button"
              className="btn btn--primary"
              disabled={inviting}
              onClick={(e) => void handleInvite(e as unknown as React.FormEvent)}
            >
              {inviting ? <span className="spinner" /> : 'Send invite'}
            </button>
          </div>
        }
        width={400}
      >
        <form onSubmit={(e) => void handleInvite(e)} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div className="field">
            <label className="field__label">Email address</label>
            <input
              className="field__input"
              type="email"
              placeholder="colleague@example.com"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              required
              autoFocus
            />
          </div>
          <div className="field">
            <label className="field__label">Role</label>
            <select
              className="field__input"
              value={inviteRole}
              onChange={(e) => setInviteRole(e.target.value as TeamRole)}
            >
              <option value="member">Member</option>
              {viewerRole === 'owner' && <option value="admin">Admin</option>}
            </select>
          </div>
        </form>
      </Drawer>

      {/* Remove confirm dialog */}
      <Dialog
        open={!!removeTarget}
        onClose={() => setRemoveTarget(null)}
        title="Remove member"
        message={`Remove ${(removeTarget?.displayName || removeTarget?.email) ?? ''} from this team? They will lose access to all team hosts.`}
        confirmLabel="Remove"
        confirmVariant="danger"
        onConfirm={() => void handleRemove()}
        loading={removing}
      />
    </div>
  );
}
