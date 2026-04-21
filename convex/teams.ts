import { mutation, query } from "./_generated/server";
import type { Id } from "./_generated/dataModel";
import { v } from "convex/values";

import { normalizeTeamRole, requireTeamPermission } from "./teamAccess";

function normalizeSlug(raw: string, fallback: string): string {
  const slug = raw
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48);
  return slug || fallback.toLowerCase();
}

export const listForUser = query({
  args: {
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const memberships = await ctx.db
      .query("teamMembers")
      .withIndex("by_user", (q) => q.eq("clerkUserId", args.clerkUserId))
      .collect();

    const teams = await Promise.all(
      memberships
        .filter((member) => member.status === "active")
        .map(async (member) => {
          const team = await ctx.db.get(member.teamId);
          if (!team || team.status !== "active") {
            return null;
          }
          return {
            id: team._id,
            name: team.name,
            slug: team.slug,
            displayOrder: team.displayOrder,
            role: team.ownerClerkUserId === args.clerkUserId ? "owner" : normalizeTeamRole(member.role),
          };
        }),
    );

    return teams.filter(Boolean).sort((a, b) => a.displayOrder - b.displayOrder);
  },
});

export const create = mutation({
  args: {
    clerkUserId: v.string(),
    userEmail: v.string(),
    displayName: v.string(),
    name: v.string(),
  },
  handler: async (ctx, args) => {
    const name = args.name.trim();
    if (!name) {
      throw new Error("team_name_required");
    }

    const existing = await ctx.db
      .query("teams")
      .withIndex("by_owner_and_display_order", (q) => q.eq("ownerClerkUserId", args.clerkUserId))
      .collect();

    const displayOrder = existing
      .filter((team) => team.status === "active")
      .reduce((maxOrder, team) => Math.max(maxOrder, team.displayOrder), -1) + 1;

    const now = Date.now();
    const teamId = await ctx.db.insert("teams", {
      ownerClerkUserId: args.clerkUserId,
      name,
      slug: normalizeSlug(name, `${args.clerkUserId.slice(0, 8)}-${displayOrder + 1}`),
      displayOrder,
      status: "active",
      createdAt: now,
      updatedAt: now,
    });

    await ctx.db.insert("teamMembers", {
      teamId,
      clerkUserId: args.clerkUserId,
      email: args.userEmail,
      displayName: args.displayName,
      role: "owner",
      status: "active",
      joinedAt: now,
      createdAt: now,
      updatedAt: now,
    });

    return {
      id: teamId,
      name,
      slug: normalizeSlug(name, `${args.clerkUserId.slice(0, 8)}-${displayOrder + 1}`),
      displayOrder,
      role: "owner",
    };
  },
});

export const rename = mutation({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
    name: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_team");
    const name = args.name.trim();
    if (!name) {
      throw new Error("team_name_required");
    }
    const slug = normalizeSlug(name, access.team.slug);
    await ctx.db.patch(args.teamId, {
      name,
      slug,
      updatedAt: Date.now(),
    });
    return {
      id: args.teamId,
      name,
      slug,
      displayOrder: access.team.displayOrder,
      role: access.role,
    };
  },
});

export const remove = mutation({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "delete_team");
    const team = access.team;
    const now = Date.now();
    await ctx.db.patch(args.teamId, {
      status: "deleted",
      updatedAt: now,
    });

    const members = await ctx.db
      .query("teamMembers")
      .withIndex("by_team", (q) => q.eq("teamId", team._id))
      .collect();
    for (const member of members) {
      await ctx.db.patch(member._id, {
        status: "deleted",
        updatedAt: now,
      });
    }

    const invites = await ctx.db
      .query("teamInvites")
      .withIndex("by_team", (q) => q.eq("teamId", team._id))
      .collect();
    for (const invite of invites) {
      await ctx.db.patch(invite._id, {
        status: invite.status === "accepted" ? invite.status : "revoked",
        updatedAt: now,
      });
    }

    const hosts = await ctx.db
      .query("teamHosts")
      .withIndex("by_team", (q) => q.eq("teamId", team._id))
      .collect();
    for (const host of hosts) {
      const shared = await ctx.db
        .query("teamHostSharedCredentials")
        .withIndex("by_host", (q) => q.eq("hostId", host._id))
        .first();
      if (shared) {
        await ctx.db.delete(shared._id);
      }
      const personal = await ctx.db
        .query("teamHostPersonalCredentials")
        .withIndex("by_host_and_user", (q) => q.eq("hostId", host._id))
        .collect();
      for (const credential of personal) {
        await ctx.db.delete(credential._id);
      }
      await ctx.db.delete(host._id);
    }

    return { ok: true };
  },
});

export const reorder = mutation({
  args: {
    clerkUserId: v.string(),
    teamIds: v.array(v.id("teams")),
  },
  handler: async (ctx, args) => {
    for (let i = 0; i < args.teamIds.length; i += 1) {
      const access = await requireTeamPermission(
        ctx,
        args.teamIds[i] as Id<"teams">,
        args.clerkUserId,
        "manage_team",
      );
      await ctx.db.patch(access.team._id, {
        displayOrder: i,
        updatedAt: Date.now(),
      });
    }
    return { ok: true };
  },
});

export const getSummary = query({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "read");
    return {
      id: access.team._id,
      name: access.team.name,
      slug: access.team.slug,
      displayOrder: access.team.displayOrder,
      role: access.role,
    };
  },
});
