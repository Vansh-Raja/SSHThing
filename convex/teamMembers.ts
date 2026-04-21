import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

import { normalizeTeamRole, requireTeamPermission } from "./teamAccess";

export const listForTeam = query({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "read");

    const members = await ctx.db
      .query("teamMembers")
      .withIndex("by_team", (q) => q.eq("teamId", args.teamId))
      .collect();

    return members
      .filter((member) => member.status === "active")
      .map((member) => ({
        id: member._id,
        teamId: member.teamId,
        clerkUserId: member.clerkUserId,
        email: member.email,
        displayName: member.displayName,
        role: normalizeTeamRole(member.role),
        status: member.status,
        joinedAt: member.joinedAt ?? null,
      }));
  },
});

export const updateRole = mutation({
  args: {
    teamId: v.id("teams"),
    memberId: v.id("teamMembers"),
    clerkUserId: v.string(),
    role: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_members");
    const member = await ctx.db.get(args.memberId);
    if (!member || member.teamId !== args.teamId || member.status !== "active") {
      throw new Error("member_not_found");
    }
    if (member.clerkUserId === access.team.ownerClerkUserId) {
      throw new Error("cannot_change_owner_role");
    }

    const role = normalizeTeamRole(args.role);
    if (role === "owner") {
      throw new Error("cannot_promote_to_owner");
    }

    await ctx.db.patch(args.memberId, {
      role,
      updatedAt: Date.now(),
    });

    return {
      id: member._id,
      role,
      status: "active",
    };
  },
});

export const remove = mutation({
  args: {
    teamId: v.id("teams"),
    memberId: v.id("teamMembers"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_members");
    const member = await ctx.db.get(args.memberId);
    if (!member || member.teamId !== args.teamId || member.status !== "active") {
      throw new Error("member_not_found");
    }
    if (member.clerkUserId === access.team.ownerClerkUserId) {
      throw new Error("cannot_remove_owner");
    }

    await ctx.db.patch(args.memberId, {
      status: "removed",
      updatedAt: Date.now(),
    });

    return { ok: true };
  },
});
