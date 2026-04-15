import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

export const getCurrentTeam = query({
  args: {
    clerkUserId: v.string()
  },
  handler: async (ctx, args) => {
    const membership = await ctx.db
      .query("teamMembers")
      .withIndex("by_clerk_user", (q) => q.eq("clerkUserId", args.clerkUserId))
      .first();

    if (!membership) {
      return null;
    }

    const team = await ctx.db.get(membership.teamId);
    if (!team) {
      return null;
    }

    return {
      id: team._id,
      name: team.name,
      slug: team.slug,
      role: membership.role,
      memberStatus: membership.status
    };
  }
});

export const listHosts = query({
  args: {
    teamId: v.id("teams")
  },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("teamHosts")
      .withIndex("by_team", (q) => q.eq("teamId", args.teamId))
      .collect();
  }
});

export const listMembers = query({
  args: {
    teamId: v.id("teams")
  },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("teamMembers")
      .withIndex("by_team", (q) => q.eq("teamId", args.teamId))
      .collect();
  }
});

export const createTeam = mutation({
  args: {
    clerkOrganizationId: v.string(),
    createdByUserId: v.string(),
    name: v.string(),
    slug: v.string(),
    creatorEmail: v.string(),
    creatorName: v.string()
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const existing = await ctx.db
      .query("teams")
      .withIndex("by_clerk_org", (q) => q.eq("clerkOrganizationId", args.clerkOrganizationId))
      .first();

    if (existing) {
      return existing._id;
    }

    const teamId = await ctx.db.insert("teams", {
      clerkOrganizationId: args.clerkOrganizationId,
      name: args.name,
      slug: args.slug,
      status: "active",
      createdByUserId: args.createdByUserId,
      createdAt: now
    });

    await ctx.db.insert("teamMembers", {
      teamId,
      clerkUserId: args.createdByUserId,
      email: args.creatorEmail,
      displayName: args.creatorName,
      role: "owner",
      status: "active",
      joinedAt: now,
      lastSeenAt: now
    });

    return teamId;
  }
});

export const seedDemoHosts = mutation({
  args: {
    teamId: v.id("teams"),
    actorUserId: v.string()
  },
  handler: async (ctx, args) => {
    const existingHosts = await ctx.db
      .query("teamHosts")
      .withIndex("by_team", (q) => q.eq("teamId", args.teamId))
      .collect();

    if (existingHosts.length > 0) {
      return { created: 0 };
    }

    const now = Date.now();
    const hosts = [
      {
        label: "prod-bastion",
        group: "Production",
        tags: ["shared", "bastion"],
        hostname: "prod-bastion.internal",
        username: "ubuntu",
        port: 22,
        shareMode: "host_plus_shared_credential",
        notes: ["Shared bastion host for production access."]
      },
      {
        label: "prod-api-1",
        group: "Production",
        tags: ["api"],
        hostname: "10.0.1.10",
        username: "ubuntu",
        port: 22,
        shareMode: "host_only",
        notes: ["Use your own personal SSH credential."]
      },
      {
        label: "staging-web-1",
        group: "Staging",
        tags: ["web"],
        hostname: "10.0.2.21",
        username: "deploy",
        port: 22,
        shareMode: "host_plus_shared_credential",
        notes: ["Shared staging deployment box."]
      }
    ];

    for (const host of hosts) {
      await ctx.db.insert("teamHosts", {
        teamId: args.teamId,
        ...host,
        lastActivityAt: now,
        createdBy: args.actorUserId,
        createdAt: now,
        updatedBy: args.actorUserId,
        updatedAt: now
      });
    }

    return { created: hosts.length };
  }
});
