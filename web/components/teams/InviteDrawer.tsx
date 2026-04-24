"use client";

import { useEffect, useState, type FormEvent } from "react";

import Drawer from "../ui/Drawer";
import { toast } from "../ui/toast";
import { apiRequest, errorMessage } from "./api";
import type { TeamInvite, TeamRole } from "./types";

type InviteDrawerProps = {
  open: boolean;
  teamId: string;
  teamName: string;
  onClose: () => void;
  onSaved: () => void;
};

async function copyText(value: string, successMessage: string) {
  try {
    await navigator.clipboard.writeText(value);
    toast.success(successMessage);
  } catch {
    toast.error("Couldn't copy to clipboard.");
  }
}

export default function InviteDrawer({
  open,
  teamId,
  teamName,
  onClose,
  onSaved,
}: InviteDrawerProps) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<TeamRole>("member");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      setEmail("");
      setRole("member");
      setSubmitting(false);
    }
  }, [open]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = email.trim();
    if (!teamId || !trimmed) return;
    try {
      setSubmitting(true);
      const invite = await apiRequest<TeamInvite>(
        `/api/teams/${teamId}/invites`,
        {
          method: "POST",
          body: JSON.stringify({ email: trimmed, role }),
        },
      );
      onSaved();
      if (invite.shareUrl) {
        await copyText(invite.shareUrl, `Copied invite link for ${invite.email}.`);
      } else {
        toast.success(`Created invite for ${invite.email}.`);
      }
      onClose();
    } catch (err) {
      toast.error(errorMessage(err, "invite_failed"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={`Invite to ${teamName}`}
      width={440}
      footer={
        <>
          <button
            type="button"
            className="btn"
            onClick={onClose}
            disabled={submitting}
          >
            Cancel
          </button>
          <button
            type="submit"
            form="invite-form"
            className="btn btn--primary"
            disabled={submitting || !email.trim()}
          >
            {submitting ? "Sending…" : "Send invite"}
          </button>
        </>
      }
    >
      <form
        id="invite-form"
        className="stack"
        style={{ gap: 14 }}
        onSubmit={handleSubmit}
      >
        <label className="field">
          <span className="field__label">Email</span>
          <input
            className="field__input"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="ops@example.com"
            required
          />
        </label>
        <label className="field">
          <span className="field__label">Role</span>
          <select
            className="field__input"
            value={role}
            onChange={(e) => setRole(e.target.value as TeamRole)}
          >
            <option value="member">member</option>
            <option value="admin">admin</option>
          </select>
        </label>
        <p className="muted" style={{ fontSize: 12, lineHeight: 1.55, margin: 0 }}>
          An invite link will be generated and copied to your clipboard if the
          recipient doesn&apos;t already have a Clerk account.
        </p>
      </form>
    </Drawer>
  );
}
