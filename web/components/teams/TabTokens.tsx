"use client";

import DropdownMenu from "../ui/DropdownMenu";
import { formatTime } from "./utils";
import type { TeamAutomationToken, TeamHost, TeamSummary } from "./types";

type TabTokensProps = {
  selectedTeam: TeamSummary | null;
  tokens: TeamAutomationToken[];
  hosts: TeamHost[];
  canManage: boolean;
  onCreate: () => void;
  onRevoke: (tokenId: string, name: string) => void;
  onDelete: (tokenId: string, name: string) => void;
};

export default function TabTokens({
  selectedTeam,
  tokens,
  hosts,
  canManage,
  onCreate,
  onRevoke,
  onDelete,
}: TabTokensProps) {
  return (
    <>
      <div className="tab-toolbar">
        <h2 className="tab-toolbar__title">Automation Tokens</h2>
        <div className="tab-toolbar__actions">
          {canManage && selectedTeam && hosts.length > 0 ? (
            <button type="button" className="btn btn--primary" onClick={onCreate}>
              + New token
            </button>
          ) : null}
        </div>
      </div>

      <div className="tab-content">
        {!selectedTeam ? (
          <div className="empty-state">
            <div className="empty-state__title">No team selected</div>
            <p className="empty-state__body">Pick a team from the switcher above.</p>
          </div>
        ) : !canManage ? (
          <div className="empty-state">
            <div className="empty-state__title">Admin only</div>
            <p className="empty-state__body">
              Owners and admins can create and revoke AI-agent automation tokens.
            </p>
          </div>
        ) : tokens.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state__title">No team tokens yet</div>
            <p className="empty-state__body">
              Team tokens let agents run commands through SSHThing with every command logged.
            </p>
            {hosts.length > 0 ? (
              <button type="button" className="btn btn--primary" onClick={onCreate}>
                Create first token
              </button>
            ) : null}
          </div>
        ) : (
          <div>
            {tokens.map((token) => (
              <div key={token.id} className="data-row">
                <div className="data-row__primary">
                  <span className="data-row__title">{token.name}</span>
                  <span className="data-row__meta">
                    {token.hostCount} {token.hostCount === 1 ? "host" : "hosts"}
                    {" · "}created by {token.createdByDisplayName || "unknown"}
                    {" · "}used {token.useCount} times
                    {token.lastUsedAt ? ` · last ${formatTime(token.lastUsedAt)}` : ""}
                  </span>
                  <span className="data-row__chips">
                    <span className={token.status === "active" ? "chip" : "chip chip--danger"}>
                      {token.status}
                    </span>
                    {token.hosts?.slice(0, 3).map((host) => (
                      <span key={host.hostId} className="chip chip--muted">
                        {host.hostLabel}
                      </span>
                    ))}
                  </span>
                </div>
                <div className="data-row__trail">
                  <DropdownMenu
                    align="end"
                    triggerAriaLabel={`Actions for ${token.name}`}
                    triggerClassName="row-menu"
                    trigger={<span aria-hidden="true">⋯</span>}
                  >
                    <DropdownMenu.Item
                      onSelect={() => onRevoke(token.id, token.name)}
                      disabled={token.status !== "active"}
                      variant="danger"
                    >
                      Revoke token…
                    </DropdownMenu.Item>
                    <DropdownMenu.Item
                      onSelect={() => onDelete(token.id, token.name)}
                      disabled={token.status === "active"}
                      variant="danger"
                    >
                      Delete revoked token…
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
