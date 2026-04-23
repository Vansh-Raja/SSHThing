"use client";

import type { FormEvent } from "react";
import { useEffect, useMemo, useState } from "react";

type TeamRole = "owner" | "admin" | "member";
type DashboardTab = "overview" | "members" | "hosts" | "invites" | "audit";

type TeamSummary = {
  id: string;
  name: string;
  slug: string;
  displayOrder: number;
  role: TeamRole;
};

type TeamMember = {
  id: string;
  teamId: string;
  clerkUserId: string;
  email: string;
  displayName: string;
  role: TeamRole;
  status: string;
  joinedAt: number | null;
};

type TeamInvite = {
  id: string;
  teamId: string;
  teamName: string;
  teamSlug: string;
  email: string;
  role: TeamRole;
  status: string;
  expiresAt: number;
  createdAt: number;
  shareUrl?: string | null;
};

type TeamHost = {
  id: string;
  teamId: string;
  label: string;
  hostname: string;
  username: string;
  port: number;
  group: string;
  tags: string[];
  notes: string;
  authMode?: string;
  credentialMode: "shared" | "per_member";
  credentialType: "none" | "password" | "private_key";
  secretVisibility: string;
  lastConnectedAt: number | null;
  createdAt: number;
  updatedAt: number;
  canManageHosts?: boolean;
  canRevealSecrets?: boolean;
  canEditNotes?: boolean;
};

type TeamHostDetail = TeamHost & {
  sharedCredential: string | null;
  sharedCredentialConfigured?: boolean;
};

type PersonalCredential = {
  hostId: string;
  credentialMode: "shared" | "per_member";
  credentialType: "none" | "password" | "private_key";
  username: string | null;
  hasCredential: boolean;
  secret: string;
  updatedAt?: number | null;
  viewerCanEdit?: boolean;
};

type CredentialRosterEntry = {
  memberId: string;
  displayName: string;
  email: string;
  role: TeamRole;
  isOwner: boolean;
  isCurrentUser: boolean;
  hasCredential: boolean;
  credentialType: "none" | "password" | "private_key";
  username: string | null;
  updatedAt: number | null;
};

type RevealedCredential = {
  hostId: string;
  memberClerkUserId?: string;
  credentialType: "none" | "password" | "private_key";
  username?: string | null;
  secret: string;
  updatedAt?: number | null;
};

type TeamAuditEvent = {
  id: string;
  teamId: string;
  actorClerkUserId: string;
  actorDisplayName: string;
  entityType: string;
  entityId: string;
  eventType: string;
  targetClerkUserId?: string | null;
  targetDisplayName?: string | null;
  summary: string;
  metadata?: {
    hostLabel?: string;
    credentialMode?: string;
    credentialType?: string;
  } | null;
  createdAt: number;
};

type InviteResponse = {
  incoming: TeamInvite[];
  sent: TeamInvite[];
};

type HostFormState = {
  label: string;
  hostname: string;
  username: string;
  port: string;
  group: string;
  tags: string;
  notes: string;
  credentialMode: "shared" | "per_member";
  credentialType: "none" | "password" | "private_key";
  sharedCredential: string;
};

type PersonalCredentialFormState = {
  username: string;
  credentialType: "password" | "private_key";
  secret: string;
};

const blankHostForm: HostFormState = {
  label: "",
  hostname: "",
  username: "",
  port: "22",
  group: "",
  tags: "",
  notes: "",
  credentialMode: "shared",
  credentialType: "none",
  sharedCredential: "",
};

const blankPersonalCredentialForm: PersonalCredentialFormState = {
  username: "",
  credentialType: "password",
  secret: "",
};

function parseTags(raw: string): string[] {
  return raw
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function formatTime(value: number | null) {
  if (!value) {
    return "Never";
  }
  return new Date(value).toLocaleString();
}

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });
  const data = (await response.json().catch(() => ({}))) as T & { error?: string };
  if (!response.ok) {
    throw new Error(data.error || "request_failed");
  }
  return data;
}

export default function TeamsDashboard() {
  const [teams, setTeams] = useState<TeamSummary[]>([]);
  const [selectedTeamId, setSelectedTeamId] = useState("");
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [hosts, setHosts] = useState<TeamHost[]>([]);
  const [invites, setInvites] = useState<InviteResponse>({ incoming: [], sent: [] });
  const [activeTab, setActiveTab] = useState<DashboardTab>("overview");
  const [createTeamName, setCreateTeamName] = useState("");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<TeamRole>("member");
  const [editingHostId, setEditingHostId] = useState("");
  const [hostForm, setHostForm] = useState<HostFormState>(blankHostForm);
  const [personalCredential, setPersonalCredential] = useState<PersonalCredential | null>(null);
  const [credentialRoster, setCredentialRoster] = useState<CredentialRosterEntry[]>([]);
  const [revealedCredential, setRevealedCredential] = useState<RevealedCredential | null>(null);
  const [auditEvents, setAuditEvents] = useState<TeamAuditEvent[]>([]);
  const [personalCredentialForm, setPersonalCredentialForm] = useState<PersonalCredentialFormState>(
    blankPersonalCredentialForm,
  );
  const [flash, setFlash] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);

  const selectedTeam = useMemo(
    () => teams.find((team) => team.id === selectedTeamId) ?? null,
    [selectedTeamId, teams],
  );
  const canManageMembers = selectedTeam?.role === "owner" || selectedTeam?.role === "admin";
  const canManageHosts = canManageMembers;
  const canManageTeam = canManageMembers;
  const canRevealSecrets = canManageMembers;

  async function refreshTeams(preferredTeamId?: string) {
    const nextTeams = await apiRequest<TeamSummary[]>("/api/teams/list");
    setTeams(nextTeams);
    setSelectedTeamId((current) => {
      const candidate = preferredTeamId || current || nextTeams[0]?.id || "";
      return nextTeams.some((team) => team.id === candidate) ? candidate : nextTeams[0]?.id || "";
    });
  }

  async function refreshInvites() {
    const nextInvites = await apiRequest<InviteResponse>("/api/teams/invites");
    setInvites(nextInvites);
  }

  async function refreshTeamData(teamId: string) {
    if (!teamId) {
      setMembers([]);
      setHosts([]);
      return;
    }

    const [nextMembers, nextHosts] = await Promise.all([
      apiRequest<TeamMember[]>(`/api/teams/${teamId}/members`),
      apiRequest<TeamHost[]>(`/api/teams/${teamId}/hosts`),
    ]);
    setMembers(nextMembers);
    setHosts(nextHosts);
  }

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        setLoading(true);
        setError("");
        const [nextTeams, nextInvites] = await Promise.all([
          apiRequest<TeamSummary[]>("/api/teams/list"),
          apiRequest<InviteResponse>("/api/teams/invites"),
        ]);
        if (cancelled) {
          return;
        }
        setTeams(nextTeams);
        setInvites(nextInvites);
        setSelectedTeamId((current) => current || nextTeams[0]?.id || "");
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : "load_failed");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function loadTeamData() {
      if (!selectedTeamId) {
        setMembers([]);
        setHosts([]);
        return;
      }

      try {
        setError("");
        const [nextMembers, nextHosts] = await Promise.all([
          apiRequest<TeamMember[]>(`/api/teams/${selectedTeamId}/members`),
          apiRequest<TeamHost[]>(`/api/teams/${selectedTeamId}/hosts`),
        ]);
        if (!cancelled) {
          setMembers(nextMembers);
          setHosts(nextHosts);
        }
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : "team_load_failed");
        }
      }
    }

    void loadTeamData();
    return () => {
      cancelled = true;
    };
  }, [selectedTeamId]);

  useEffect(() => {
    let cancelled = false;

    async function loadAuditEvents() {
      if (!selectedTeamId || !canManageHosts || activeTab !== "audit") {
        setAuditEvents([]);
        return;
      }

      try {
        const events = await apiRequest<TeamAuditEvent[]>(`/api/teams/${selectedTeamId}/audit`);
        if (!cancelled) {
          setAuditEvents(events);
        }
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : "audit_load_failed");
        }
      }
    }

    void loadAuditEvents();
    return () => {
      cancelled = true;
    };
  }, [activeTab, canManageHosts, selectedTeamId]);

  useEffect(() => {
    let cancelled = false;

    async function loadPersonalCredential() {
      if (!editingHostId || hostForm.credentialMode !== "per_member") {
        setPersonalCredential(null);
        setPersonalCredentialForm(blankPersonalCredentialForm);
        return;
      }

      try {
        const credential = await apiRequest<PersonalCredential>(
          `/api/teams/hosts/${editingHostId}/my-credential`,
        );
        if (cancelled) {
          return;
        }
        setPersonalCredential(credential);
        setPersonalCredentialForm({
          username: credential.username ?? "",
          credentialType:
            credential.credentialType === "private_key" ? "private_key" : "password",
          secret: credential.secret ?? "",
        });
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : "credential_load_failed");
        }
      }
    }

    void loadPersonalCredential();
    return () => {
      cancelled = true;
    };
  }, [editingHostId, hostForm.credentialMode]);

  useEffect(() => {
    let cancelled = false;

    async function loadCredentialRoster() {
      if (!editingHostId || hostForm.credentialMode !== "per_member" || !canManageHosts) {
        setCredentialRoster([]);
        return;
      }

      try {
        const roster = await apiRequest<CredentialRosterEntry[]>(
          `/api/teams/hosts/${editingHostId}/credentials`,
        );
        if (!cancelled) {
          setCredentialRoster(roster);
        }
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : "credential_roster_failed");
        }
      }
    }

    void loadCredentialRoster();
    return () => {
      cancelled = true;
    };
  }, [canManageHosts, editingHostId, hostForm.credentialMode]);

  async function copyText(value: string, successMessage: string) {
    await navigator.clipboard.writeText(value);
    setFlash(successMessage);
    setError("");
  }

  async function handleCreateTeam(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!createTeamName.trim()) {
      return;
    }

    try {
      setError("");
      const created = await apiRequest<TeamSummary>("/api/teams/create", {
        method: "POST",
        body: JSON.stringify({ name: createTeamName.trim() }),
      });
      setCreateTeamName("");
      await refreshTeams(created.id);
      await refreshInvites();
      setFlash(`Created ${created.name}.`);
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : "create_team_failed");
    }
  }

  async function handleRenameTeam() {
    if (!selectedTeam) {
      return;
    }
    const nextName = window.prompt("Rename team", selectedTeam.name)?.trim();
    if (!nextName || nextName === selectedTeam.name) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/${selectedTeam.id}`, {
        method: "PATCH",
        body: JSON.stringify({ name: nextName }),
      });
      await refreshTeams(selectedTeam.id);
      setFlash(`Renamed team to ${nextName}.`);
    } catch (renameError) {
      setError(renameError instanceof Error ? renameError.message : "rename_team_failed");
    }
  }

  async function handleDeleteTeam() {
    if (!selectedTeam || !window.confirm(`Delete ${selectedTeam.name}?`)) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/${selectedTeam.id}`, {
        method: "DELETE",
      });
      await refreshTeams();
      await refreshInvites();
      setFlash(`Deleted ${selectedTeam.name}.`);
      setEditingHostId("");
      setHostForm(blankHostForm);
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "delete_team_failed");
    }
  }

  async function handleMoveTeam(direction: -1 | 1) {
    if (!selectedTeam) {
      return;
    }
    const currentIndex = teams.findIndex((team) => team.id === selectedTeam.id);
    const nextIndex = currentIndex + direction;
    if (currentIndex < 0 || nextIndex < 0 || nextIndex >= teams.length) {
      return;
    }

    const reordered = [...teams];
    [reordered[currentIndex], reordered[nextIndex]] = [reordered[nextIndex], reordered[currentIndex]];

    try {
      setError("");
      await apiRequest("/api/teams/reorder", {
        method: "POST",
        body: JSON.stringify({ teamIds: reordered.map((team) => team.id) }),
      });
      setTeams(reordered);
      setFlash(`Moved ${selectedTeam.name}.`);
    } catch (reorderError) {
      setError(reorderError instanceof Error ? reorderError.message : "reorder_teams_failed");
    }
  }

  async function handleInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedTeamId || !inviteEmail.trim()) {
      return;
    }

    try {
      setError("");
      const invite = await apiRequest<TeamInvite>(`/api/teams/${selectedTeamId}/invites`, {
        method: "POST",
        body: JSON.stringify({ email: inviteEmail.trim(), role: inviteRole }),
      });
      setInviteEmail("");
      setInviteRole("member");
      await refreshInvites();
      if (invite.shareUrl) {
        await copyText(invite.shareUrl, `Copied invite link for ${invite.email}.`);
      } else {
        setFlash(`Created invite for ${invite.email}.`);
      }
    } catch (inviteError) {
      setError(inviteError instanceof Error ? inviteError.message : "invite_failed");
    }
  }

  async function handleMemberRoleChange(memberId: string, role: TeamRole) {
    if (!selectedTeamId) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/${selectedTeamId}/members/${memberId}`, {
        method: "PATCH",
        body: JSON.stringify({ role }),
      });
      await refreshTeamData(selectedTeamId);
      setFlash("Updated member role.");
    } catch (updateError) {
      setError(updateError instanceof Error ? updateError.message : "update_member_failed");
    }
  }

  async function handleRemoveMember(memberId: string, displayName: string) {
    if (!selectedTeamId || !window.confirm(`Remove ${displayName} from this team?`)) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/${selectedTeamId}/members/${memberId}`, {
        method: "DELETE",
      });
      await refreshTeamData(selectedTeamId);
      setFlash(`Removed ${displayName}.`);
    } catch (removeError) {
      setError(removeError instanceof Error ? removeError.message : "remove_member_failed");
    }
  }

  async function handleRevokeInvite(inviteId: string) {
    if (!window.confirm("Revoke this invite?")) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/invites/${inviteId}`, {
        method: "DELETE",
      });
      await refreshInvites();
      setFlash("Invite revoked.");
    } catch (revokeError) {
      setError(revokeError instanceof Error ? revokeError.message : "revoke_invite_failed");
    }
  }

  async function handleEditHost(hostId: string) {
    try {
      setError("");
      setDetailLoading(true);
      setRevealedCredential(null);
      const host = await apiRequest<TeamHostDetail>(`/api/teams/hosts/${hostId}`);
      setEditingHostId(host.id);
      setHostForm({
        label: host.label,
        hostname: host.hostname,
        username: host.username,
        port: String(host.port || 22),
        group: host.group,
        tags: host.tags.join(", "),
        notes: host.notes ?? "",
        credentialMode: host.credentialMode,
        credentialType: host.credentialType,
        sharedCredential: host.sharedCredential ?? "",
      });
      setActiveTab("hosts");
    } catch (hostError) {
      setError(hostError instanceof Error ? hostError.message : "host_load_failed");
    } finally {
      setDetailLoading(false);
    }
  }

  function handleNewHost() {
    setEditingHostId("");
    setHostForm(blankHostForm);
    setPersonalCredential(null);
    setCredentialRoster([]);
    setRevealedCredential(null);
    setPersonalCredentialForm(blankPersonalCredentialForm);
    setActiveTab("hosts");
  }

  async function handleSaveHost(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedTeamId) {
      return;
    }

    try {
      setError("");
      const payload = {
        label: hostForm.label.trim() || hostForm.hostname.trim(),
        hostname: hostForm.hostname.trim(),
        username: hostForm.username.trim(),
        port: Number(hostForm.port) || 22,
        group: hostForm.group.trim(),
        tags: parseTags(hostForm.tags),
        notes: hostForm.notes.trim(),
        credentialMode: hostForm.credentialMode,
        credentialType: hostForm.credentialType,
        secretVisibility: "revealed_to_access_holders",
        sharedCredential:
          hostForm.credentialMode === "shared" && hostForm.credentialType !== "none"
            ? hostForm.sharedCredential
            : null,
      };

      if (editingHostId) {
        await apiRequest(`/api/teams/hosts/${editingHostId}`, {
          method: "PATCH",
          body: JSON.stringify(payload),
        });
      } else {
        await apiRequest(`/api/teams/${selectedTeamId}/hosts`, {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }

      await refreshTeamData(selectedTeamId);
      setFlash(editingHostId ? "Host updated." : "Host created.");
      if (!editingHostId) {
        handleNewHost();
      } else {
        await handleEditHost(editingHostId);
      }
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "save_host_failed");
    }
  }

  async function handleSaveNotes() {
    if (!editingHostId) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/hosts/${editingHostId}`, {
        method: "PATCH",
        body: JSON.stringify({ notes: hostForm.notes.trim() }),
      });
      if (selectedTeamId) {
        await refreshTeamData(selectedTeamId);
      }
      setFlash("Notes saved.");
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "save_notes_failed");
    }
  }

  async function handleDeleteHost() {
    if (!editingHostId || !window.confirm("Delete this host?")) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/hosts/${editingHostId}`, {
        method: "DELETE",
      });
      if (selectedTeamId) {
        await refreshTeamData(selectedTeamId);
      }
      handleNewHost();
      setFlash("Host deleted.");
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "delete_host_failed");
    }
  }

  async function handleSavePersonalCredential(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editingHostId) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/hosts/${editingHostId}/my-credential`, {
        method: "PUT",
        body: JSON.stringify(personalCredentialForm),
      });
      const nextCredential = await apiRequest<PersonalCredential>(
        `/api/teams/hosts/${editingHostId}/my-credential`,
      );
      setPersonalCredential(nextCredential);
      setPersonalCredentialForm({
        username: nextCredential.username ?? "",
        credentialType:
          nextCredential.credentialType === "private_key" ? "private_key" : "password",
        secret: nextCredential.secret ?? "",
      });
      if (canManageHosts) {
        const roster = await apiRequest<CredentialRosterEntry[]>(
          `/api/teams/hosts/${editingHostId}/credentials`,
        );
        setCredentialRoster(roster);
      }
      setFlash("Personal credential saved.");
    } catch (credentialError) {
      setError(credentialError instanceof Error ? credentialError.message : "save_credential_failed");
    }
  }

  async function handleDeletePersonalCredential() {
    if (!editingHostId || !window.confirm("Delete your personal credential for this host?")) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/hosts/${editingHostId}/my-credential`, {
        method: "DELETE",
      });
      setPersonalCredential({
        hostId: editingHostId,
        credentialMode: "per_member",
        credentialType: hostForm.credentialType,
        username: null,
        hasCredential: false,
        secret: "",
        updatedAt: null,
      });
      setPersonalCredentialForm(blankPersonalCredentialForm);
      if (canManageHosts) {
        const roster = await apiRequest<CredentialRosterEntry[]>(
          `/api/teams/hosts/${editingHostId}/credentials`,
        );
        setCredentialRoster(roster);
      }
      setFlash("Personal credential deleted.");
    } catch (credentialError) {
      setError(credentialError instanceof Error ? credentialError.message : "delete_credential_failed");
    }
  }

  async function handleRevealSharedCredential() {
    if (!editingHostId || !window.confirm("Reveal the shared credential? This action will be logged.")) {
      return;
    }

    try {
      setError("");
      const revealed = await apiRequest<RevealedCredential>(
        `/api/teams/hosts/${editingHostId}/credentials/shared/reveal`,
        {
          method: "POST",
        },
      );
      setRevealedCredential(revealed);
      setHostForm((current) => ({ ...current, sharedCredential: revealed.secret }));
      setFlash("Shared credential revealed and logged.");
      if (selectedTeamId) {
        const events = await apiRequest<TeamAuditEvent[]>(`/api/teams/${selectedTeamId}/audit`);
        setAuditEvents(events);
      }
    } catch (revealError) {
      setError(revealError instanceof Error ? revealError.message : "reveal_shared_credential_failed");
    }
  }

  async function handleRevealMemberCredential(memberId: string) {
    if (!editingHostId || !window.confirm("Reveal this member credential? This action will be logged.")) {
      return;
    }

    try {
      setError("");
      const revealed = await apiRequest<RevealedCredential>(
        `/api/teams/hosts/${editingHostId}/credentials/${memberId}/reveal`,
        {
          method: "POST",
        },
      );
      setRevealedCredential(revealed);
      setFlash("Member credential revealed and logged.");
      if (selectedTeamId) {
        const events = await apiRequest<TeamAuditEvent[]>(`/api/teams/${selectedTeamId}/audit`);
        setAuditEvents(events);
      }
    } catch (revealError) {
      setError(revealError instanceof Error ? revealError.message : "reveal_member_credential_failed");
    }
  }

  async function handleDeleteMemberCredential(memberId: string, displayName: string) {
    if (!editingHostId || !window.confirm(`Delete ${displayName}'s credential? This action will be logged.`)) {
      return;
    }

    try {
      setError("");
      await apiRequest(`/api/teams/hosts/${editingHostId}/credentials/${memberId}`, {
        method: "DELETE",
      });
      setRevealedCredential(null);
      const roster = await apiRequest<CredentialRosterEntry[]>(
        `/api/teams/hosts/${editingHostId}/credentials`,
      );
      setCredentialRoster(roster);
      if (selectedTeamId) {
        const events = await apiRequest<TeamAuditEvent[]>(`/api/teams/${selectedTeamId}/audit`);
        setAuditEvents(events);
      }
      setFlash(`Deleted ${displayName}'s credential.`);
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "delete_member_credential_failed");
    }
  }

  if (loading) {
    return (
      <div className="block stack">
        <span className="eyebrow">Loading</span>
        <h1 className="text-xl fw-800">Syncing your teams.</h1>
        <p className="muted text-sm">Fetching teams, hosts, members, and pending invites.</p>
      </div>
    );
  }

  return (
    <section className="teams-layout">
      <aside className="block stack teams-sidebar" style={{ gap: 18 }}>
        <div className="stack" style={{ gap: 8 }}>
          <span className="eyebrow">Teams</span>
          <h1 className="text-xl fw-800" style={{ lineHeight: 1.1 }}>
            Browser control plane.
          </h1>
          <p className="muted text-sm" style={{ lineHeight: 1.6 }}>
            Create teams here, manage members and invites here, and keep the TUI focused on browsing and connecting.
          </p>
        </div>

        <form className="stack" style={{ gap: 10 }} onSubmit={handleCreateTeam}>
          <label className="field">
            <span className="field__label">New team</span>
            <input
              className="field__input"
              value={createTeamName}
              onChange={(event) => setCreateTeamName(event.target.value)}
              placeholder="Production"
            />
          </label>
          <button className="btn btn--primary" type="submit">
            Create team
          </button>
        </form>

        <div className="stack" style={{ gap: 10 }}>
          {teams.length === 0 ? (
            <div className="teams-empty">
              No teams yet. Create one to start sharing hosts.
            </div>
          ) : (
            teams.map((team) => (
              <button
                key={team.id}
                className={`teams-listItem ${team.id === selectedTeamId ? "teams-listItem--active" : ""}`}
                type="button"
                onClick={() => setSelectedTeamId(team.id)}
              >
                <span className="teams-listItem__title">{team.name}</span>
                <span className="teams-listItem__meta">
                  {team.role} · {team.slug}
                </span>
              </button>
            ))
          )}
        </div>
      </aside>

      <div className="stack teams-main" style={{ gap: 16 }}>
        <div className="block stack teams-panel" style={{ gap: 14 }}>
          <div className="row-between teams-toolbar" style={{ alignItems: "flex-start" }}>
            <div className="stack" style={{ gap: 8 }}>
              <span className="eyebrow">Current team</span>
              <h2 className="text-xl fw-800" style={{ lineHeight: 1.1 }}>
                {selectedTeam?.name || "No team selected"}
              </h2>
              <p className="muted text-sm">
                {selectedTeam
                  ? `${selectedTeam.slug} · ${selectedTeam.role}`
                  : "You can still accept incoming invites without creating a team first."}
              </p>
            </div>

            {selectedTeam ? (
              <div className="row teams-actions">
                <button className="btn" type="button" onClick={() => void refreshTeams(selectedTeam.id)}>
                  Refresh
                </button>
                {canManageTeam ? (
                  <>
                    <button className="btn" type="button" onClick={handleRenameTeam}>
                      Rename
                    </button>
                    <button className="btn" type="button" onClick={() => void handleMoveTeam(-1)}>
                      Move up
                    </button>
                    <button className="btn" type="button" onClick={() => void handleMoveTeam(1)}>
                      Move down
                    </button>
                  </>
                ) : null}
                {selectedTeam.role === "owner" ? (
                  <button className="btn" type="button" onClick={handleDeleteTeam}>
                    Delete
                  </button>
                ) : null}
              </div>
            ) : null}
          </div>

          <div className="teams-tabs">
            {((canManageHosts
              ? ["overview", "members", "hosts", "invites", "audit"]
              : ["overview", "members", "hosts", "invites"]) as DashboardTab[]).map((tab) => (
              <button
                key={tab}
                className={`teams-tab ${activeTab === tab ? "teams-tab--active" : ""}`}
                type="button"
                onClick={() => setActiveTab(tab)}
              >
                {tab}
              </button>
            ))}
          </div>

          {flash ? <div className="teams-notice teams-notice--success">{flash}</div> : null}
          {error ? <div className="teams-notice teams-notice--error">{error}</div> : null}
        </div>

        {activeTab === "overview" ? (
          <div className="grid-2 teams-sectionGrid">
            <div className="block stack teams-panel">
              <span className="eyebrow">Status</span>
              <h3 className="text-lg fw-800">What this team can do now</h3>
              <p className="muted text-sm" style={{ lineHeight: 1.6 }}>
                Hosts and credentials are stored in Convex. The browser owns team setup, invites, and host management. The TUI reads the same API and uses connect-config to launch your local SSH client.
              </p>
              <div className="stack" style={{ gap: 8 }}>
                <div className="teams-stat">
                  <strong>{teams.length}</strong>
                  <span>teams visible to you</span>
                </div>
                <div className="teams-stat">
                  <strong>{members.length}</strong>
                  <span>active members in the selected team</span>
                </div>
                <div className="teams-stat">
                  <strong>{hosts.length}</strong>
                  <span>hosts in the selected team</span>
                </div>
              </div>
            </div>

            <div className="block stack teams-panel">
              <span className="eyebrow">Terminal handoff</span>
              <h3 className="text-lg fw-800">Connect from the TUI</h3>
              <p className="muted text-sm" style={{ lineHeight: 1.6 }}>
                Run <code>sshthing login</code>, switch to Teams mode with <code>Shift+T</code>, and the terminal will fetch the same teams and hosts shown here.
              </p>
              <div className="term" style={{ fontSize: 12 }}>
                <div className="term__bar">~ sshthing</div>
                <div className="term__body">
                  <span className="term__line">
                    <span className="term__prompt">$</span> sshthing login
                  </span>
                  <span className="term__line muted">→ browser auth complete</span>
                  <span className="term__line">
                    <span className="term__prompt">$</span> shift+t
                  </span>
                  <span className="term__line muted">→ browse team hosts and connect</span>
                </div>
              </div>
            </div>
          </div>
        ) : null}

        {activeTab === "members" ? (
          <div className="grid-2 teams-sectionGrid">
            <div className="block stack teams-panel">
              <div className="row-between">
                <div>
                  <span className="eyebrow">Members</span>
                  <h3 className="text-lg fw-800">Team access</h3>
                </div>
              </div>

              {selectedTeam ? (
                members.length > 0 ? (
                  <div className="stack" style={{ gap: 10 }}>
                    {members.map((member) => (
                      <div key={member.id} className="teams-cardRow">
                        <div className="stack" style={{ gap: 4 }}>
                          <strong>{member.displayName || member.email}</strong>
                          <span className="muted text-sm">{member.email}</span>
                        </div>
                        <div className="row">
                          <span className="tag">{member.role}</span>
                          {canManageMembers && member.role !== "owner" ? (
                            <>
                              <select
                                className="field__input field__input--compact"
                                value={member.role}
                                onChange={(event) =>
                                  void handleMemberRoleChange(member.id, event.target.value as TeamRole)
                                }
                              >
                                <option value="admin">admin</option>
                                <option value="member">member</option>
                              </select>
                              <button
                                className="btn"
                                type="button"
                                onClick={() =>
                                  void handleRemoveMember(member.id, member.displayName || member.email)
                                }
                              >
                                Remove
                              </button>
                            </>
                          ) : null}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="teams-empty">No members yet.</div>
                )
              ) : (
                <div className="teams-empty">Select a team first.</div>
              )}
            </div>

            <div className="block stack teams-panel">
              <span className="eyebrow">Invite</span>
              <h3 className="text-lg fw-800">Add someone to this team</h3>
              {canManageMembers && selectedTeam ? (
                <form className="stack" style={{ gap: 12 }} onSubmit={handleInvite}>
                  <label className="field">
                    <span className="field__label">Email</span>
                    <input
                      className="field__input"
                      type="email"
                      value={inviteEmail}
                      onChange={(event) => setInviteEmail(event.target.value)}
                      placeholder="ops@example.com"
                    />
                  </label>
                  <label className="field">
                    <span className="field__label">Role</span>
                    <select
                      className="field__input"
                      value={inviteRole}
                      onChange={(event) => setInviteRole(event.target.value as TeamRole)}
                    >
                      <option value="member">member</option>
                      <option value="admin">admin</option>
                    </select>
                  </label>
                  <button className="btn btn--primary" type="submit">
                    Create invite link
                  </button>
                </form>
              ) : (
                <div className="teams-empty">
                  {selectedTeam
                    ? "Only owners and admins can invite members."
                    : "Create or select a team before inviting anyone."}
                </div>
              )}
            </div>
          </div>
        ) : null}

        {activeTab === "hosts" ? (
          <div className="grid-2 teams-sectionGrid">
            <div className="block stack teams-panel">
              <div className="row-between">
                <div>
                  <span className="eyebrow">Hosts</span>
                  <h3 className="text-lg fw-800">Team hosts</h3>
                </div>
                {canManageHosts ? (
                  <button className="btn" type="button" onClick={handleNewHost}>
                    New host
                  </button>
                ) : null}
              </div>

              {selectedTeam ? (
                hosts.length > 0 ? (
                  <div className="stack" style={{ gap: 10 }}>
                    {hosts.map((host) => (
                      <button
                        key={host.id}
                        className={`teams-listItem ${host.id === editingHostId ? "teams-listItem--active" : ""}`}
                        type="button"
                        onClick={() => void handleEditHost(host.id)}
                      >
                        <span className="teams-listItem__title">{host.label || host.hostname}</span>
                        <span className="teams-listItem__meta">
                          {host.username}@{host.hostname}:{host.port} · {host.credentialMode} · {host.credentialType}
                        </span>
                      </button>
                    ))}
                  </div>
                ) : (
                  <div className="teams-empty">No hosts in this team yet.</div>
                )
              ) : (
                <div className="teams-empty">Select a team first.</div>
              )}
            </div>

            <div className="block stack teams-panel">
              <div className="row-between">
                <div>
                  <span className="eyebrow">{editingHostId ? "Edit host" : "Create host"}</span>
                  <h3 className="text-lg fw-800">
                    {editingHostId ? "Host details" : "New host"}
                  </h3>
                </div>
                {detailLoading ? <span className="muted text-sm">Loading…</span> : null}
              </div>

              {selectedTeam ? (
                canManageHosts ? (
                  <>
                    <form className="stack" style={{ gap: 12 }} onSubmit={handleSaveHost}>
                      <label className="field">
                        <span className="field__label">Label</span>
                        <input
                          className="field__input"
                          value={hostForm.label}
                          onChange={(event) => setHostForm((current) => ({ ...current, label: event.target.value }))}
                          placeholder="Production bastion"
                        />
                      </label>

                      <div className="grid-2">
                        <label className="field">
                          <span className="field__label">Hostname</span>
                          <input
                            className="field__input"
                            value={hostForm.hostname}
                            onChange={(event) =>
                              setHostForm((current) => ({ ...current, hostname: event.target.value }))
                            }
                            placeholder="server.example.com"
                          />
                        </label>
                        <label className="field">
                          <span className="field__label">Username</span>
                          <input
                            className="field__input"
                            value={hostForm.username}
                            onChange={(event) =>
                              setHostForm((current) => ({ ...current, username: event.target.value }))
                            }
                            placeholder="root"
                          />
                        </label>
                      </div>

                      <div className="grid-2">
                        <label className="field">
                          <span className="field__label">Port</span>
                          <input
                            className="field__input"
                            value={hostForm.port}
                            onChange={(event) => setHostForm((current) => ({ ...current, port: event.target.value }))}
                            placeholder="22"
                          />
                        </label>
                        <label className="field">
                          <span className="field__label">Group</span>
                          <input
                            className="field__input"
                            value={hostForm.group}
                            onChange={(event) =>
                              setHostForm((current) => ({ ...current, group: event.target.value }))
                            }
                            placeholder="prod"
                          />
                        </label>
                      </div>

                      <label className="field">
                        <span className="field__label">Tags</span>
                        <input
                          className="field__input"
                          value={hostForm.tags}
                          onChange={(event) => setHostForm((current) => ({ ...current, tags: event.target.value }))}
                          placeholder="ssh, linux, us-east-1"
                        />
                      </label>

                      <label className="field">
                        <span className="field__label">Notes</span>
                        <textarea
                          className="field__input field__textarea"
                          value={hostForm.notes}
                          onChange={(event) => setHostForm((current) => ({ ...current, notes: event.target.value }))}
                          placeholder="Shared deployment notes, caveats, or runbook steps"
                        />
                      </label>

                      <div className="grid-2">
                        <label className="field">
                          <span className="field__label">Credential mode</span>
                          <select
                            className="field__input"
                            value={hostForm.credentialMode}
                            onChange={(event) =>
                              setHostForm((current) => ({
                                ...current,
                                credentialMode: event.target.value as "shared" | "per_member",
                                sharedCredential:
                                  event.target.value === "per_member" ? "" : current.sharedCredential,
                              }))
                            }
                          >
                            <option value="shared">shared</option>
                            <option value="per_member">per member</option>
                          </select>
                        </label>
                        <label className="field">
                          <span className="field__label">Credential type</span>
                          <select
                            className="field__input"
                            value={hostForm.credentialType}
                            onChange={(event) =>
                              setHostForm((current) => ({
                                ...current,
                                credentialType: event.target.value as "none" | "password" | "private_key",
                              }))
                            }
                          >
                            <option value="none">none</option>
                            <option value="password">password</option>
                            <option value="private_key">private key</option>
                          </select>
                        </label>
                      </div>

                      {hostForm.credentialMode === "shared" && hostForm.credentialType !== "none" ? (
                        <div className="stack" style={{ gap: 10 }}>
                          <label className="field">
                            <span className="field__label">Shared secret</span>
                            <textarea
                              className="field__input field__textarea"
                              value={hostForm.sharedCredential}
                              onChange={(event) =>
                                setHostForm((current) => ({ ...current, sharedCredential: event.target.value }))
                              }
                              placeholder={
                                hostForm.credentialType === "private_key"
                                  ? "Paste the private key"
                                  : "Paste the password"
                              }
                            />
                          </label>
                          {editingHostId ? (
                            <div className="row">
                              <button className="btn" type="button" onClick={() => void handleRevealSharedCredential()}>
                                Reveal shared credential
                              </button>
                              <span className="muted text-sm">
                                {hostForm.sharedCredential
                                  ? "Editing revealed/shared secret"
                                  : "Reveal is audited and only available to owners/admins."}
                              </span>
                            </div>
                          ) : null}
                        </div>
                      ) : null}

                      <div className="row">
                        <button className="btn btn--primary" type="submit">
                          {editingHostId ? "Save host" : "Create host"}
                        </button>
                        {editingHostId ? (
                          <>
                            <button className="btn" type="button" onClick={handleNewHost}>
                              Clear form
                            </button>
                            <button className="btn" type="button" onClick={handleDeleteHost}>
                              Delete
                            </button>
                          </>
                        ) : null}
                      </div>
                    </form>

                    {editingHostId && hostForm.credentialMode === "per_member" ? (
                      <>
                        <div className="block stack teams-subpanel" style={{ gap: 12 }}>
                          <span className="eyebrow">Your credential</span>
                          <p className="muted text-sm">
                            This host uses per-member credentials. Save your own secret here; other members cannot read it through the self-service path.
                          </p>
                          <p className="muted text-sm">
                            Status: {personalCredential?.hasCredential ? "configured" : "not configured"}
                            {personalCredential?.updatedAt ? ` · updated ${formatTime(personalCredential.updatedAt)}` : ""}
                          </p>
                          <form className="stack" style={{ gap: 12 }} onSubmit={handleSavePersonalCredential}>
                            <label className="field">
                              <span className="field__label">Username override</span>
                              <input
                                className="field__input"
                                value={personalCredentialForm.username}
                                onChange={(event) =>
                                  setPersonalCredentialForm((current) => ({
                                    ...current,
                                    username: event.target.value,
                                  }))
                                }
                                placeholder="Optional"
                              />
                            </label>
                            <label className="field">
                              <span className="field__label">Credential type</span>
                              <select
                                className="field__input"
                                value={personalCredentialForm.credentialType}
                                onChange={(event) =>
                                  setPersonalCredentialForm((current) => ({
                                    ...current,
                                    credentialType: event.target.value as "password" | "private_key",
                                  }))
                                }
                              >
                                <option value="password">password</option>
                                <option value="private_key">private key</option>
                              </select>
                            </label>
                            <label className="field">
                              <span className="field__label">Secret</span>
                              <textarea
                                className="field__input field__textarea"
                                value={personalCredentialForm.secret}
                                onChange={(event) =>
                                  setPersonalCredentialForm((current) => ({
                                    ...current,
                                    secret: event.target.value,
                                  }))
                                }
                                placeholder={
                                  personalCredentialForm.credentialType === "private_key"
                                    ? "Paste your private key"
                                    : "Paste your password"
                                }
                              />
                            </label>
                            <div className="row">
                              <button className="btn btn--primary" type="submit">
                                Save my credential
                              </button>
                              {personalCredential?.hasCredential ? (
                                <button className="btn" type="button" onClick={handleDeletePersonalCredential}>
                                  Delete my credential
                                </button>
                              ) : null}
                            </div>
                          </form>
                        </div>

                        <div className="block stack teams-subpanel" style={{ gap: 12 }}>
                          <span className="eyebrow">Member credentials</span>
                          <p className="muted text-sm">
                            Owners and admins can inspect configuration status, reveal secrets, and delete member credentials. Reveal and delete actions are audited.
                          </p>
                          {credentialRoster.length > 0 ? (
                            <div className="stack" style={{ gap: 10 }}>
                              {credentialRoster.map((entry) => (
                                <div key={entry.memberId} className="teams-cardRow">
                                  <div className="stack" style={{ gap: 4 }}>
                                    <strong>{entry.displayName}</strong>
                                    <span className="muted text-sm">
                                      {entry.role} · {entry.email || entry.memberId}
                                    </span>
                                    <span className="muted text-sm">
                                      {entry.hasCredential
                                        ? `${entry.credentialType}${entry.username ? ` · ${entry.username}` : ""}${entry.updatedAt ? ` · ${formatTime(entry.updatedAt)}` : ""}`
                                        : "missing credential"}
                                    </span>
                                  </div>
                                  <div className="row">
                                    <button
                                      className="btn"
                                      type="button"
                                      onClick={() => void handleRevealMemberCredential(entry.memberId)}
                                      disabled={!entry.hasCredential}
                                    >
                                      Reveal
                                    </button>
                                    <button
                                      className="btn"
                                      type="button"
                                      onClick={() => void handleDeleteMemberCredential(entry.memberId, entry.displayName)}
                                      disabled={!entry.hasCredential}
                                    >
                                      Delete
                                    </button>
                                  </div>
                                </div>
                              ))}
                            </div>
                          ) : (
                            <div className="teams-empty">No member credential data yet.</div>
                          )}
                        </div>
                      </>
                    ) : null}

                    {revealedCredential ? (
                      <div className="block stack teams-subpanel" style={{ gap: 12 }}>
                        <span className="eyebrow">Revealed credential</span>
                        <p className="muted text-sm">
                          This output was revealed through an audited admin action.
                        </p>
                        <div className="stack" style={{ gap: 6 }}>
                          <span className="muted text-sm">
                            {revealedCredential.credentialType}
                            {revealedCredential.username ? ` · ${revealedCredential.username}` : ""}
                            {revealedCredential.updatedAt ? ` · ${formatTime(revealedCredential.updatedAt)}` : ""}
                          </span>
                          <textarea
                            className="field__input field__textarea"
                            value={revealedCredential.secret}
                            readOnly
                          />
                        </div>
                        <div className="row">
                          <button
                            className="btn"
                            type="button"
                            onClick={() => void copyText(revealedCredential.secret, "Credential copied.")}
                          >
                            Copy secret
                          </button>
                          <button className="btn" type="button" onClick={() => setRevealedCredential(null)}>
                            Clear
                          </button>
                        </div>
                      </div>
                    ) : null}
                  </>
                ) : (
                  editingHostId ? (
                    <div className="stack" style={{ gap: 12 }}>
                      <div className="block stack teams-subpanel" style={{ gap: 10 }}>
                        <span className="eyebrow">Host details</span>
                        <h3 className="text-lg fw-800">{hostForm.label || hostForm.hostname}</h3>
                        <p className="muted text-sm">
                          {hostForm.username}@{hostForm.hostname}:{hostForm.port}
                        </p>
                        <p className="muted text-sm">
                          {hostForm.credentialMode} · {hostForm.credentialType}
                        </p>
                      </div>

                      <div className="block stack teams-subpanel" style={{ gap: 12 }}>
                        <span className="eyebrow">Notes</span>
                        <p className="muted text-sm">
                          Shared host notes are collaborative and visible in the browser and TUI.
                        </p>
                        <label className="field">
                          <span className="field__label">Host notes</span>
                          <textarea
                            className="field__input field__textarea"
                            value={hostForm.notes}
                            onChange={(event) => setHostForm((current) => ({ ...current, notes: event.target.value }))}
                            placeholder="Shared deployment notes, caveats, or runbook steps"
                          />
                        </label>
                        <div className="row">
                          <button className="btn btn--primary" type="button" onClick={() => void handleSaveNotes()}>
                            Save notes
                          </button>
                        </div>
                      </div>

                      {hostForm.credentialMode === "per_member" ? (
                        <div className="block stack teams-subpanel" style={{ gap: 12 }}>
                          <span className="eyebrow">Your credential</span>
                          <p className="muted text-sm">
                            This host uses per-member credentials. Save your own secret here.
                          </p>
                          <p className="muted text-sm">
                            Status: {personalCredential?.hasCredential ? "configured" : "not configured"}
                            {personalCredential?.updatedAt ? ` · updated ${formatTime(personalCredential.updatedAt)}` : ""}
                          </p>
                          <form className="stack" style={{ gap: 12 }} onSubmit={handleSavePersonalCredential}>
                            <label className="field">
                              <span className="field__label">Username override</span>
                              <input
                                className="field__input"
                                value={personalCredentialForm.username}
                                onChange={(event) =>
                                  setPersonalCredentialForm((current) => ({
                                    ...current,
                                    username: event.target.value,
                                  }))
                                }
                                placeholder="Optional"
                              />
                            </label>
                            <label className="field">
                              <span className="field__label">Credential type</span>
                              <select
                                className="field__input"
                                value={personalCredentialForm.credentialType}
                                onChange={(event) =>
                                  setPersonalCredentialForm((current) => ({
                                    ...current,
                                    credentialType: event.target.value as "password" | "private_key",
                                  }))
                                }
                              >
                                <option value="password">password</option>
                                <option value="private_key">private key</option>
                              </select>
                            </label>
                            <label className="field">
                              <span className="field__label">Secret</span>
                              <textarea
                                className="field__input field__textarea"
                                value={personalCredentialForm.secret}
                                onChange={(event) =>
                                  setPersonalCredentialForm((current) => ({
                                    ...current,
                                    secret: event.target.value,
                                  }))
                                }
                                placeholder={
                                  personalCredentialForm.credentialType === "private_key"
                                    ? "Paste your private key"
                                    : "Paste your password"
                                }
                              />
                            </label>
                            <div className="row">
                              <button className="btn btn--primary" type="submit">
                                Save my credential
                              </button>
                              {personalCredential?.hasCredential ? (
                                <button className="btn" type="button" onClick={handleDeletePersonalCredential}>
                                  Delete my credential
                                </button>
                              ) : null}
                            </div>
                          </form>
                        </div>
                      ) : (
                        <div className="teams-empty">
                          This host uses a shared credential. Only owners and admins can reveal or change it.
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="teams-empty">
                      You can browse hosts in this team, edit shared notes, and manage your own credential where needed.
                    </div>
                  )
                )
              ) : (
                <div className="teams-empty">Select a team first.</div>
              )}
            </div>
          </div>
        ) : null}

        {activeTab === "audit" ? (
          <div className="block stack teams-panel">
            <span className="eyebrow">Audit</span>
            <h3 className="text-lg fw-800">Credential access log</h3>
            <p className="muted text-sm">
              Owner/admin reveal and delete actions for team credentials are recorded here.
            </p>
            {auditEvents.length > 0 ? (
              <div className="stack" style={{ gap: 10 }}>
                {auditEvents.map((event) => (
                  <div key={event.id} className="teams-cardRow">
                    <div className="stack" style={{ gap: 4 }}>
                      <strong>{event.summary}</strong>
                      <span className="muted text-sm">
                        {event.actorDisplayName}
                        {event.targetDisplayName ? ` → ${event.targetDisplayName}` : ""}
                      </span>
                      <span className="muted text-sm">
                        {formatTime(event.createdAt)}
                        {event.metadata?.hostLabel ? ` · ${event.metadata.hostLabel}` : ""}
                        {event.metadata?.credentialType ? ` · ${event.metadata.credentialType}` : ""}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="teams-empty">No audited credential events yet.</div>
            )}
          </div>
        ) : null}

        {activeTab === "invites" ? (
          <div className="grid-2 teams-sectionGrid">
            <div className="block stack teams-panel">
              <span className="eyebrow">Incoming</span>
              <h3 className="text-lg fw-800">Invites for your account</h3>
              {invites.incoming.length > 0 ? (
                <div className="stack" style={{ gap: 10 }}>
                  {invites.incoming.map((invite) => (
                    <div key={invite.id} className="teams-cardRow">
                      <div className="stack" style={{ gap: 4 }}>
                        <strong>{invite.teamName}</strong>
                        <span className="muted text-sm">
                          {invite.role} · expires {formatTime(invite.expiresAt)}
                        </span>
                      </div>
                      <a className="btn" href={`/teams/invites/${invite.id}`}>
                        Review
                      </a>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="teams-empty">No pending invites for your email.</div>
              )}
            </div>

            <div className="block stack teams-panel">
              <span className="eyebrow">Sent</span>
              <h3 className="text-lg fw-800">Outstanding invite links</h3>
              {invites.sent.length > 0 ? (
                <div className="stack" style={{ gap: 10 }}>
                  {invites.sent.map((invite) => (
                    <div key={invite.id} className="teams-cardRow">
                      <div className="stack" style={{ gap: 4 }}>
                        <strong>{invite.email}</strong>
                        <span className="muted text-sm">
                          {invite.teamName} · {invite.role}
                        </span>
                      </div>
                      <div className="row">
                        {invite.shareUrl ? (
                          <button
                            className="btn"
                            type="button"
                            onClick={() => void copyText(invite.shareUrl || "", "Invite link copied.")}
                          >
                            Copy link
                          </button>
                        ) : null}
                        <button className="btn" type="button" onClick={() => void handleRevokeInvite(invite.id)}>
                          Revoke
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="teams-empty">No outstanding invites created by you.</div>
              )}
            </div>
          </div>
        ) : null}
      </div>
    </section>
  );
}
