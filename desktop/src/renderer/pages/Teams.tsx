/**
 * Teams — Phase 5 teams UI.
 *
 * Layout:
 * - Top bar with team switcher
 * - Tab strip: Hosts | Members | Invites | Tokens | Audit | Settings
 * - Tab content pane
 *
 * Sign-in guard: if the daemon returns not_signed_in error, shows a prompt.
 */
import { useCallback, useState } from 'react';
import { useTeams } from '../hooks/useTeams';
import TeamSwitcher from '../components/teams/TeamSwitcher';
import TeamHostsTab from '../components/teams/TeamHostsTab';
import MembersTab from '../components/teams/MembersTab';
import InvitesTab from '../components/teams/InvitesTab';
import TokensTab from '../components/teams/TokensTab';
import AuditTab from '../components/teams/AuditTab';
import TeamSettingsTab from '../components/teams/TeamSettingsTab';
import EmptyState from '../components/EmptyState';
import { SkeletonRows } from '../components/Skeleton';
import { useNotifications } from '../hooks/useNotifications';
import { useTeamContext } from '../contexts/TeamContext';
import { toast } from '../ui/toast';
import Modal from '../ui/Modal';

type TabId = 'hosts' | 'members' | 'invites' | 'tokens' | 'audit' | 'settings';

const TAB_ITEMS: { id: TabId; label: string }[] = [
  { id: 'hosts', label: 'Hosts' },
  { id: 'members', label: 'Members' },
  { id: 'invites', label: 'Invites' },
  { id: 'tokens', label: 'Tokens' },
  { id: 'audit', label: 'Audit' },
  { id: 'settings', label: 'Settings' },
];

export default function Teams() {
  const { teams, loading, notSignedIn, error, reload } = useTeams();
  const { activeTeamId, setActiveTeamId } = useTeamContext();
  const [activeTab, setActiveTab] = useState<TabId>('hosts');

  // Create team modal state (for empty-state button)
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState('');
  const [creating, setCreating] = useState(false);

  // After loading, auto-select the first team if none selected.
  const activeTeam = teams.find((t) => t.id === activeTeamId) ?? teams[0] ?? null;
  const effectiveTeamId = activeTeam?.id ?? null;
  const viewerRole: TeamRole = activeTeam?.role ?? 'member';

  const handleSelectTeam = useCallback((team: TeamSummary) => {
    setActiveTeamId(team.id);
    setActiveTab('hosts');
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

  // Reload teams if vault lock notification fires
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
        <button
          type="button"
          className="btn btn--ghost"
          style={{ fontSize: 11, padding: '3px 8px' }}
          onClick={() => setCreateOpen(true)}
        >
          + New team
        </button>
      </div>

      {effectiveTeamId && activeTeam ? (
        <div style={{ display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' }}>
          {/* Tab strip — sticky so it stays visible when content scrolls */}
          <div
            role="tablist"
            aria-label="Team tabs"
            style={{
              display: 'flex',
              alignItems: 'center',
              background: 'var(--paper-2)',
              borderBottom: '1.5px solid var(--line)',
              flexShrink: 0,
              overflowX: 'auto',
              scrollbarWidth: 'none',
              position: 'sticky',
              top: 0,
              zIndex: 10,
            }}
          >
            {TAB_ITEMS.map((tab) => (
              <button
                key={tab.id}
                type="button"
                role="tab"
                aria-selected={activeTab === tab.id}
                onClick={() => setActiveTab(tab.id)}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  padding: '8px 14px',
                  border: 'none',
                  borderBottom: `2px solid ${activeTab === tab.id ? 'var(--accent)' : 'transparent'}`,
                  background: 'transparent',
                  color: activeTab === tab.id ? 'var(--ink)' : 'var(--muted)',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 12,
                  fontWeight: 700,
                  letterSpacing: '0.04em',
                  cursor: 'pointer',
                  whiteSpace: 'nowrap',
                  flexShrink: 0,
                  outline: 'none',
                }}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab content */}
          <div style={{ flex: 1, overflow: 'auto' }}>
            {activeTab === 'hosts' && (
              <TeamHostsTab teamId={effectiveTeamId} viewerRole={viewerRole} />
            )}
            {activeTab === 'members' && (
              <MembersTab teamId={effectiveTeamId} viewerRole={viewerRole} />
            )}
            {activeTab === 'invites' && (
              <InvitesTab teamId={effectiveTeamId} viewerRole={viewerRole} />
            )}
            {activeTab === 'tokens' && (
              <TokensTab teamId={effectiveTeamId} viewerRole={viewerRole} />
            )}
            {activeTab === 'audit' && (
              <AuditTab teamId={effectiveTeamId} />
            )}
            {activeTab === 'settings' && (
              <TeamSettingsTab team={activeTeam} viewerRole={viewerRole} onRefresh={reload} />
            )}
          </div>
        </div>
      ) : (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', flex: 1, color: 'var(--muted)', fontSize: 13 }}>
          Select a team above to get started.
        </div>
      )}
    </div>

    {/* Create team modal (used from "+ New team" button in header) */}
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
