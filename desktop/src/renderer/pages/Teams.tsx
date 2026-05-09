/**
 * Teams — TUI-parity team host list.
 *
 * Layout: top bar with team switcher + pane-shell (sidebar list + detail pane).
 * Members/Invites/Audit/Settings accessible via dropdown, not primary tabs.
 */
import { useCallback, useState } from 'react';
import { useTeams } from '../hooks/useTeams';
import TeamSwitcher from '../components/teams/TeamSwitcher';
import TeamHostsTab from '../components/teams/TeamHostsTab';
import MembersTab from '../components/teams/MembersTab';
import InvitesTab from '../components/teams/InvitesTab';
import AuditTab from '../components/teams/AuditTab';
import TeamSettingsTab from '../components/teams/TeamSettingsTab';
import EmptyState from '../components/EmptyState';
import { SkeletonRows } from '../components/Skeleton';
import { useNotifications } from '../hooks/useNotifications';
import { useTeamContext } from '../contexts/TeamContext';
import { toast } from '../ui/toast';
import Modal from '../ui/Modal';
import DropdownMenu, { type MenuItemDef } from '../ui/DropdownMenu';

type SubView = 'hosts' | 'members' | 'invites' | 'audit' | 'settings';

export default function Teams() {
  const { teams, loading, notSignedIn, error, reload } = useTeams();
  const { activeTeamId, setActiveTeamId } = useTeamContext();
  const [subView, setSubView] = useState<SubView>('hosts');

  // Create team modal state
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState('');
  const [creating, setCreating] = useState(false);

  const activeTeam = teams.find((t) => t.id === activeTeamId) ?? teams[0] ?? null;
  const effectiveTeamId = activeTeam?.id ?? null;
  const viewerRole: TeamRole = activeTeam?.role ?? 'member';

  const handleSelectTeam = useCallback((team: TeamSummary) => {
    setActiveTeamId(team.id);
    setSubView('hosts');
  }, [setActiveTeamId]);

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
      reload();
    } catch (err) {
      const e = err as Error;
      toast.error(e.message ?? 'Failed to create team');
    } finally {
      setCreating(false);
    }
  }, [createName, setActiveTeamId, reload]);

  const handleNotification = useCallback((method: string) => {
    if (method === 'vault.locked') {
      window.location.hash = '/unlock';
    }
  }, []);
  useNotifications(handleNotification);

  // ── Loading state ──
  if (loading) {
    return (
      <div style={{ padding: '24px 20px' }}>
        <SkeletonRows count={3} />
      </div>
    );
  }

  // ── Not signed in ──
  if (notSignedIn) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
        <EmptyState
          icon={
            <svg width={28} height={28} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.6} strokeLinecap="round" strokeLinejoin="round" style={{ color: 'var(--muted)' }} aria-hidden="true">
              <path d="M9 4H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h4" />
              <path d="M16 17l5-5-5-5" />
              <path d="M21 12H9" />
            </svg>
          }
          title="Sign in to use Teams"
          description="Team features require a signed-in account."
          action={{ label: 'Sign in', onClick: () => { window.location.hash = '/sign-in'; } }}
        />
      </div>
    );
  }

  // ── Error state ──
  if (error) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
        <EmptyState
          title="Failed to load teams"
          description={error}
          action={{ label: 'Retry', onClick: reload }}
        />
      </div>
    );
  }

  // ── No teams ──
  if (teams.length === 0) {
    return (
      <>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
          <EmptyState
            title="No teams yet"
            description="Ask a team owner to invite you, or create your own team."
            action={{ label: '+ Create team', onClick: () => setCreateOpen(true) }}
          />
        </div>
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
            <label style={{ fontSize: 12, color: 'var(--muted)' }}>Team name</label>
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

  const moreMenuItems: MenuItemDef[] = [
    { kind: 'item', label: 'Members', onClick: () => setSubView('members') },
    { kind: 'item', label: 'Invites', onClick: () => setSubView('invites') },
    { kind: 'item', label: 'Audit', onClick: () => setSubView('audit') },
    { kind: 'separator' },
    { kind: 'item', label: 'Team settings', onClick: () => setSubView('settings') },
  ];

  return (
    <>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        {/* Top bar */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '8px 16px',
            borderBottom: '1.5px solid var(--line)',
            background: 'var(--paper-2)',
            flexShrink: 0,
          }}
        >
          <span
            style={{
              fontSize: 11,
              fontWeight: 800,
              letterSpacing: '0.14em',
              textTransform: 'uppercase',
              color: 'var(--muted)',
            }}
          >
            Teams
          </span>
          <TeamSwitcher
            teams={teams}
            activeTeamId={effectiveTeamId}
            onSelect={handleSelectTeam}
            onReorder={reload}
          />
          <div style={{ flex: 1 }} />
          {subView !== 'hosts' && (
            <button
              type="button"
              className="btn btn--ghost"
              style={{ fontSize: 11, padding: '3px 8px' }}
              onClick={() => setSubView('hosts')}
            >
              ← Back to hosts
            </button>
          )}
          <DropdownMenu
            trigger={
              <button
                type="button"
                className="btn btn--ghost"
                style={{ fontSize: 14, padding: '3px 8px', lineHeight: 1 }}
                title="More"
              >
                ⋯
              </button>
            }
            items={moreMenuItems}
          />
          <button
            type="button"
            className="btn btn--ghost"
            style={{ fontSize: 11, padding: '3px 8px' }}
            onClick={() => setCreateOpen(true)}
          >
            + New team
          </button>
        </div>

        {/* Main content */}
        <div style={{ flex: 1, overflow: 'hidden' }}>
          {effectiveTeamId && activeTeam ? (
            <>
              {subView === 'hosts' && (
                <TeamHostsTab teamId={effectiveTeamId} viewerRole={viewerRole} />
              )}
              {subView === 'members' && (
                <MembersTab teamId={effectiveTeamId} viewerRole={viewerRole} />
              )}
              {subView === 'invites' && (
                <InvitesTab teamId={effectiveTeamId} viewerRole={viewerRole} />
              )}
              {subView === 'audit' && (
                <AuditTab teamId={effectiveTeamId} />
              )}
              {subView === 'settings' && (
                <TeamSettingsTab team={activeTeam} viewerRole={viewerRole} onRefresh={reload} />
              )}
            </>
          ) : (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', flex: 1, color: 'var(--muted)', fontSize: 13 }}>
              Select a team above to get started.
            </div>
          )}
        </div>
      </div>

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
          <label style={{ fontSize: 12, color: 'var(--muted)' }}>Team name</label>
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
