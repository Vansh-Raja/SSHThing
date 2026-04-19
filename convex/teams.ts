import { mutation, query, type MutationCtx, type QueryCtx } from "./_generated/server";
import { v } from "convex/values";

function normalizeSlug(raw: string, fallback: string): string {
  const slug = raw
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48);
  return slug || fallback.toLowerCase();
}

async function requireOwnedTeam(
  ctx: QueryCtx | MutationCtx,
  teamId: string,
  clerkUserId: string,
) {
  const team = await ctx.db.get(teamId);
  if (!team || team.ownerClerkUserId !== clerkUserId || team.status !== "active") {
    throw new Error("team_not_found");
  }
  return team;
}

export const listForUser = query({
  args: {
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const teams = await ctx.db
      .query("teams")
      .withIndex("by_owner_and_display_order", (q) => q.eq("ownerClerkUserId", args.clerkUserId))
      .collect();

    return teams
      .filter((team) => team.status === "active")
      .map((team) => ({
        id: team._id,
        name: team.name,
        slug: team.slug,
        displayOrder: team.displayOrder,
      }));
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
    const team = await requireOwnedTeam(ctx, args.teamId, args.clerkUserId);
    const name = args.name.trim();
    if (!name) {
      throw new Error("team_name_required");
    }
    const slug = normalizeSlug(name, team.slug);
    await ctx.db.patch(args.teamId, {
      name,
      slug,
      updatedAt: Date.now(),
    });
    return {
      id: args.teamId,
      name,
      slug,
      displayOrder: team.displayOrder,
    };
  },
});

export const remove = mutation({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const team = await requireOwnedTeam(ctx, args.teamId, args.clerkUserId);
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
      const team = await requireOwnedTeam(ctx, args.teamIds[i], args.clerkUserId);
      await ctx.db.patch(team._id, {
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
    const team = await requireOwnedTeam(ctx, args.teamId, args.clerkUserId);
    return {
      id: team._id,
      name: team.name,
      slug: team.slug,
      displayOrder: team.displayOrder,
    };
  },
});

export const listHosts = query({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    await requireOwnedTeam(ctx, args.teamId, args.clerkUserId);
    const hosts = await ctx.db
      .query("teamHosts")
      .withIndex("by_team", (q) => q.eq("teamId", args.teamId))
      .collect();

    return hosts.map((host) => ({
      id: host._id,
      teamId: host.teamId,
      label: host.label,
      hostname: host.hostname,
      username: host.username,
      port: host.port,
      group: host.group,
      tags: host.tags,
      authMode: host.authMode ?? "",
      lastConnectedAt: host.lastConnectedAt ?? null,
    }));
  },
});
