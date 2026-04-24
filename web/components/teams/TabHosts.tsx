"use client";

import { useMemo, useState } from "react";

import DropdownMenu from "../ui/DropdownMenu";
import type { TeamHost, TeamSummary } from "./types";

type TabHostsProps = {
  selectedTeam: TeamSummary | null;
  hosts: TeamHost[];
  canManage: boolean;
  activeHostId?: string;
  onNewHost: () => void;
  onEditHost: (hostId: string) => void;
  onDeleteHost: (hostId: string, label: string) => void;
};

function credentialTypeLabel(type: TeamHost["credentialType"]): string {
  if (type === "password") return "PASSWORD";
  if (type === "private_key") return "KEY";
  return "NONE";
}

function credentialModeLabel(mode: TeamHost["credentialMode"]): string {
  return mode === "shared" ? "SHARED" : "PER-MEMBER";
}

export default function TabHosts({
  selectedTeam,
  hosts,
  canManage,
  activeHostId,
  onNewHost,
  onEditHost,
  onDeleteHost,
}: TabHostsProps) {
  const [query, setQuery] = useState("");

  const filteredHosts = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return hosts;
    return hosts.filter((host) => {
      const haystack = `${host.label} ${host.hostname} ${host.username} ${host.group} ${host.tags.join(" ")}`.toLowerCase();
      return haystack.includes(q);
    });
  }, [hosts, query]);

  return (
    <>
      <div className="tab-toolbar">
        <h2 className="tab-toolbar__title">Hosts</h2>
        {selectedTeam && hosts.length > 0 ? (
          <input
            className="tab-toolbar__search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search label, hostname, tag…"
            aria-label="Search hosts"
          />
        ) : null}
        <div className="tab-toolbar__actions">
          {canManage && selectedTeam ? (
            <button type="button" className="btn btn--primary" onClick={onNewHost}>
              + New host
            </button>
          ) : null}
        </div>
      </div>

      <div className="tab-content">
        {!selectedTeam ? (
          <div className="empty-state">
            <div className="empty-state__title">No team selected</div>
            <p className="empty-state__body">
              Pick a team from the switcher above.
            </p>
          </div>
        ) : hosts.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state__title">No hosts yet</div>
            <p className="empty-state__body">
              Add a host to share it with your team. Credentials are encrypted at
              rest and never leave Convex unless an authorized member reveals them.
            </p>
            {canManage ? (
              <button type="button" className="btn btn--primary" onClick={onNewHost}>
                Create first host
              </button>
            ) : null}
          </div>
        ) : filteredHosts.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state__title">No matches</div>
            <p className="empty-state__body">
              No hosts match &ldquo;{query}&rdquo;.
            </p>
          </div>
        ) : (
          <div>
            {filteredHosts.map((host) => {
              const handleRowKey = (e: React.KeyboardEvent<HTMLDivElement>) => {
                if (e.target !== e.currentTarget) return;
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onEditHost(host.id);
                }
              };
              return (
                <div
                  key={host.id}
                  role="button"
                  tabIndex={0}
                  className={`data-row data-row--clickable${
                    host.id === activeHostId ? " data-row--active" : ""
                  }`}
                  onClick={(e) => {
                    if (e.target !== e.currentTarget) {
                      // Only trigger edit when the row chrome is clicked — not
                      // when the click originated on an inner button (⋯ menu).
                      const target = e.target as HTMLElement;
                      if (target.closest(".row-menu, .dropdown-menu")) return;
                    }
                    onEditHost(host.id);
                  }}
                  onKeyDown={handleRowKey}
                >
                  <span className="data-row__primary">
                    <span className="data-row__title">
                      {host.label || host.hostname}
                    </span>
                    <span className="data-row__meta">
                      {host.username ? `${host.username}@` : ""}
                      {host.hostname}:{host.port}
                      {host.group ? ` · ${host.group}` : ""}
                    </span>
                    <span className="data-row__chips">
                      <span className="chip">
                        {credentialModeLabel(host.credentialMode)}
                      </span>
                      <span className="chip chip--muted">
                        {credentialTypeLabel(host.credentialType)}
                      </span>
                      {host.tags.slice(0, 3).map((tag) => (
                        <span key={tag} className="chip chip--muted">
                          {tag}
                        </span>
                      ))}
                    </span>
                  </span>
                  <span className="data-row__trail">
                    {canManage ? (
                      <DropdownMenu
                        align="end"
                        triggerAriaLabel={`Actions for ${host.label || host.hostname}`}
                        triggerClassName="row-menu"
                        trigger={<span aria-hidden="true">⋯</span>}
                      >
                        <DropdownMenu.Item onSelect={() => onEditHost(host.id)}>
                          Edit host…
                        </DropdownMenu.Item>
                        <DropdownMenu.Separator />
                        <DropdownMenu.Item
                          onSelect={() =>
                            onDeleteHost(host.id, host.label || host.hostname)
                          }
                          variant="danger"
                        >
                          Delete host…
                        </DropdownMenu.Item>
                      </DropdownMenu>
                    ) : null}
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </>
  );
}
