"use client";

import DropdownMenu from "../ui/DropdownMenu";
import type { TeamSummary } from "./types";

type TeamSwitcherProps = {
  teams: TeamSummary[];
  selectedTeamId: string;
  onSelect: (teamId: string) => void;
  onCreateRequested: () => void;
};

export default function TeamSwitcher({
  teams,
  selectedTeamId,
  onSelect,
  onCreateRequested,
}: TeamSwitcherProps) {
  const selected = teams.find((team) => team.id === selectedTeamId) ?? null;
  const label = selected?.name ?? "No team";

  return (
    <DropdownMenu
      align="start"
      triggerAriaLabel="Switch team"
      triggerClassName="team-bar__switcher"
      trigger={
        <>
          <span className="team-bar__switcher-name">{label}</span>
          <span className="team-bar__switcher-chevron" aria-hidden="true">
            ▼
          </span>
        </>
      }
    >
      {teams.length > 0 ? <DropdownMenu.Label>Teams</DropdownMenu.Label> : null}
      {teams.map((team) => (
        <DropdownMenu.Item
          key={team.id}
          onSelect={() => onSelect(team.id)}
          className={`dropdown-menu__item--team${
            team.id === selectedTeamId ? " dropdown-menu__item--active" : ""
          }`}
        >
          <span style={{ fontWeight: 700 }}>{team.name}</span>
          <span className="dropdown-menu__item-sub">
            {team.role} · {team.slug}
          </span>
        </DropdownMenu.Item>
      ))}
      {teams.length > 0 ? <DropdownMenu.Separator /> : null}
      <DropdownMenu.Item onSelect={onCreateRequested}>
        + New team…
      </DropdownMenu.Item>
    </DropdownMenu>
  );
}
