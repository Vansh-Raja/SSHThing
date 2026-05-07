/**
 * TeamHostsTab — list, add, edit, delete team hosts.
 * Wave 2B additions:
 * - Reveal shared credential (canRevealSecrets gate)
 * - Per-member credential roster (per_member hosts)
 * - Import personal host → team
 * - Connect-config error surfaces
 */
import { useCallback, useState } from 'react';
import { useTeamHosts } from '../../hooks/useTeamHosts';
import TeamHostDrawer from './TeamHostDrawer';
import DropdownMenu, { type MenuItemDef } from '../../ui/DropdownMenu';
import Dialog from '../../ui/Dialog';
import Modal from '../../ui/Modal';
import { toast } from '../../ui/toast';
import RevealTeamCredentialModal from './RevealTeamCredentialModal';
import PerMemberCredentialRoster from './PerMemberCredentialRoster';
import ImportPersonalHostModal from './ImportPersonalHostModal';

type TeamHostsTabProps = {
  teamId: string;
  viewerRole: TeamRole;
};

function formatLastConnected(ts: number | null | undefined): string {
  if (!ts) return 'Never';
  return new Date(ts * 1000).toLocaleDateString();
}

type CredErrorModalState = {
  kind: 'personal_credential_not_configured' | 'shared_credential_not_configured';
  hostLabel: string;
  hostId: string;
  isPerMember: boolean;
} | null;

export default function TeamHostsTab({ teamId, viewerRole }: TeamHostsTabProps) {
  const { hosts, loading, error, reload } = useTeamHosts(teamId);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingHost, setEditingHost] = useState<TeamHost | null>(null);

  const [deleteTarget, setDeleteTarget] = useState<TeamHost | null>(null);
  const [deleting, setDeleting] = useState(false);

  // Reveal shared credential state
  const [revealTarget, setRevealTarget] = useState<TeamHost | null>(null);
  const [revealedCred, setRevealedCred] = useState<RevealedTeamHostCredential | null>(null);
  const [revealing, setRevealing] = useState(false);

  // Roster panel state (for per_member hosts)
  const [rosterHost, setRosterHost] = useState<TeamHost | null>(null);

  // Import personal host state
  const [importOpen, setImportOpen] = useState(false);

  // Connect-config error modal
  const [credErrorModal, setCredErrorModal] = useState<CredErrorModalState>(null);

  const canAdd = viewerRole === 'owner' || viewerRole === 'admin';

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await window.sshthing.teamsHostsDelete(deleteTarget.id);
      toast.success(`${deleteTarget.label || deleteTarget.hostname} deleted`);
      setDeleteTarget(null);
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to delete host');
    } finally {
      setDeleting(false);
    }
  }, [deleteTarget, reload]);

  const handleRevealShared = useCallback(async (host: TeamHost) => {
    setRevealTarget(host);
    setRevealing(true);
    try {
      const cred = await window.sshthing.teamsHostsRevealShared(host.id);
      setRevealedCred(cred);
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to reveal credential');
      setRevealTarget(null);
    } finally {
      setRevealing(false);
    }
  }, []);

  if (loading) {
    return (
      <div style={{ padding: 32, textAlign: 'center', color: 'var(--muted)', fontSize: 13 }}>
        Loading hosts…
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 32, textAlign: 'center', color: 'var(--danger)', fontSize: 13 }}>
        {error}
      </div>
    );
  }

  return (
    <div style={{ padding: '16px 20px' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, gap: 8 }}>
        <span style={{ fontSize: 12, color: 'var(--muted)' }}>
          {hosts.length} {hosts.length === 1 ? 'host' : 'hosts'}
        </span>
        <div style={{ display: 'flex', gap: 6 }}>
          {canAdd && (
            <button
              type="button"
              className="btn btn--ghost"
              style={{ fontSize: 12, padding: '5px 12px' }}
              onClick={() => setImportOpen(true)}
            >
              Import from vault
            </button>
          )}
          {canAdd && (
            <button
              type="button"
              className="btn btn--primary"
              style={{ fontSize: 12, padding: '5px 12px' }}
              onClick={() => { setEditingHost(null); setDrawerOpen(true); }}
            >
              Add host
            </button>
          )}
        </div>
      </div>

      {hosts.length === 0 ? (
        <div style={{ padding: 32, textAlign: 'center', color: 'var(--muted)', fontSize: 13 }}>
          No hosts yet.{canAdd ? ' Click "Add host" to get started.' : ''}
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {hosts.map((host) => {
            const displayLabel = host.label || host.hostname;
            const isPerMember = host.credentialMode === 'per_member';
            const isShared = host.credentialMode === 'shared';
            const menuItems: MenuItemDef[] = [];

            if (host.canRevealSecrets && isShared && host.credentialType !== 'none') {
              menuItems.push({
                kind: 'item',
                label: revealing && revealTarget?.id === host.id ? 'Revealing…' : 'Reveal credential',
                onClick: () => void handleRevealShared(host),
              });
            }

            if (isPerMember) {
              menuItems.push({
                kind: 'item',
                label: rosterHost?.id === host.id ? 'Hide roster' : 'View credential roster',
                onClick: () => setRosterHost(rosterHost?.id === host.id ? null : host),
              });
            }

            if (menuItems.length > 0 && host.canManageHosts) {
              menuItems.push({ kind: 'separator' });
            }

            if (host.canManageHosts) {
              menuItems.push({
                kind: 'item',
                label: 'Edit',
                onClick: () => { setEditingHost(host); setDrawerOpen(true); },
              });
              menuItems.push({ kind: 'separator' });
              menuItems.push({
                kind: 'item',
                label: 'Delete',
                danger: true,
                onClick: () => setDeleteTarget(host),
              });
            }

            return (
              <div key={host.id}>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 10,
                    padding: '9px 12px',
                    borderRadius: rosterHost?.id === host.id ? '4px 4px 0 0' : 4,
                    background: 'var(--paper-2)',
                    border: '1px solid var(--line)',
                    borderBottom: rosterHost?.id === host.id ? 'none' : undefined,
                  }}
                >
                  {/* Icon */}
                  <span style={{ fontSize: 14, color: 'var(--accent)', flexShrink: 0 }}>⬡</span>

                  {/* Info */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 13, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {displayLabel}
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {host.username}@{host.hostname}:{host.port}
                    </div>
                  </div>

                  {/* Credential mode badge */}
                  {host.credentialMode && (
                    <span className="chip" style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.06em' }}>
                      {host.credentialMode}
                    </span>
                  )}

                  {/* Group */}
                  {host.group && (
                    <span style={{ fontSize: 11, color: 'var(--muted)' }}>{host.group}</span>
                  )}

                  {/* Last connected */}
                  <span style={{ fontSize: 11, color: 'var(--muted)', flexShrink: 0 }}>
                    {formatLastConnected(host.lastConnectedAt)}
                  </span>

                  {/* Actions */}
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
                          title="Host actions"
                        >
                          ⋯
                        </button>
                      }
                      items={menuItems}
                    />
                  )}
                </div>

                {/* Inline roster for per_member hosts */}
                {rosterHost?.id === host.id && (
                  <div
                    style={{
                      padding: '12px',
                      background: 'var(--paper-2)',
                      border: '1px solid var(--line)',
                      borderTop: '1px solid color-mix(in srgb, var(--line) 50%, transparent)',
                      borderRadius: '0 0 4px 4px',
                    }}
                  >
                    <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--muted)', marginBottom: 6, textTransform: 'uppercase', letterSpacing: '0.06em' }}>
                      Credential roster
                    </div>
                    <PerMemberCredentialRoster
                      hostId={host.id}
                      hostLabel={displayLabel}
                      canManage={!!host.canManageHosts}
                      credentialType={host.credentialType ?? 'password'}
                    />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      <TeamHostDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        teamId={teamId}
        host={editingHost}
        onSaved={reload}
      />

      <Dialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        title="Delete host"
        message={`Delete "${(deleteTarget?.label || deleteTarget?.hostname) ?? ''}"? This cannot be undone.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        onConfirm={() => void handleDelete()}
        loading={deleting}
      />

      {/* Reveal shared credential modal */}
      <RevealTeamCredentialModal
        open={!!revealedCred}
        onClose={() => { setRevealedCred(null); setRevealTarget(null); }}
        hostLabel={revealTarget?.label || revealTarget?.hostname || ''}
        credentialScope="shared"
        credentialType={revealedCred?.credentialType ?? ''}
        secret={revealedCred?.secret ?? ''}
        username={revealedCred?.username}
      />

      {/* Import personal host modal */}
      <ImportPersonalHostModal
        open={importOpen}
        onClose={() => setImportOpen(false)}
        teamId={teamId}
        onImported={reload}
      />

      {/* Connect-config error modal */}
      <Modal
        open={!!credErrorModal}
        onClose={() => setCredErrorModal(null)}
        title="Credential not configured"
        maxWidth={440}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="btn btn--ghost" onClick={() => setCredErrorModal(null)}>
              Close
            </button>
          </div>
        }
      >
        {credErrorModal && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10, fontSize: 13 }}>
            {credErrorModal.kind === 'personal_credential_not_configured' ? (
              <>
                <p style={{ margin: 0, lineHeight: 1.55 }}>
                  You haven&apos;t set a personal credential for{' '}
                  <strong>{credErrorModal.hostLabel}</strong>.
                </p>
                <p style={{ margin: 0, lineHeight: 1.55, color: 'var(--muted)', fontSize: 12 }}>
                  Open the credential roster (⋯ → View credential roster) and click{' '}
                  <strong>Set credential</strong> on your row to add your credential.
                </p>
              </>
            ) : (
              <p style={{ margin: 0, lineHeight: 1.55 }}>
                The shared credential for <strong>{credErrorModal.hostLabel}</strong> has
                not been configured. An admin needs to edit the host and provide the shared
                credential.
              </p>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
