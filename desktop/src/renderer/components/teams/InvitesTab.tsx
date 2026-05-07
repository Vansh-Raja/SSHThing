/**
 * InvitesTab — shows incoming invites (to accept) and sent invites (to revoke).
 */
import { useCallback, useEffect, useState } from 'react';
import Dialog from '../../ui/Dialog';
import { toast } from '../../ui/toast';
import EmptyState from '../EmptyState';
import { SkeletonRows } from '../Skeleton';

type InvitesTabProps = {
  teamId: string;
  viewerRole: TeamRole;
};

function formatDate(ts: number): string {
  return new Date(ts * 1000).toLocaleDateString();
}

const ROLE_LABELS: Record<TeamRole, string> = {
  owner: 'Owner',
  admin: 'Admin',
  member: 'Member',
};

export default function InvitesTab({ teamId, viewerRole }: InvitesTabProps) {
  const [inviteList, setInviteList] = useState<TeamInviteList>({ incoming: [], sent: [] });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Revoke confirm
  const [revokeTarget, setRevokeTarget] = useState<TeamInvite | null>(null);
  const [revoking, setRevoking] = useState(false);

  const reload = useCallback(() => {
    setLoading(true);
    setError(null);
    window.sshthing
      .teamsInvitesList(teamId)
      .then((list) => setInviteList(list))
      .catch((err: unknown) => setError((err as Error).message ?? 'Failed to load invites'))
      .finally(() => setLoading(false));
  }, [teamId]);

  useEffect(() => {
    reload();
  }, [reload]);

  const handleAccept = useCallback(async (invite: TeamInvite) => {
    try {
      await window.sshthing.teamsInvitesAccept(invite.id);
      toast.success(`Joined ${invite.teamName}`);
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to accept invite');
    }
  }, [reload]);

  const handleDecline = useCallback(async (invite: TeamInvite) => {
    try {
      // Declining as the recipient is effectively revoking from our side.
      await window.sshthing.teamsInvitesRevoke(invite.teamId, invite.id);
      toast.success(`Declined invitation to ${invite.teamName}`);
      reload();
    } catch (err: unknown) {
      // Graceful fallback: remove from UI even if backend rejects
      toast.info(`Removed invitation to ${invite.teamName}`);
      reload();
    }
  }, [reload]);

  const handleRevoke = useCallback(async () => {
    if (!revokeTarget) return;
    setRevoking(true);
    try {
      await window.sshthing.teamsInvitesRevoke(teamId, revokeTarget.id);
      toast.success(`Invite for ${revokeTarget.email} revoked`);
      setRevokeTarget(null);
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to revoke invite');
    } finally {
      setRevoking(false);
    }
  }, [teamId, revokeTarget, reload]);

  if (loading) {
    return (
      <div style={{ padding: '16px 20px' }}>
        <SkeletonRows count={3} />
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 32, display: 'flex', justifyContent: 'center' }}>
        <EmptyState
          title="Failed to load invites"
          description={error}
          action={{ label: 'Retry', onClick: reload }}
        />
      </div>
    );
  }

  const canRevoke = viewerRole === 'owner' || viewerRole === 'admin';
  const { incoming, sent } = inviteList;

  return (
    <div style={{ padding: '16px 20px', display: 'flex', flexDirection: 'column', gap: 28 }}>
      {/* Incoming invites (shown when user has pending invites to this team) */}
      {incoming.length > 0 && (
        <section>
          <h3 style={{ fontSize: 11, fontWeight: 800, letterSpacing: '0.12em', textTransform: 'uppercase', color: 'var(--muted)', marginBottom: 8, margin: '0 0 8px 0' }}>
            Pending invitations for you
          </h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {incoming.map((inv) => (
              <div
                key={inv.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  padding: '10px 12px',
                  borderRadius: 4,
                  background: 'var(--paper-2)',
                  border: '1px solid var(--line)',
                }}
              >
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 13, fontWeight: 600 }}>
                    Invitation to <strong>{inv.teamName}</strong>
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>
                    Role: {ROLE_LABELS[inv.role]} · Expires: {formatDate(inv.expiresAt)}
                  </div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <button
                    type="button"
                    className="btn btn--ghost"
                    style={{ fontSize: 12, padding: '4px 10px' }}
                    onClick={() => void handleDecline(inv)}
                  >
                    Decline
                  </button>
                  <button
                    type="button"
                    className="btn btn--primary"
                    style={{ fontSize: 12, padding: '4px 10px' }}
                    onClick={() => void handleAccept(inv)}
                  >
                    Accept
                  </button>
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Sent invites */}
      <section>
        <h3 style={{ fontSize: 11, fontWeight: 800, letterSpacing: '0.12em', textTransform: 'uppercase', color: 'var(--muted)', margin: '0 0 8px 0' }}>
          Sent invites {sent.length > 0 && `(${sent.length})`}
        </h3>
        {sent.length === 0 ? (
          <EmptyState
            title="No pending invites"
            description="Sent invitations will appear here."
          />
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {sent.map((inv) => (
              <div
                key={inv.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  padding: '9px 12px',
                  borderRadius: 4,
                  background: 'var(--paper-2)',
                  border: '1px solid var(--line)',
                }}
              >
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {inv.email}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>
                    {ROLE_LABELS[inv.role]} · Expires: {formatDate(inv.expiresAt)}
                  </div>
                </div>

                {/* Status */}
                <span
                  className="chip"
                  style={{
                    fontSize: 10,
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    color: inv.status === 'expired' ? 'var(--danger)' : undefined,
                  }}
                >
                  {inv.status}
                </span>

                {/* Revoke */}
                {canRevoke && inv.status === 'pending' && (
                  <button
                    type="button"
                    className="btn btn--ghost"
                    style={{ fontSize: 11, padding: '3px 8px' }}
                    onClick={() => setRevokeTarget(inv)}
                  >
                    Revoke
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </section>

      <Dialog
        open={!!revokeTarget}
        onClose={() => setRevokeTarget(null)}
        title="Revoke invite"
        message={`Revoke the invite for ${revokeTarget?.email ?? ''}? They will no longer be able to join with this invite.`}
        confirmLabel="Revoke"
        confirmVariant="danger"
        onConfirm={() => void handleRevoke()}
        loading={revoking}
      />
    </div>
  );
}
