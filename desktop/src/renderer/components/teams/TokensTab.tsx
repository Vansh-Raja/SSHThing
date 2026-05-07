/**
 * TokensTab — lists automation tokens and allows creating/revoking them.
 * Tokens are not team-scoped at the daemon level, but shown here for convenience.
 * Only owners and admins can manage tokens.
 */
import { useCallback, useEffect, useState } from 'react';
import { useTeamHosts } from '../../hooks/useTeamHosts';
import DropdownMenu, { type MenuItemDef } from '../../ui/DropdownMenu';
import Dialog from '../../ui/Dialog';
import Modal from '../../ui/Modal';
import { toast } from '../../ui/toast';

type TokensTabProps = {
  teamId: string;
  viewerRole: TeamRole;
};

function formatDate(ts: number | null | undefined): string {
  if (!ts) return '—';
  return new Date(ts * 1000).toLocaleDateString();
}

export default function TokensTab({ teamId, viewerRole }: TokensTabProps) {
  const [tokens, setTokens] = useState<TokenSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { hosts } = useTeamHosts(teamId);

  // Create form
  const [createOpen, setCreateOpen] = useState(false);
  const [tokenName, setTokenName] = useState('');
  const [selectedHostIds, setSelectedHostIds] = useState<string[]>([]);
  const [creating, setCreating] = useState(false);
  const [newRawToken, setNewRawToken] = useState<string | null>(null);

  // Revoke confirm
  const [revokeTarget, setRevokeTarget] = useState<TokenSummary | null>(null);
  const [revoking, setRevoking] = useState(false);

  const canManage = viewerRole === 'owner' || viewerRole === 'admin';

  const reload = useCallback(() => {
    setLoading(true);
    setError(null);
    window.sshthing
      .tokensList()
      .then((result) => setTokens(result.tokens ?? []))
      .catch((err: unknown) => setError((err as Error).message ?? 'Failed to load tokens'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  const handleCreate = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tokenName.trim()) {
      toast.error('Token name is required');
      return;
    }
    if (selectedHostIds.length === 0) {
      toast.error('Select at least one host');
      return;
    }
    setCreating(true);
    try {
      const result = await window.sshthing.tokensCreate(
        tokenName.trim(),
        selectedHostIds.map((hostId) => ({ hostId })),
      );
      setNewRawToken(result.rawToken);
      setCreateOpen(false);
      setTokenName('');
      setSelectedHostIds([]);
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to create token');
    } finally {
      setCreating(false);
    }
  }, [tokenName, selectedHostIds, reload]);

  const handleRevoke = useCallback(async () => {
    if (!revokeTarget) return;
    setRevoking(true);
    try {
      await window.sshthing.tokensRevoke(revokeTarget.id);
      toast.success(`Token "${revokeTarget.name}" revoked`);
      setRevokeTarget(null);
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to revoke token');
    } finally {
      setRevoking(false);
    }
  }, [revokeTarget, reload]);

  const handleDeleteRevoked = useCallback(async (token: TokenSummary) => {
    try {
      await window.sshthing.tokensDeleteRevoked(token.id);
      toast.success(`Token "${token.name}" deleted`);
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to delete token');
    }
  }, [reload]);

  if (loading) {
    return (
      <div style={{ padding: 32, textAlign: 'center', color: 'var(--muted)', fontSize: 13 }}>
        Loading tokens…
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

  const toggleHost = (hostId: string) => {
    setSelectedHostIds((prev) =>
      prev.includes(hostId) ? prev.filter((id) => id !== hostId) : [...prev, hostId],
    );
  };

  return (
    <div style={{ padding: '16px 20px' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <span style={{ fontSize: 12, color: 'var(--muted)' }}>
          {tokens.length} {tokens.length === 1 ? 'token' : 'tokens'}
        </span>
        {canManage && (
          <button
            type="button"
            className="btn btn--primary"
            style={{ fontSize: 12, padding: '5px 12px' }}
            onClick={() => { setTokenName(''); setSelectedHostIds([]); setCreateOpen(true); }}
          >
            Create token
          </button>
        )}
      </div>

      {tokens.length === 0 ? (
        <div style={{ padding: 32, textAlign: 'center', color: 'var(--muted)', fontSize: 13 }}>
          No tokens yet.{canManage ? ' Click "Create token" to get started.' : ''}
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {tokens.map((token) => {
            const isRevoked = token.status === 'revoked';
            const menuItems: MenuItemDef[] = [];
            if (canManage) {
              if (!isRevoked) {
                menuItems.push({ kind: 'item', label: 'Revoke', danger: true, onClick: () => setRevokeTarget(token) });
              } else {
                menuItems.push({ kind: 'item', label: 'Delete', danger: true, onClick: () => void handleDeleteRevoked(token) });
              }
            }

            return (
              <div
                key={token.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  padding: '9px 12px',
                  borderRadius: 4,
                  background: 'var(--paper-2)',
                  border: '1px solid var(--line)',
                  opacity: isRevoked ? 0.6 : 1,
                }}
              >
                <span style={{ fontSize: 14, color: 'var(--accent)', flexShrink: 0 }}>⬡</span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {token.name}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>
                    Created: {formatDate(token.createdAt)}
                    {token.lastUsedAt ? ` · Last used: ${formatDate(token.lastUsedAt)}` : ''}
                    {token.useCount ? ` · ${token.useCount} uses` : ''}
                  </div>
                </div>
                <span
                  className="chip"
                  style={{
                    fontSize: 10,
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    color: isRevoked ? 'var(--danger)' : undefined,
                  }}
                >
                  {token.status}
                </span>
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
                        title="Token actions"
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

      {/* Create token modal */}
      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Create token"
        maxWidth={480}
        footer={
          <div className="modal__actions">
            <button type="button" className="btn btn--ghost" onClick={() => setCreateOpen(false)} disabled={creating}>
              Cancel
            </button>
            <button
              type="button"
              className="btn btn--primary"
              disabled={creating}
              onClick={(e) => void handleCreate(e as unknown as React.FormEvent)}
            >
              {creating ? <span className="spinner" /> : 'Create'}
            </button>
          </div>
        }
      >
        <form onSubmit={(e) => void handleCreate(e)} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div className="field">
            <label className="field__label">Token name</label>
            <input
              className="field__input"
              type="text"
              placeholder="CI / Deploy bot"
              value={tokenName}
              onChange={(e) => setTokenName(e.target.value)}
              autoFocus
              required
            />
          </div>
          <div className="field">
            <label className="field__label">Grant access to hosts</label>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4, maxHeight: 200, overflowY: 'auto', border: '1px solid var(--line)', borderRadius: 4, padding: 6 }}>
              {hosts.length === 0 ? (
                <span style={{ fontSize: 12, color: 'var(--muted)', padding: 4 }}>No team hosts available</span>
              ) : (
                hosts.map((h) => (
                  <label key={h.id} style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, cursor: 'pointer', padding: '3px 4px', borderRadius: 3 }}>
                    <input
                      type="checkbox"
                      checked={selectedHostIds.includes(h.id)}
                      onChange={() => toggleHost(h.id)}
                    />
                    <span style={{ fontWeight: 500 }}>{h.label || h.hostname}</span>
                    <span style={{ color: 'var(--muted)' }}>{h.hostname}:{h.port}</span>
                  </label>
                ))
              )}
            </div>
          </div>
        </form>
      </Modal>

      {/* Raw token reveal modal (only shown immediately after creation) */}
      <Modal
        open={!!newRawToken}
        onClose={() => setNewRawToken(null)}
        title="Token created"
        dismissible={false}
        maxWidth={480}
        footer={
          <div className="modal__actions">
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => {
                if (newRawToken) void navigator.clipboard.writeText(newRawToken);
                toast.success('Copied to clipboard');
              }}
            >
              Copy token
            </button>
            <button type="button" className="btn btn--ghost" onClick={() => setNewRawToken(null)}>
              Done
            </button>
          </div>
        }
      >
        <p style={{ fontSize: 13, color: 'var(--muted)', marginTop: 0 }}>
          Copy this token now — it will not be shown again.
        </p>
        <div
          style={{
            background: 'var(--paper)',
            border: '1px solid var(--line)',
            borderRadius: 4,
            padding: '8px 10px',
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            wordBreak: 'break-all',
            userSelect: 'all',
          }}
        >
          {newRawToken}
        </div>
      </Modal>

      {/* Revoke confirm */}
      <Dialog
        open={!!revokeTarget}
        onClose={() => setRevokeTarget(null)}
        title="Revoke token"
        message={`Revoke "${revokeTarget?.name ?? ''}"? Any automation using this token will stop working immediately.`}
        confirmLabel="Revoke"
        confirmVariant="danger"
        onConfirm={() => void handleRevoke()}
        loading={revoking}
      />
    </div>
  );
}
