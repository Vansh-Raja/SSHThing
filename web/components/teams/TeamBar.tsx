"use client";

import DropdownMenu from "../ui/DropdownMenu";
import TeamSwitcher from "./TeamSwitcher";
import type { DashboardTab, TeamRole, TeamSummary } from "./types";

type Counts = {
  members: number;
  hosts: number;
  pendingInvites: number;
};

type TeamBarProps = {
  teams: TeamSummary[];
  selectedTeam: TeamSummary | null;
  counts: Counts;
  activeTab: DashboardTab;
  availableTabs: DashboardTab[];
  onTabChange: (tab: DashboardTab) => void;
  onSelectTeam: (teamId: string) => void;
  onCreateTeam: () => void;
  onRename: () => void;
  onDelete: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onRefresh: () => void;
  canMoveUp: boolean;
  canMoveDown: boolean;
};

const TAB_LABEL: Record<DashboardTab, string> = {
  members: "Members",
  hosts: "Hosts",
  invites: "Invites",
  tokens: "Tokens",
  audit: "Audit",
};

function roleChipVariant(role: TeamRole | undefined) {
  if (role === "owner") return "chip chip--accent";
  if (role === "admin") return "chip";
  return "chip chip--muted";
}

export default function TeamBar({
  teams,
  selectedTeam,
  counts,
  activeTab,
  availableTabs,
  onTabChange,
  onSelectTeam,
  onCreateTeam,
  onRename,
  onDelete,
  onMoveUp,
  onMoveDown,
  onRefresh,
  canMoveUp,
  canMoveDown,
}: TeamBarProps) {
  const canManage = selectedTeam
    ? selectedTeam.role === "owner" || selectedTeam.role === "admin"
    : false;
  const isOwner = selectedTeam?.role === "owner";

  return (
    <div className="team-bar">
      <div className="team-bar__row">
        <TeamSwitcher
          teams={teams}
          selectedTeamId={selectedTeam?.id ?? ""}
          onSelect={onSelectTeam}
          onCreateRequested={onCreateTeam}
        />

        <div className="team-bar__tabs" role="tablist" aria-label="Team sections">
          {availableTabs.map((tab) => (
            <button
              key={tab}
              type="button"
              role="tab"
              aria-selected={activeTab === tab}
              className="team-bar__tab"
              onClick={() => onTabChange(tab)}
            >
              {TAB_LABEL[tab]}
            </button>
          ))}
        </div>

        <div className="team-bar__trail">
          {selectedTeam ? (
            <DropdownMenu
              align="end"
              triggerAriaLabel="Team actions"
              triggerClassName="row-menu"
              trigger={<span aria-hidden="true">⋯</span>}
            >
              <DropdownMenu.Item onSelect={onRefresh}>Refresh</DropdownMenu.Item>
              <DropdownMenu.Item onSelect={onRename} disabled={!canManage}>
                Rename team…
              </DropdownMenu.Item>
              <DropdownMenu.Item
                onSelect={onMoveUp}
                disabled={!canManage || !canMoveUp}
              >
                Move up
              </DropdownMenu.Item>
              <DropdownMenu.Item
                onSelect={onMoveDown}
                disabled={!canManage || !canMoveDown}
              >
                Move down
              </DropdownMenu.Item>
              {isOwner ? (
                <>
                  <DropdownMenu.Separator />
                  <DropdownMenu.Item onSelect={onDelete} variant="danger">
                    Delete team…
                  </DropdownMenu.Item>
                </>
              ) : null}
            </DropdownMenu>
          ) : null}
        </div>
      </div>

      {selectedTeam ? (
        <div className="team-bar__meta">
          <span>
            <strong>{counts.members}</strong>{" "}
            {counts.members === 1 ? "member" : "members"}
          </span>
          <span aria-hidden="true">·</span>
          <span>
            <strong>{counts.hosts}</strong>{" "}
            {counts.hosts === 1 ? "host" : "hosts"}
          </span>
          <span aria-hidden="true">·</span>
          <span>
            <strong>{counts.pendingInvites}</strong> pending{" "}
            {counts.pendingInvites === 1 ? "invite" : "invites"}
          </span>
          <span aria-hidden="true">·</span>
          <span>
            your role:{" "}
            <span className={roleChipVariant(selectedTeam.role)}>
              {selectedTeam.role}
            </span>
          </span>
        </div>
      ) : (
        <div className="team-bar__meta">
          <span>No team selected yet. Create one or accept a pending invite.</span>
        </div>
      )}
    </div>
  );
}
