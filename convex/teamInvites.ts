import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

import { normalizeTeamRole, requireTeamPermission } from "./teamAccess";

function ensurePending(invite: { status: string; expiresAt: number }) {
  if (invite.status !== "pending") {
    throw new Error("invite_not_pending");
  }
  if (invite.expiresAt <= Date.now()) {
    throw new Error("invite_expired");
  }
}

export const listForUser = query({
  args: {
    clerkUserId: v.string(),
    emailLower: v.string(),
  },
  handler: async (ctx, args) => {
    const incoming = await ctx.db
      .query("teamInvites")
      .withIndex("by_email_lower_and_status", (q) => q.eq("emailLower", args.emailLower).eq("status", "pending"))
      .collect();

    const sent = await ctx.db
      .query("teamInvites")
      .withIndex("by_invited_by_and_status", (q) =>
        q.eq("invitedByClerkUserId", args.clerkUserId).eq("status", "pending"),
      )
      .collect();

    const hydrate = async (invite: (typeof incoming)[number]) => {
      const team = await ctx.db.get(invite.teamId);
      if (!team || team.status !== "active") {
        return null;
      }
      return {
        id: invite._id,
        teamId: invite.teamId,
        teamName: team.name,
        teamSlug: team.slug,
        email: invite.emailLower,
        role: invite.role,
        status: invite.status,
        expiresAt: invite.expiresAt,
        createdAt: invite.createdAt,
        tokenCiphertext: invite.invitedByClerkUserId === args.clerkUserId ? invite.tokenCiphertext : null,
      };
    };

    return {
      incoming: (await Promise.all(incoming.map(hydrate))).filter(Boolean),
      sent: (await Promise.all(sent.map(hydrate))).filter(Boolean),
    };
  },
});

export const getForToken = query({
  args: {
    inviteId: v.id("teamInvites"),
    tokenHash: v.optional(v.union(v.string(), v.null())),
    clerkUserId: v.string(),
    emailLower: v.string(),
  },
  handler: async (ctx, args) => {
    const invite = await ctx.db.get(args.inviteId);
    if (!invite) {
      return null;
    }

    const tokenMatches = Boolean(args.tokenHash) && invite.tokenHash === args.tokenHash;
    const viewerMatches = invite.emailLower === args.emailLower || invite.invitedByClerkUserId === args.clerkUserId;
    if (!tokenMatches && !viewerMatches) {
      return null;
    }

    const team = await ctx.db.get(invite.teamId);
    if (!team || team.status !== "active") {
      return null;
    }

    return {
      id: invite._id,
      teamId: invite.teamId,
      teamName: team.name,
      teamSlug: team.slug,
      email: invite.emailLower,
      role: invite.role,
      status: invite.status,
      expiresAt: invite.expiresAt,
      createdAt: invite.createdAt,
    };
  },
});

export const create = mutation({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
    emailLower: v.string(),
    role: v.string(),
    tokenHash: v.string(),
    tokenCiphertext: v.string(),
    expiresAt: v.number(),
  },
  handler: async (ctx, args) => {
    await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_members");
    const role = normalizeTeamRole(args.role);
    if (role === "owner") {
      throw new Error("cannot_invite_owner");
    }

    const existing = await ctx.db
      .query("teamInvites")
      .withIndex("by_team", (q) => q.eq("teamId", args.teamId))
      .collect();

    const activeInvite = existing.find(
      (invite) => invite.status === "pending" && invite.emailLower === args.emailLower,
    );

    const existingMembers = await ctx.db
      .query("teamMembers")
      .withIndex("by_team", (q) => q.eq("teamId", args.teamId))
      .collect();
    const activeMember = existingMembers.find(
      (member) => member.status === "active" && member.email.toLowerCase() === args.emailLower,
    );
    if (activeMember) {
      throw new Error("member_already_active");
    }

    const now = Date.now();
    if (activeInvite) {
      await ctx.db.patch(activeInvite._id, {
        role,
        tokenHash: args.tokenHash,
        tokenCiphertext: args.tokenCiphertext,
        invitedByClerkUserId: args.clerkUserId,
        expiresAt: args.expiresAt,
        updatedAt: now,
      });
      return {
        id: activeInvite._id,
        teamId: args.teamId,
        email: args.emailLower,
        role,
        status: "pending",
        expiresAt: args.expiresAt,
      };
    }

    const inviteId = await ctx.db.insert("teamInvites", {
      teamId: args.teamId,
      emailLower: args.emailLower,
      role,
      invitedByClerkUserId: args.clerkUserId,
      status: "pending",
      tokenHash: args.tokenHash,
      tokenCiphertext: args.tokenCiphertext,
      expiresAt: args.expiresAt,
      createdAt: now,
      updatedAt: now,
    });

    return {
      id: inviteId,
      teamId: args.teamId,
      email: args.emailLower,
      role,
      status: "pending",
      expiresAt: args.expiresAt,
    };
  },
});

export const accept = mutation({
  args: {
    inviteId: v.id("teamInvites"),
    tokenHash: v.optional(v.union(v.string(), v.null())),
    clerkUserId: v.string(),
    emailLower: v.string(),
    displayName: v.string(),
  },
  handler: async (ctx, args) => {
    const invite = await ctx.db.get(args.inviteId);
    if (!invite) {
      throw new Error("invite_not_found");
    }
    const tokenMatches = Boolean(args.tokenHash) && invite.tokenHash === args.tokenHash;
    if (!tokenMatches && invite.emailLower !== args.emailLower) {
      throw new Error("invite_not_found");
    }
    ensurePending(invite);
    if (invite.emailLower !== args.emailLower) {
      throw new Error("invite_email_mismatch");
    }

    const team = await ctx.db.get(invite.teamId);
    if (!team || team.status !== "active") {
      throw new Error("team_not_found");
    }

    const existing = await ctx.db
      .query("teamMembers")
      .withIndex("by_team_and_user", (q) => q.eq("teamId", invite.teamId).eq("clerkUserId", args.clerkUserId))
      .first();

    const now = Date.now();
    if (existing) {
      await ctx.db.patch(existing._id, {
        email: args.emailLower,
        displayName: args.displayName,
        role: existing.clerkUserId === team.ownerClerkUserId ? "owner" : normalizeTeamRole(invite.role),
        status: "active",
        joinedAt: existing.joinedAt ?? now,
        updatedAt: now,
      });
    } else {
      await ctx.db.insert("teamMembers", {
        teamId: invite.teamId,
        clerkUserId: args.clerkUserId,
        email: args.emailLower,
        displayName: args.displayName,
        role: normalizeTeamRole(invite.role),
        status: "active",
        joinedAt: now,
        createdAt: now,
        updatedAt: now,
      });
    }

    await ctx.db.patch(invite._id, {
      status: "accepted",
      acceptedAt: now,
      acceptedByClerkUserId: args.clerkUserId,
      updatedAt: now,
    });

    return {
      ok: true,
      teamId: invite.teamId,
      teamName: team.name,
      teamSlug: team.slug,
    };
  },
});

export const revoke = mutation({
  args: {
    inviteId: v.id("teamInvites"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const invite = await ctx.db.get(args.inviteId);
    if (!invite) {
      throw new Error("invite_not_found");
    }
    await requireTeamPermission(ctx, invite.teamId, args.clerkUserId, "manage_members");
    ensurePending(invite);

    await ctx.db.patch(invite._id, {
      status: "revoked",
      updatedAt: Date.now(),
    });
    return { ok: true };
  },
});
