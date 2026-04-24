"use client";

import { useEffect, useState, type FormEvent } from "react";

import Drawer from "../ui/Drawer";
import { confirmDialog } from "../ui/dialogs";
import { toast } from "../ui/toast";
import { apiRequest, errorMessage } from "./api";
import { blankHostForm, blankPersonalCredentialForm } from "./forms";
import { formatTime, parseTags } from "./utils";
import type {
  CredentialRosterEntry,
  HostFormState,
  PersonalCredential,
  PersonalCredentialFormState,
  RevealedCredential,
  TeamHostDetail,
} from "./types";

type HostDrawerProps = {
  open: boolean;
  teamId: string;
  mode: "create" | "edit";
  hostId?: string;
  hostLabel?: string;
  canManageHosts: boolean;
  canRevealSecrets: boolean;
  onClose: () => void;
  onChanged: () => void;
  onAuditChanged?: () => void;
};

type Segment = "details" | "credentials";

async function copyText(value: string, successMessage: string) {
  try {
    await navigator.clipboard.writeText(value);
    toast.success(successMessage);
  } catch {
    toast.error("Couldn't copy to clipboard.");
  }
}

export default function HostDrawer({
  open,
  teamId,
  mode,
  hostId,
  hostLabel,
  canManageHosts,
  canRevealSecrets,
  onClose,
  onChanged,
  onAuditChanged,
}: HostDrawerProps) {
  const [segment, setSegment] = useState<Segment>("details");
  const [hostForm, setHostForm] = useState<HostFormState>(blankHostForm);
  const [saving, setSaving] = useState(false);
  const [personalCredential, setPersonalCredential] =
    useState<PersonalCredential | null>(null);
  const [personalForm, setPersonalForm] =
    useState<PersonalCredentialFormState>(blankPersonalCredentialForm);
  const [roster, setRoster] = useState<CredentialRosterEntry[]>([]);
  const [revealed, setRevealed] = useState<RevealedCredential | null>(null);
  const [showRevealedSecret, setShowRevealedSecret] = useState(false);

  // Reset state when drawer opens or mode changes.
  useEffect(() => {
    if (!open) return;
    setSegment("details");
    setRevealed(null);
    setShowRevealedSecret(false);
    if (mode === "create") {
      setHostForm(blankHostForm);
      setPersonalCredential(null);
      setPersonalForm(blankPersonalCredentialForm);
      setRoster([]);
    }
  }, [open, mode]);

  // When editing, fetch host detail. Blank the form first so we don't briefly
  // show a previous host's values while the fetch is in flight (e.g. when the
  // drawer is already open and the user clicks a different host row).
  useEffect(() => {
    if (!open || mode !== "edit" || !hostId) return;
    setHostForm(blankHostForm);
    let cancelled = false;
    (async () => {
      try {
        const host = await apiRequest<TeamHostDetail>(`/api/teams/hosts/${hostId}`);
        if (cancelled) return;
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
      } catch (err) {
        if (!cancelled) toast.error(errorMessage(err, "host_load_failed"));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, mode, hostId]);

  // Load personal credential when in per-member edit mode.
  useEffect(() => {
    if (
      !open ||
      mode !== "edit" ||
      !hostId ||
      hostForm.credentialMode !== "per_member"
    ) {
      setPersonalCredential(null);
      setPersonalForm(blankPersonalCredentialForm);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const cred = await apiRequest<PersonalCredential>(
          `/api/teams/hosts/${hostId}/my-credential`,
        );
        if (cancelled) return;
        setPersonalCredential(cred);
        setPersonalForm({
          username: cred.username ?? "",
          credentialType:
            cred.credentialType === "private_key" ? "private_key" : "password",
          secret: cred.secret ?? "",
        });
      } catch (err) {
        if (!cancelled) toast.error(errorMessage(err, "credential_load_failed"));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, mode, hostId, hostForm.credentialMode]);

  // Load member credential roster for admins.
  useEffect(() => {
    if (
      !open ||
      mode !== "edit" ||
      !hostId ||
      hostForm.credentialMode !== "per_member" ||
      !canManageHosts
    ) {
      setRoster([]);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const entries = await apiRequest<CredentialRosterEntry[]>(
          `/api/teams/hosts/${hostId}/credentials`,
        );
        if (!cancelled) setRoster(entries);
      } catch (err) {
        if (!cancelled) toast.error(errorMessage(err, "credential_roster_failed"));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, mode, hostId, hostForm.credentialMode, canManageHosts]);

  async function handleSaveHost(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!teamId) return;
    try {
      setSaving(true);
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
      if (mode === "edit" && hostId) {
        await apiRequest(`/api/teams/hosts/${hostId}`, {
          method: "PATCH",
          body: JSON.stringify(payload),
        });
        toast.success("Host updated.");
      } else {
        await apiRequest(`/api/teams/${teamId}/hosts`, {
          method: "POST",
          body: JSON.stringify(payload),
        });
        toast.success("Host created.");
      }
      onChanged();
      onClose();
    } catch (err) {
      toast.error(errorMessage(err, "save_host_failed"));
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteHost() {
    if (!hostId) return;
    const ok = await confirmDialog({
      title: "Delete host",
      message: `Delete ${hostForm.label || hostForm.hostname}? Member credentials for this host will also be deleted.`,
      variant: "danger",
      confirmLabel: "Delete host",
    });
    if (!ok) return;
    try {
      setSaving(true);
      await apiRequest(`/api/teams/hosts/${hostId}`, { method: "DELETE" });
      toast.success("Host deleted.");
      onChanged();
      onClose();
    } catch (err) {
      toast.error(errorMessage(err, "delete_host_failed"));
    } finally {
      setSaving(false);
    }
  }

  async function handleSavePersonal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!hostId) return;
    try {
      await apiRequest(`/api/teams/hosts/${hostId}/my-credential`, {
        method: "PUT",
        body: JSON.stringify(personalForm),
      });
      const next = await apiRequest<PersonalCredential>(
        `/api/teams/hosts/${hostId}/my-credential`,
      );
      setPersonalCredential(next);
      setPersonalForm({
        username: next.username ?? "",
        credentialType:
          next.credentialType === "private_key" ? "private_key" : "password",
        secret: next.secret ?? "",
      });
      if (canManageHosts) {
        const entries = await apiRequest<CredentialRosterEntry[]>(
          `/api/teams/hosts/${hostId}/credentials`,
        );
        setRoster(entries);
      }
      toast.success("Personal credential saved.");
    } catch (err) {
      toast.error(errorMessage(err, "save_credential_failed"));
    }
  }

  async function handleDeletePersonal() {
    if (!hostId) return;
    const ok = await confirmDialog({
      title: "Delete your credential",
      message: "Your saved credential for this host will be removed.",
      variant: "danger",
      confirmLabel: "Delete",
    });
    if (!ok) return;
    try {
      await apiRequest(`/api/teams/hosts/${hostId}/my-credential`, {
        method: "DELETE",
      });
      setPersonalCredential({
        hostId,
        credentialMode: "per_member",
        credentialType: hostForm.credentialType,
        username: null,
        hasCredential: false,
        secret: "",
        updatedAt: null,
      });
      setPersonalForm(blankPersonalCredentialForm);
      if (canManageHosts) {
        const entries = await apiRequest<CredentialRosterEntry[]>(
          `/api/teams/hosts/${hostId}/credentials`,
        );
        setRoster(entries);
      }
      toast.success("Personal credential deleted.");
    } catch (err) {
      toast.error(errorMessage(err, "delete_credential_failed"));
    }
  }

  async function handleRevealShared() {
    if (!hostId) return;
    const ok = await confirmDialog({
      title: "Reveal shared credential",
      message:
        "This will decrypt the shared credential and record an audit entry visible to owners and admins.",
      confirmLabel: "Reveal (audited)",
    });
    if (!ok) return;
    try {
      const rev = await apiRequest<RevealedCredential>(
        `/api/teams/hosts/${hostId}/credentials/shared/reveal`,
        { method: "POST" },
      );
      setRevealed(rev);
      setShowRevealedSecret(false);
      setHostForm((cur) => ({ ...cur, sharedCredential: rev.secret }));
      toast.success("Shared credential revealed and logged.");
      onAuditChanged?.();
    } catch (err) {
      toast.error(errorMessage(err, "reveal_shared_credential_failed"));
    }
  }

  async function handleRevealMember(memberId: string) {
    if (!hostId) return;
    const ok = await confirmDialog({
      title: "Reveal member credential",
      message:
        "This will decrypt the member's credential and record an audit entry.",
      confirmLabel: "Reveal (audited)",
    });
    if (!ok) return;
    try {
      const rev = await apiRequest<RevealedCredential>(
        `/api/teams/hosts/${hostId}/credentials/${memberId}/reveal`,
        { method: "POST" },
      );
      setRevealed(rev);
      setShowRevealedSecret(false);
      toast.success("Member credential revealed and logged.");
      onAuditChanged?.();
    } catch (err) {
      toast.error(errorMessage(err, "reveal_member_credential_failed"));
    }
  }

  async function handleDeleteMember(memberId: string, displayName: string) {
    if (!hostId) return;
    const ok = await confirmDialog({
      title: "Delete member credential",
      message: `Delete ${displayName}'s credential? This action is audited.`,
      variant: "danger",
      confirmLabel: "Delete",
    });
    if (!ok) return;
    try {
      await apiRequest(
        `/api/teams/hosts/${hostId}/credentials/${memberId}`,
        { method: "DELETE" },
      );
      setRevealed(null);
      const entries = await apiRequest<CredentialRosterEntry[]>(
        `/api/teams/hosts/${hostId}/credentials`,
      );
      setRoster(entries);
      toast.success(`Deleted ${displayName}'s credential.`);
      onAuditChanged?.();
    } catch (err) {
      toast.error(errorMessage(err, "delete_member_credential_failed"));
    }
  }

  const title =
    mode === "create"
      ? "New host"
      : hostForm.label || hostForm.hostname || hostLabel || "Edit host";

  const hasCredentialsSegment = mode === "edit";

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={title}
      width={560}
      footer={
        segment === "details" ? (
          <>
            {mode === "edit" && canManageHosts ? (
              <div className="drawer__footer-left">
                <button
                  type="button"
                  className="btn btn--danger"
                  onClick={handleDeleteHost}
                  disabled={saving}
                >
                  Delete
                </button>
              </div>
            ) : null}
            <button
              type="button"
              className="btn"
              onClick={onClose}
              disabled={saving}
            >
              Cancel
            </button>
            {canManageHosts ? (
              <button
                type="submit"
                form="host-form"
                className="btn btn--primary"
                disabled={saving}
              >
                {saving
                  ? "Saving…"
                  : mode === "create"
                    ? "Create host"
                    : "Save host"}
              </button>
            ) : null}
          </>
        ) : (
          <button type="button" className="btn" onClick={onClose}>
            Close
          </button>
        )
      }
    >
      {hasCredentialsSegment ? (
        <div className="segmented" role="tablist" aria-label="Host sections">
          <button
            type="button"
            role="tab"
            aria-selected={segment === "details"}
            className="segmented__item"
            onClick={() => setSegment("details")}
          >
            Details
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={segment === "credentials"}
            className="segmented__item"
            onClick={() => setSegment("credentials")}
          >
            Credentials
          </button>
        </div>
      ) : null}

      {segment === "details" ? (
        <form
          id="host-form"
          className="stack"
          style={{ gap: 12 }}
          onSubmit={handleSaveHost}
        >
          <label className="field">
            <span className="field__label">Label</span>
            <input
              className="field__input"
              value={hostForm.label}
              onChange={(e) =>
                setHostForm((cur) => ({ ...cur, label: e.target.value }))
              }
              placeholder="Production bastion"
              disabled={!canManageHosts}
            />
          </label>

          <div className="grid-2">
            <label className="field">
              <span className="field__label">Hostname</span>
              <input
                className="field__input"
                value={hostForm.hostname}
                onChange={(e) =>
                  setHostForm((cur) => ({ ...cur, hostname: e.target.value }))
                }
                placeholder="server.example.com"
                disabled={!canManageHosts}
                required
              />
            </label>
            <label className="field">
              <span className="field__label">Username</span>
              <input
                className="field__input"
                value={hostForm.username}
                onChange={(e) =>
                  setHostForm((cur) => ({ ...cur, username: e.target.value }))
                }
                placeholder="root"
                disabled={!canManageHosts}
              />
            </label>
          </div>

          <div className="grid-2">
            <label className="field">
              <span className="field__label">Port</span>
              <input
                className="field__input"
                value={hostForm.port}
                onChange={(e) =>
                  setHostForm((cur) => ({ ...cur, port: e.target.value }))
                }
                placeholder="22"
                disabled={!canManageHosts}
                inputMode="numeric"
              />
            </label>
            <label className="field">
              <span className="field__label">Group</span>
              <input
                className="field__input"
                value={hostForm.group}
                onChange={(e) =>
                  setHostForm((cur) => ({ ...cur, group: e.target.value }))
                }
                placeholder="prod"
                disabled={!canManageHosts}
              />
            </label>
          </div>

          <label className="field">
            <span className="field__label">Tags</span>
            <input
              className="field__input"
              value={hostForm.tags}
              onChange={(e) =>
                setHostForm((cur) => ({ ...cur, tags: e.target.value }))
              }
              placeholder="ssh, linux, us-east-1"
              disabled={!canManageHosts}
            />
          </label>

          <label className="field">
            <span className="field__label">Notes</span>
            <textarea
              className="field__input field__textarea"
              value={hostForm.notes}
              onChange={(e) =>
                setHostForm((cur) => ({ ...cur, notes: e.target.value }))
              }
              placeholder="Shared deployment notes, caveats, or runbook steps"
            />
          </label>

          <div className="grid-2">
            <label className="field">
              <span className="field__label">Credential mode</span>
              <select
                className="field__input"
                value={hostForm.credentialMode}
                onChange={(e) =>
                  setHostForm((cur) => ({
                    ...cur,
                    credentialMode: e.target.value as "shared" | "per_member",
                    sharedCredential:
                      e.target.value === "per_member"
                        ? ""
                        : cur.sharedCredential,
                  }))
                }
                disabled={!canManageHosts}
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
                onChange={(e) =>
                  setHostForm((cur) => ({
                    ...cur,
                    credentialType: e.target.value as
                      | "none"
                      | "password"
                      | "private_key",
                  }))
                }
                disabled={!canManageHosts}
              >
                <option value="none">none</option>
                <option value="password">password</option>
                <option value="private_key">private key</option>
              </select>
            </label>
          </div>

          {hostForm.credentialMode === "shared" &&
          hostForm.credentialType !== "none" ? (
            <label className="field">
              <span className="field__label">Shared secret</span>
              <textarea
                className="field__input field__textarea"
                value={hostForm.sharedCredential}
                onChange={(e) =>
                  setHostForm((cur) => ({
                    ...cur,
                    sharedCredential: e.target.value,
                  }))
                }
                placeholder={
                  hostForm.credentialType === "private_key"
                    ? "Paste the private key"
                    : "Paste the password"
                }
                disabled={!canManageHosts}
              />
              {mode === "edit" && canRevealSecrets ? (
                <span className="muted" style={{ fontSize: 12 }}>
                  Reveal is audited. Use the Credentials tab to pull the
                  currently-stored secret.
                </span>
              ) : null}
            </label>
          ) : null}
        </form>
      ) : (
        <div className="stack" style={{ gap: 14 }}>
          {hostForm.credentialMode === "shared" ? (
            <section className="stack" style={{ gap: 10 }}>
              <span className="eyebrow">Shared credential</span>
              {hostForm.credentialType === "none" ? (
                <p className="muted" style={{ fontSize: 13, margin: 0 }}>
                  This host has no credential configured. Switch to a password or
                  private key in Details to enable reveal.
                </p>
              ) : canRevealSecrets ? (
                <div className="row">
                  <button
                    type="button"
                    className="btn btn--primary"
                    onClick={handleRevealShared}
                  >
                    Reveal shared credential
                  </button>
                  <span className="muted" style={{ fontSize: 12 }}>
                    Audited action. Output appears below.
                  </span>
                </div>
              ) : (
                <p className="muted" style={{ fontSize: 13, margin: 0 }}>
                  Only owners and admins can reveal shared credentials.
                </p>
              )}
            </section>
          ) : (
            <>
              <section className="stack" style={{ gap: 10 }}>
                <span className="eyebrow">Your credential</span>
                <p className="muted" style={{ fontSize: 13, margin: 0 }}>
                  Per-member host. Your secret isn&apos;t visible to other members
                  via the self-service path.
                  {personalCredential?.updatedAt
                    ? ` · updated ${formatTime(personalCredential.updatedAt)}`
                    : ""}
                </p>
                <form
                  className="stack"
                  style={{ gap: 10 }}
                  onSubmit={handleSavePersonal}
                >
                  <label className="field">
                    <span className="field__label">Username override</span>
                    <input
                      className="field__input"
                      value={personalForm.username}
                      onChange={(e) =>
                        setPersonalForm((cur) => ({
                          ...cur,
                          username: e.target.value,
                        }))
                      }
                      placeholder="Optional"
                    />
                  </label>
                  <label className="field">
                    <span className="field__label">Credential type</span>
                    <select
                      className="field__input"
                      value={personalForm.credentialType}
                      onChange={(e) =>
                        setPersonalForm((cur) => ({
                          ...cur,
                          credentialType: e.target.value as
                            | "password"
                            | "private_key",
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
                      value={personalForm.secret}
                      onChange={(e) =>
                        setPersonalForm((cur) => ({
                          ...cur,
                          secret: e.target.value,
                        }))
                      }
                      placeholder={
                        personalForm.credentialType === "private_key"
                          ? "Paste your private key"
                          : "Paste your password"
                      }
                    />
                  </label>
                  <div className="row">
                    <button type="submit" className="btn btn--primary">
                      Save my credential
                    </button>
                    {personalCredential?.hasCredential ? (
                      <button
                        type="button"
                        className="btn"
                        onClick={handleDeletePersonal}
                      >
                        Delete mine
                      </button>
                    ) : null}
                  </div>
                </form>
              </section>

              {canManageHosts ? (
                <section className="stack" style={{ gap: 10 }}>
                  <span className="eyebrow">Member credentials</span>
                  <p className="muted" style={{ fontSize: 13, margin: 0 }}>
                    Reveal and delete actions are audited.
                  </p>
                  {roster.length === 0 ? (
                    <div className="empty-state">
                      <div className="empty-state__title">No data yet</div>
                      <p className="empty-state__body">
                        Member credential states will appear here once recorded.
                      </p>
                    </div>
                  ) : (
                    <div>
                      {roster.map((entry) => (
                        <div key={entry.memberId} className="data-row">
                          <div className="data-row__primary">
                            <span className="data-row__title">
                              {entry.displayName}
                            </span>
                            <span className="data-row__meta">
                              {entry.role} · {entry.email || entry.memberId}
                            </span>
                            <span className="data-row__meta">
                              {entry.hasCredential
                                ? `${entry.credentialType}${
                                    entry.username ? ` · ${entry.username}` : ""
                                  }${
                                    entry.updatedAt
                                      ? ` · ${formatTime(entry.updatedAt)}`
                                      : ""
                                  }`
                                : "no credential saved"}
                            </span>
                          </div>
                          <div className="data-row__trail">
                            <button
                              type="button"
                              className="btn"
                              onClick={() => handleRevealMember(entry.memberId)}
                              disabled={!entry.hasCredential}
                            >
                              Reveal
                            </button>
                            <button
                              type="button"
                              className="btn btn--danger"
                              onClick={() =>
                                handleDeleteMember(
                                  entry.memberId,
                                  entry.displayName,
                                )
                              }
                              disabled={!entry.hasCredential}
                            >
                              Delete
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </section>
              ) : null}
            </>
          )}

          {revealed ? (
            <section className="stack" style={{ gap: 10 }}>
              <span className="eyebrow">Revealed credential</span>
              <p className="muted" style={{ fontSize: 12, margin: 0 }}>
                {revealed.credentialType}
                {revealed.username ? ` · ${revealed.username}` : ""}
                {revealed.updatedAt
                  ? ` · ${formatTime(revealed.updatedAt)}`
                  : ""}
              </p>
              {showRevealedSecret ? (
                <textarea
                  className="field__input field__textarea"
                  value={revealed.secret}
                  readOnly
                />
              ) : (
                <div
                  className="field__input field__textarea"
                  style={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    color: "var(--muted)",
                    fontSize: 12,
                  }}
                >
                  Secret hidden.
                </div>
              )}
              <div className="row">
                <button
                  type="button"
                  className="btn btn--primary"
                  onClick={() =>
                    copyText(revealed.secret, "Credential copied.")
                  }
                >
                  Copy secret
                </button>
                <button
                  type="button"
                  className="btn"
                  onClick={() => setShowRevealedSecret((v) => !v)}
                >
                  {showRevealedSecret ? "Hide" : "Show"}
                </button>
                <button
                  type="button"
                  className="btn"
                  onClick={() => {
                    setRevealed(null);
                    setShowRevealedSecret(false);
                  }}
                >
                  Clear
                </button>
              </div>
            </section>
          ) : null}
        </div>
      )}
    </Drawer>
  );
}
