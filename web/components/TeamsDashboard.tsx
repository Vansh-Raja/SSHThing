"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";

import HostDrawer from "./teams/HostDrawer";
import InviteDrawer from "./teams/InviteDrawer";
import TabAudit from "./teams/TabAudit";
import TabHosts from "./teams/TabHosts";
import TabInvites from "./teams/TabInvites";
import TabMembers from "./teams/TabMembers";
import TabTokens from "./teams/TabTokens";
import TeamBar from "./teams/TeamBar";
import { apiRequest, errorMessage } from "./teams/api";
import type {
  DashboardTab,
  InviteResponse,
  TeamAuditEvent,
  TeamAutomationToken,
  TeamHost,
  TeamMember,
  TeamRole,
  TeamSummary,
} from "./teams/types";
import { confirmDialog, promptDialog } from "./ui/dialogs";
import { toast } from "./ui/toast";

const ALL_TABS: DashboardTab[] = ["members", "hosts", "invites", "tokens", "audit"];
const BASE_TABS: DashboardTab[] = ["members", "hosts", "invites"];

async function copyText(value: string, successMessage: string) {
  try {
    await navigator.clipboard.writeText(value);
    toast.success(successMessage);
  } catch {
    toast.error("Couldn't copy to clipboard.");
  }
}

export default function TeamsDashboard() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [teams, setTeams] = useState<TeamSummary[]>([]);
  const [selectedTeamId, setSelectedTeamId] = useState("");
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [hosts, setHosts] = useState<TeamHost[]>([]);
  const [invites, setInvites] = useState<InviteResponse>({
    incoming: [],
    sent: [],
  });
  const [auditEvents, setAuditEvents] = useState<TeamAuditEvent[]>([]);
  const [teamTokens, setTeamTokens] = useState<TeamAutomationToken[]>([]);
  const [loading, setLoading] = useState(true);

  // Drawer state.
  const [hostDrawerOpen, setHostDrawerOpen] = useState(false);
  const [hostDrawerMode, setHostDrawerMode] = useState<"create" | "edit">(
    "create",
  );
  const [editingHostId, setEditingHostId] = useState<string | undefined>(
    undefined,
  );
  const [inviteDrawerOpen, setInviteDrawerOpen] = useState(false);

  const selectedTeam = useMemo(
    () => teams.find((team) => team.id === selectedTeamId) ?? null,
    [selectedTeamId, teams],
  );
  const canManage =
    selectedTeam?.role === "owner" || selectedTeam?.role === "admin";
  const canRevealSecrets = canManage;
  const isOwner = selectedTeam?.role === "owner";

  const availableTabs = canManage ? ALL_TABS : BASE_TABS;

  // URL is the source of truth for activeTab. Fall back to "hosts" when the
  // query param is missing or refers to a tab the user can't see.
  const rawTab = searchParams.get("tab");
  const activeTab: DashboardTab = availableTabs.includes(rawTab as DashboardTab)
    ? (rawTab as DashboardTab)
    : "hosts";

  const setActiveTab = useCallback(
    (next: DashboardTab) => {
      const params = new URLSearchParams(Array.from(searchParams.entries()));
      params.set("tab", next);
      router.replace(`/teams?${params.toString()}`, { scroll: false });
    },
    [router, searchParams],
  );

  // If the current activeTab becomes unavailable (e.g. user just lost admin on
  // the selected team), bounce them to hosts.
  useEffect(() => {
    if (rawTab && !availableTabs.includes(rawTab as DashboardTab)) {
      setActiveTab("hosts");
    }
  }, [rawTab, availableTabs, setActiveTab]);

  // Initial load.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        setLoading(true);
        const [nextTeams, nextInvites] = await Promise.all([
          apiRequest<TeamSummary[]>("/api/teams/list"),
          apiRequest<InviteResponse>("/api/teams/invites"),
        ]);
        if (cancelled) return;
        setTeams(nextTeams);
        setInvites(nextInvites);
        setSelectedTeamId((cur) => cur || nextTeams[0]?.id || "");
      } catch (err) {
        if (!cancelled) toast.error(errorMessage(err, "load_failed"));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Per-team data.
  const loadTeamData = useCallback(async (teamId: string) => {
    try {
      const [nextMembers, nextHosts] = await Promise.all([
        apiRequest<TeamMember[]>(`/api/teams/${teamId}/members`),
        apiRequest<TeamHost[]>(`/api/teams/${teamId}/hosts`),
      ]);
      setMembers(nextMembers);
      setHosts(nextHosts);
    } catch (err) {
      toast.error(errorMessage(err, "team_load_failed"));
    }
  }, []);

  useEffect(() => {
    if (!selectedTeamId) {
      setMembers([]);
      setHosts([]);
      return;
    }
    void loadTeamData(selectedTeamId);
  }, [selectedTeamId, loadTeamData]);

  // Audit events (only when on the audit tab and manager).
  const loadAuditEvents = useCallback(async (teamId: string) => {
    try {
      const events = await apiRequest<TeamAuditEvent[]>(
        `/api/teams/${teamId}/audit`,
      );
      setAuditEvents(events);
    } catch (err) {
      toast.error(errorMessage(err, "audit_load_failed"));
    }
  }, []);

  const loadTeamTokens = useCallback(async (teamId: string) => {
    try {
      const tokens = await apiRequest<TeamAutomationToken[]>(
        `/api/teams/${teamId}/tokens`,
      );
      setTeamTokens(tokens);
    } catch (err) {
      toast.error(errorMessage(err, "tokens_load_failed"));
    }
  }, []);

  useEffect(() => {
    if (!selectedTeamId || !canManage || activeTab !== "audit") {
      if (activeTab !== "audit") setAuditEvents([]);
      return;
    }
    void loadAuditEvents(selectedTeamId);
  }, [selectedTeamId, canManage, activeTab, loadAuditEvents]);

  useEffect(() => {
    if (!selectedTeamId || !canManage || activeTab !== "tokens") {
      if (activeTab !== "tokens") setTeamTokens([]);
      return;
    }
    void loadTeamTokens(selectedTeamId);
  }, [selectedTeamId, canManage, activeTab, loadTeamTokens]);

  async function refreshTeams(preferredTeamId?: string) {
    try {
      const nextTeams = await apiRequest<TeamSummary[]>("/api/teams/list");
      setTeams(nextTeams);
      setSelectedTeamId((cur) => {
        const candidate = preferredTeamId || cur || nextTeams[0]?.id || "";
        return nextTeams.some((t) => t.id === candidate)
          ? candidate
          : nextTeams[0]?.id || "";
      });
    } catch (err) {
      toast.error(errorMessage(err, "list_teams_failed"));
    }
  }

  async function refreshInvites() {
    try {
      const next = await apiRequest<InviteResponse>("/api/teams/invites");
      setInvites(next);
    } catch (err) {
      toast.error(errorMessage(err, "list_invites_failed"));
    }
  }

  // Team actions.
  async function handleCreateTeam() {
    const name = await promptDialog({
      title: "Create team",
      label: "Team name",
      placeholder: "Production",
      confirmLabel: "Create",
      validate: (value) => (value ? null : "Name is required."),
    });
    if (!name) return;
    try {
      const created = await apiRequest<TeamSummary>("/api/teams/create", {
        method: "POST",
        body: JSON.stringify({ name }),
      });
      await refreshTeams(created.id);
      await refreshInvites();
      toast.success(`Created ${created.name}.`);
    } catch (err) {
      toast.error(errorMessage(err, "create_team_failed"));
    }
  }

  async function handleRenameTeam() {
    if (!selectedTeam) return;
    const name = await promptDialog({
      title: "Rename team",
      label: "Team name",
      defaultValue: selectedTeam.name,
      confirmLabel: "Rename",
      validate: (value) => (value ? null : "Name is required."),
    });
    if (!name || name === selectedTeam.name) return;
    try {
      await apiRequest(`/api/teams/${selectedTeam.id}`, {
        method: "PATCH",
        body: JSON.stringify({ name }),
      });
      await refreshTeams(selectedTeam.id);
      toast.success(`Renamed team to ${name}.`);
    } catch (err) {
      toast.error(errorMessage(err, "rename_team_failed"));
    }
  }

  async function handleDeleteTeam() {
    if (!selectedTeam) return;
    const ok = await confirmDialog({
      title: "Delete team",
      message: `Delete ${selectedTeam.name}? This removes all hosts, invites, and audit history.`,
      variant: "danger",
      confirmLabel: "Delete team",
    });
    if (!ok) return;
    try {
      await apiRequest(`/api/teams/${selectedTeam.id}`, { method: "DELETE" });
      // Close any open drawers — their teamId/hostId now points to a deleted team.
      setHostDrawerOpen(false);
      setInviteDrawerOpen(false);
      await refreshTeams();
      await refreshInvites();
      toast.success(`Deleted ${selectedTeam.name}.`);
    } catch (err) {
      toast.error(errorMessage(err, "delete_team_failed"));
    }
  }

  /**
   * Switching teams must also close any open drawers — otherwise the drawer
   * stays mounted pointing at the old team's teamId / hostId, and in create
   * mode a subsequent save would POST stale data into the newly-selected team.
   */
  const handleSelectTeam = useCallback((teamId: string) => {
    setHostDrawerOpen(false);
    setInviteDrawerOpen(false);
    setSelectedTeamId(teamId);
  }, []);

  async function handleMoveTeam(direction: -1 | 1) {
    if (!selectedTeam) return;
    const currentIndex = teams.findIndex((t) => t.id === selectedTeam.id);
    const nextIndex = currentIndex + direction;
    if (currentIndex < 0 || nextIndex < 0 || nextIndex >= teams.length) return;
    const reordered = [...teams];
    [reordered[currentIndex], reordered[nextIndex]] = [
      reordered[nextIndex],
      reordered[currentIndex],
    ];
    try {
      await apiRequest("/api/teams/reorder", {
        method: "POST",
        body: JSON.stringify({ teamIds: reordered.map((t) => t.id) }),
      });
      setTeams(reordered);
      toast.success(`Moved ${selectedTeam.name}.`);
    } catch (err) {
      toast.error(errorMessage(err, "reorder_teams_failed"));
    }
  }

  // Member actions.
  async function handleChangeMemberRole(memberId: string, role: TeamRole) {
    if (!selectedTeamId) return;
    try {
      await apiRequest(`/api/teams/${selectedTeamId}/members/${memberId}`, {
        method: "PATCH",
        body: JSON.stringify({ role }),
      });
      await loadTeamData(selectedTeamId);
      toast.success("Updated member role.");
    } catch (err) {
      toast.error(errorMessage(err, "update_member_failed"));
    }
  }

  async function handleRemoveMember(memberId: string, displayName: string) {
    if (!selectedTeamId) return;
    const ok = await confirmDialog({
      title: "Remove member",
      message: `Remove ${displayName} from this team?`,
      variant: "danger",
      confirmLabel: "Remove",
    });
    if (!ok) return;
    try {
      await apiRequest(`/api/teams/${selectedTeamId}/members/${memberId}`, {
        method: "DELETE",
      });
      await loadTeamData(selectedTeamId);
      toast.success(`Removed ${displayName}.`);
    } catch (err) {
      toast.error(errorMessage(err, "remove_member_failed"));
    }
  }

  // Invite actions.
  async function handleRevokeInvite(inviteId: string) {
    const ok = await confirmDialog({
      title: "Revoke invite",
      message: "The link will stop working immediately.",
      variant: "danger",
      confirmLabel: "Revoke",
    });
    if (!ok) return;
    try {
      await apiRequest(`/api/teams/invites/${inviteId}`, { method: "DELETE" });
      await refreshInvites();
      toast.success("Invite revoked.");
    } catch (err) {
      toast.error(errorMessage(err, "revoke_invite_failed"));
    }
  }

  // Host actions.
  function handleNewHost() {
    setHostDrawerMode("create");
    setEditingHostId(undefined);
    setHostDrawerOpen(true);
  }

  function handleEditHost(hostId: string) {
    setHostDrawerMode("edit");
    setEditingHostId(hostId);
    setHostDrawerOpen(true);
  }

  async function handleDeleteHostFromRow(hostId: string, label: string) {
    const ok = await confirmDialog({
      title: "Delete host",
      message: `Delete ${label}? Member credentials for this host will also be deleted.`,
      variant: "danger",
      confirmLabel: "Delete host",
    });
    if (!ok) return;
    try {
      await apiRequest(`/api/teams/hosts/${hostId}`, { method: "DELETE" });
      if (selectedTeamId) await loadTeamData(selectedTeamId);
      toast.success("Host deleted.");
    } catch (err) {
      toast.error(errorMessage(err, "delete_host_failed"));
    }
  }

  async function refreshAudit() {
    if (!selectedTeamId || !canManage) return;
    await loadAuditEvents(selectedTeamId);
  }

  async function handleCreateTeamToken() {
    if (!selectedTeamId || hosts.length === 0) return;
    const name = await promptDialog({
      title: "Create team token",
      label: "Token name",
      placeholder: "claude-office-agent",
      confirmLabel: "Create",
      validate: (value) => (value ? null : "Name is required."),
    });
    if (!name) return;
    try {
      const created = await apiRequest<{ rawToken: string }>(
        `/api/teams/${selectedTeamId}/tokens`,
        {
          method: "POST",
          body: JSON.stringify({
            name,
            hostIds: hosts.map((host) => host.id),
          }),
        },
      );
      await copyText(created.rawToken, "Copied team token. Store it now; it will not be shown again.");
      await loadTeamTokens(selectedTeamId);
      toast.success("Team token created.");
    } catch (err) {
      toast.error(errorMessage(err, "create_team_token_failed"));
    }
  }

  async function handleRevokeTeamToken(tokenDocId: string, name: string) {
    if (!selectedTeamId) return;
    const ok = await confirmDialog({
      title: "Revoke team token",
      message: `Revoke ${name}? Agents using it will stop working immediately.`,
      variant: "danger",
      confirmLabel: "Revoke",
    });
    if (!ok) return;
    try {
      await apiRequest(`/api/teams/${selectedTeamId}/tokens/${tokenDocId}`, {
        method: "POST",
      });
      await loadTeamTokens(selectedTeamId);
      toast.success("Team token revoked.");
    } catch (err) {
      toast.error(errorMessage(err, "revoke_team_token_failed"));
    }
  }

  async function handleDeleteTeamToken(tokenDocId: string, name: string) {
    if (!selectedTeamId) return;
    const ok = await confirmDialog({
      title: "Delete revoked token",
      message: `Delete revoked token ${name}? Execution logs will remain.`,
      variant: "danger",
      confirmLabel: "Delete",
    });
    if (!ok) return;
    try {
      await apiRequest(`/api/teams/${selectedTeamId}/tokens/${tokenDocId}`, {
        method: "DELETE",
      });
      await loadTeamTokens(selectedTeamId);
      toast.success("Revoked team token deleted.");
    } catch (err) {
      toast.error(errorMessage(err, "delete_team_token_failed"));
    }
  }

  const pendingInvitesCount = invites.sent.filter(
    (invite) => invite.status === "pending",
  ).length;

  const currentIndex = selectedTeam
    ? teams.findIndex((t) => t.id === selectedTeam.id)
    : -1;
  const canMoveUp = currentIndex > 0;
  const canMoveDown = currentIndex >= 0 && currentIndex < teams.length - 1;

  if (loading) {
    return (
      <main className="teams-page">
        <div className="team-bar">
          <div className="team-bar__row">
            <div className="team-bar__switcher" aria-busy="true">
              Loading teams…
            </div>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="teams-page">
      <TeamBar
        teams={teams}
        selectedTeam={selectedTeam}
        counts={{
          members: members.length,
          hosts: hosts.length,
          pendingInvites: pendingInvitesCount,
        }}
        activeTab={activeTab}
        availableTabs={availableTabs}
        onTabChange={setActiveTab}
        onSelectTeam={handleSelectTeam}
        onCreateTeam={handleCreateTeam}
        onRename={handleRenameTeam}
        onDelete={handleDeleteTeam}
        onMoveUp={() => void handleMoveTeam(-1)}
        onMoveDown={() => void handleMoveTeam(1)}
        onRefresh={() => {
          void refreshTeams(selectedTeamId);
          if (selectedTeamId) void loadTeamData(selectedTeamId);
        }}
        canMoveUp={canMoveUp}
        canMoveDown={canMoveDown}
      />

      {activeTab === "members" ? (
        <TabMembers
          selectedTeam={selectedTeam}
          members={members}
          canManage={canManage}
          onInvite={() => setInviteDrawerOpen(true)}
          onChangeRole={(id, role) => void handleChangeMemberRole(id, role)}
          onRemove={(id, name) => void handleRemoveMember(id, name)}
        />
      ) : null}

      {activeTab === "hosts" ? (
        <TabHosts
          selectedTeam={selectedTeam}
          hosts={hosts}
          canManage={canManage}
          activeHostId={hostDrawerOpen ? editingHostId : undefined}
          onNewHost={handleNewHost}
          onEditHost={handleEditHost}
          onDeleteHost={(id, label) => void handleDeleteHostFromRow(id, label)}
        />
      ) : null}

      {activeTab === "invites" ? (
        <TabInvites
          selectedTeam={selectedTeam}
          invites={invites}
          canInvite={canManage}
          onInvite={() => setInviteDrawerOpen(true)}
          onCopyLink={(url, email) =>
            void copyText(url, `Copied invite link for ${email}.`)
          }
          onRevoke={(id) => void handleRevokeInvite(id)}
        />
      ) : null}

      {activeTab === "tokens" ? (
        <TabTokens
          selectedTeam={selectedTeam}
          tokens={teamTokens}
          hosts={hosts}
          canManage={canManage}
          onCreate={() => void handleCreateTeamToken()}
          onRevoke={(id, name) => void handleRevokeTeamToken(id, name)}
          onDelete={(id, name) => void handleDeleteTeamToken(id, name)}
        />
      ) : null}

      {activeTab === "audit" ? (
        <TabAudit events={auditEvents} canView={canManage} />
      ) : null}

      {selectedTeamId ? (
        <HostDrawer
          open={hostDrawerOpen}
          teamId={selectedTeamId}
          mode={hostDrawerMode}
          hostId={editingHostId}
          hostLabel={
            editingHostId
              ? hosts.find((h) => h.id === editingHostId)?.label ??
                hosts.find((h) => h.id === editingHostId)?.hostname
              : undefined
          }
          canManageHosts={canManage}
          canRevealSecrets={canRevealSecrets}
          onClose={() => setHostDrawerOpen(false)}
          onChanged={() => {
            if (selectedTeamId) void loadTeamData(selectedTeamId);
          }}
          onAuditChanged={() => void refreshAudit()}
        />
      ) : null}

      {selectedTeamId && selectedTeam ? (
        <InviteDrawer
          open={inviteDrawerOpen}
          teamId={selectedTeamId}
          teamName={selectedTeam.name}
          onClose={() => setInviteDrawerOpen(false)}
          onSaved={() => void refreshInvites()}
        />
      ) : null}
    </main>
  );
}
