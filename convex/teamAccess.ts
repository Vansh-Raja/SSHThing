import type { Doc, Id } from "./_generated/dataModel";
import type { MutationCtx, QueryCtx } from "./_generated/server";

export type TeamRole = "owner" | "admin" | "member";
export type TeamPermission =
  | "read"
  | "edit_notes"
  | "manage_members"
  | "manage_hosts"
  | "manage_team"
  | "delete_team"
  | "reveal_secret";

type TeamCtx = QueryCtx | MutationCtx;

export function normalizeTeamRole(role: string): TeamRole {
  switch (role) {
    case "owner":
    case "admin":
    case "member":
      return role;
    default:
      return "member";
  }
}

export function hasTeamPermission(role: TeamRole, permission: TeamPermission): boolean {
  switch (permission) {
    case "read":
    case "edit_notes":
      return true;
    case "delete_team":
      return role === "owner";
    case "manage_members":
    case "manage_hosts":
    case "manage_team":
    case "reveal_secret":
      return role === "owner" || role === "admin";
    default:
      return false;
  }
}

export async function getTeamMembership(
  ctx: TeamCtx,
  teamId: Id<"teams">,
  clerkUserId: string,
): Promise<{ team: Doc<"teams">; member: Doc<"teamMembers"> | null; role: TeamRole } | null> {
  const team = await ctx.db.get(teamId);
  if (!team || team.status !== "active") {
    return null;
  }

  const member = await ctx.db
    .query("teamMembers")
    .withIndex("by_team_and_user", (q) => q.eq("teamId", teamId).eq("clerkUserId", clerkUserId))
    .first();

  if (team.ownerClerkUserId === clerkUserId) {
    return {
      team,
      member: member && member.status === "active" ? member : null,
      role: "owner",
    };
  }

  if (!member || member.status !== "active") {
    return null;
  }

  return {
    team,
    member,
    role: normalizeTeamRole(member.role),
  };
}

export async function requireTeamPermission(
  ctx: TeamCtx,
  teamId: Id<"teams">,
  clerkUserId: string,
  permission: TeamPermission,
): Promise<{ team: Doc<"teams">; member: Doc<"teamMembers"> | null; role: TeamRole }> {
  const access = await getTeamMembership(ctx, teamId, clerkUserId);
  if (!access) {
    throw new Error("team_not_found");
  }
  if (!hasTeamPermission(access.role, permission)) {
    throw new Error("forbidden");
  }
  return access;
}

export async function requireHostPermission(
  ctx: TeamCtx,
  hostId: Id<"teamHosts">,
  clerkUserId: string,
  permission: TeamPermission,
): Promise<{
  host: Doc<"teamHosts">;
  team: Doc<"teams">;
  member: Doc<"teamMembers"> | null;
  role: TeamRole;
}> {
  const host = await ctx.db.get(hostId);
  if (!host) {
    throw new Error("host_not_found");
  }

  const access = await requireTeamPermission(ctx, host.teamId, clerkUserId, permission);
  return {
    host,
    team: access.team,
    member: access.member,
    role: access.role,
  };
}
