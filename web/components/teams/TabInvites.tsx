"use client";

import Link from "next/link";

import DropdownMenu from "../ui/DropdownMenu";
import { formatTime } from "./utils";
import type { InviteResponse, TeamSummary } from "./types";

type TabInvitesProps = {
  selectedTeam: TeamSummary | null;
  invites: InviteResponse;
  canInvite: boolean;
  onInvite: () => void;
  onCopyLink: (shareUrl: string, email: string) => void;
  onRevoke: (inviteId: string) => void;
};

export default function TabInvites({
  selectedTeam,
  invites,
  canInvite,
  onInvite,
  onCopyLink,
  onRevoke,
}: TabInvitesProps) {
  return (
    <>
      <div className="tab-toolbar">
        <h2 className="tab-toolbar__title">Invites</h2>
        <div className="tab-toolbar__actions">
          {canInvite && selectedTeam ? (
            <button type="button" className="btn btn--primary" onClick={onInvite}>
              + New invite
            </button>
          ) : null}
        </div>
      </div>

      <div className="tab-content">
        <div className="tab-content__section-title">
          Incoming ({invites.incoming.length})
        </div>
        {invites.incoming.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state__title">No pending invites</div>
            <p className="empty-state__body">
              Team invites sent to your email will show up here.
            </p>
          </div>
        ) : (
          <div>
            {invites.incoming.map((invite) => (
              <div key={invite.id} className="data-row">
                <div className="data-row__primary">
                  <span className="data-row__title">{invite.teamName}</span>
                  <span className="data-row__meta">
                    {invite.role} · expires {formatTime(invite.expiresAt)}
                  </span>
                </div>
                <div className="data-row__trail">
                  <Link className="btn" href={`/teams/invites/${invite.id}`}>
                    Review
                  </Link>
                </div>
              </div>
            ))}
          </div>
        )}

        <div className="tab-content__section-title">
          Sent ({invites.sent.length})
        </div>
        {invites.sent.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state__title">No outstanding invites</div>
            <p className="empty-state__body">
              Invites you create will show up here until they&apos;re accepted or
              revoked.
            </p>
            {canInvite && selectedTeam ? (
              <button
                type="button"
                className="btn btn--primary"
                onClick={onInvite}
              >
                Send an invite
              </button>
            ) : null}
          </div>
        ) : (
          <div>
            {invites.sent.map((invite) => (
              <div key={invite.id} className="data-row">
                <div className="data-row__primary">
                  <span className="data-row__title">{invite.email}</span>
                  <span className="data-row__meta">
                    {invite.teamName} · {invite.role} · expires{" "}
                    {formatTime(invite.expiresAt)}
                  </span>
                </div>
                <div className="data-row__trail">
                  <DropdownMenu
                    align="end"
                    triggerAriaLabel={`Actions for invite to ${invite.email}`}
                    triggerClassName="row-menu"
                    trigger={<span aria-hidden="true">⋯</span>}
                  >
                    {invite.shareUrl ? (
                      <DropdownMenu.Item
                        onSelect={() =>
                          onCopyLink(invite.shareUrl ?? "", invite.email)
                        }
                      >
                        Copy invite link
                      </DropdownMenu.Item>
                    ) : null}
                    <DropdownMenu.Separator />
                    <DropdownMenu.Item
                      onSelect={() => onRevoke(invite.id)}
                      variant="danger"
                    >
                      Revoke invite…
                    </DropdownMenu.Item>
                  </DropdownMenu>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  );
}
