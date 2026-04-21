"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

type InviteDetail = {
  id: string;
  teamId: string;
  teamName: string;
  teamSlug: string;
  email: string;
  role: string;
  status: string;
  expiresAt: number;
  createdAt: number;
};

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

export default function InviteAcceptView({
  inviteId,
  token,
}: {
  inviteId: string;
  token?: string | null;
}) {
  const router = useRouter();
  const [invite, setInvite] = useState<InviteDetail | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadInvite() {
      try {
        setLoading(true);
        const query = token ? `?token=${encodeURIComponent(token)}` : "";
        const nextInvite = await apiRequest<InviteDetail>(`/api/teams/invites/${inviteId}${query}`);
        if (!cancelled) {
          setInvite(nextInvite);
          setError("");
        }
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : "invite_load_failed");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadInvite();
    return () => {
      cancelled = true;
    };
  }, [inviteId, token]);

  async function handleAccept() {
    try {
      setSubmitting(true);
      setError("");
      await apiRequest(`/api/teams/invites/${inviteId}/accept`, {
        method: "POST",
        body: JSON.stringify(token ? { token } : {}),
      });
      router.push("/teams");
      router.refresh();
    } catch (acceptError) {
      setError(acceptError instanceof Error ? acceptError.message : "accept_invite_failed");
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) {
    return (
      <div className="block stack" style={{ maxWidth: 640, margin: "0 auto" }}>
        <span className="eyebrow">Invite</span>
        <h1 className="text-xl fw-800">Loading invitation details.</h1>
      </div>
    );
  }

  return (
    <div className="block stack" style={{ maxWidth: 720, margin: "0 auto" }}>
      <span className="eyebrow">Team invitation</span>
      <h1 className="text-xl fw-800">
        {invite ? `Join ${invite.teamName}` : "Invite unavailable"}
      </h1>
      {invite ? (
        <>
          <p className="muted text-sm" style={{ lineHeight: 1.6 }}>
            You&apos;re about to join <strong>{invite.teamName}</strong> as <strong>{invite.role}</strong>.
            After accepting, this team will show up in the browser dashboard and in the TUI after your next refresh.
          </p>
          <div className="stack" style={{ gap: 8 }}>
            <div className="teams-stat">
              <strong>team</strong>
              <span>{invite.teamSlug}</span>
            </div>
            <div className="teams-stat">
              <strong>email</strong>
              <span>{invite.email}</span>
            </div>
            <div className="teams-stat">
              <strong>expires</strong>
              <span>{new Date(invite.expiresAt).toLocaleString()}</span>
            </div>
          </div>
          <div className="row">
            <button className="btn btn--primary" type="button" onClick={() => void handleAccept()} disabled={submitting}>
              {submitting ? "Accepting…" : "Accept invitation"}
            </button>
            <Link className="btn" href="/teams">
              Back to teams
            </Link>
          </div>
        </>
      ) : (
        <>
          <p className="muted text-sm">This invite is no longer available.</p>
          <Link className="btn" href="/teams">
            Back to teams
          </Link>
        </>
      )}
      {error ? <div className="teams-notice teams-notice--error">{error}</div> : null}
    </div>
  );
}
