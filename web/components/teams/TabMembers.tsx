"use client";

import DropdownMenu from "../ui/DropdownMenu";
import type { TeamMember, TeamRole, TeamSummary } from "./types";

type TabMembersProps = {
  selectedTeam: TeamSummary | null;
  members: TeamMember[];
  canManage: boolean;
  onInvite: () => void;
  onChangeRole: (memberId: string, role: TeamRole) => void;
  onRemove: (memberId: string, displayName: string) => void;
};

function roleChipClass(role: TeamRole) {
  if (role === "owner") return "chip chip--accent";
  if (role === "admin") return "chip";
  return "chip chip--muted";
}

export default function TabMembers({
  selectedTeam,
  members,
  canManage,
  onInvite,
  onChangeRole,
  onRemove,
}: TabMembersProps) {
  return (
    <>
      <div className="tab-toolbar">
        <h2 className="tab-toolbar__title">Members</h2>
        <div className="tab-toolbar__actions">
          {canManage && selectedTeam ? (
            <button type="button" className="btn btn--primary" onClick={onInvite}>
              + Invite member
            </button>
          ) : null}
        </div>
      </div>

      <div className="tab-content">
        {!selectedTeam ? (
          <div className="empty-state">
            <div className="empty-state__title">No team selected</div>
            <p className="empty-state__body">
              Pick a team from the switcher above, or create a new one.
            </p>
          </div>
        ) : members.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state__title">No members yet</div>
            <p className="empty-state__body">
              Invite a teammate to start sharing hosts.
            </p>
            {canManage ? (
              <button type="button" className="btn btn--primary" onClick={onInvite}>
                Invite first member
              </button>
            ) : null}
          </div>
        ) : (
          <div>
            {members.map((member) => {
              const canActOnMember = canManage && member.role !== "owner";
              return (
                <div key={member.id} className="data-row">
                  <div className="data-row__primary">
                    <span className="data-row__title">
                      {member.displayName || member.email}
                    </span>
                    <span className="data-row__meta">{member.email}</span>
                  </div>
                  <div className="data-row__trail">
                    <span className={roleChipClass(member.role)}>{member.role}</span>
                    {canActOnMember ? (
                      <DropdownMenu
                        align="end"
                        triggerAriaLabel={`Actions for ${member.displayName || member.email}`}
                        triggerClassName="row-menu"
                        trigger={<span aria-hidden="true">⋯</span>}
                      >
                        {member.role !== "admin" ? (
                          <DropdownMenu.Item
                            onSelect={() => onChangeRole(member.id, "admin")}
                          >
                            Promote to admin
                          </DropdownMenu.Item>
                        ) : null}
                        {member.role !== "member" ? (
                          <DropdownMenu.Item
                            onSelect={() => onChangeRole(member.id, "member")}
                          >
                            Demote to member
                          </DropdownMenu.Item>
                        ) : null}
                        <DropdownMenu.Separator />
                        <DropdownMenu.Item
                          onSelect={() =>
                            onRemove(member.id, member.displayName || member.email)
                          }
                          variant="danger"
                        >
                          Remove from team…
                        </DropdownMenu.Item>
                      </DropdownMenu>
                    ) : null}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </>
  );
}
