import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

function normalizeWorkspaceRole(role: string): string {
  switch (role) {
    case "owner":
    case "admin":
    case "member":
      return role;
    default:
      return "member";
  }
}

function normalizeVaultRole(role: string): string {
  switch (role) {
    case "vault_admin":
    case "editor":
    case "operator":
    case "restricted_operator":
    case "viewer":
      return role;
    default:
      return "viewer";
  }
}

export const getAccessContext = query({
  args: {
    workspaceId: v.id("workspaces"),
    vaultId: v.id("vaults"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const workspaceMember = await ctx.db
      .query("workspaceMembers")
      .withIndex("by_workspace_user", (q) =>
        q.eq("workspaceId", args.workspaceId).eq("clerkUserId", args.clerkUserId),
      )
      .first();
    if (!workspaceMember || workspaceMember.status !== "active") {
      return null;
    }

    const vaultMember = await ctx.db
      .query("vaultMembers")
      .withIndex("by_vault_user", (q) => q.eq("vaultId", args.vaultId).eq("clerkUserId", args.clerkUserId))
      .first();

    return {
      workspaceRole: workspaceMember.workspaceRole,
      vaultRole: vaultMember?.status === "active" ? vaultMember.vaultRole : "",
    };
  },
});

export const listMembersForVault = query({
  args: {
    workspaceId: v.id("workspaces"),
    vaultId: v.id("vaults"),
  },
  handler: async (ctx, args) => {
    const workspaceMembers = await ctx.db
      .query("workspaceMembers")
      .withIndex("by_workspace", (q) => q.eq("workspaceId", args.workspaceId))
      .collect();
    const vaultMembers = await ctx.db
      .query("vaultMembers")
      .withIndex("by_vault", (q) => q.eq("vaultId", args.vaultId))
      .collect();

    const byUser = new Map(vaultMembers.map((member) => [member.clerkUserId, member]));
    return workspaceMembers.map((member) => ({
      id: member._id,
      workspaceId: member.workspaceId,
      clerkUserId: member.clerkUserId,
      email: member.email,
      displayName: member.displayName,
      workspaceRole: member.workspaceRole,
      vaultRole: byUser.get(member.clerkUserId)?.vaultRole ?? "",
      status: member.status,
      invitationId: member.invitationId ?? null,
      joinedAt: member.joinedAt ?? null,
      lastSeenAt: member.lastSeenAt ?? null,
    }));
  },
});

export const getWorkspaceMember = query({
  args: {
    workspaceId: v.id("workspaces"),
    memberId: v.id("workspaceMembers"),
  },
  handler: async (ctx, args) => {
    const member = await ctx.db.get(args.memberId);
    if (!member || member.workspaceId !== args.workspaceId) {
      return null;
    }
    return {
      id: member._id,
      clerkUserId: member.clerkUserId,
      email: member.email,
      displayName: member.displayName,
    };
  },
});

export const upsertPendingInvitation = mutation({
  args: {
    workspaceId: v.id("workspaces"),
    vaultId: v.id("vaults"),
    invitationId: v.string(),
    email: v.string(),
    invitedBy: v.string(),
    vaultRole: v.string(),
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const syntheticUserID = `invite:${args.invitationId}`;
    const displayName = args.email.split("@")[0] || args.email;

    const workspaceMember = await ctx.db
      .query("workspaceMembers")
      .withIndex("by_workspace_user", (q) =>
        q.eq("workspaceId", args.workspaceId).eq("clerkUserId", syntheticUserID),
      )
      .first();

    let workspaceMemberId = workspaceMember?._id;
    if (!workspaceMemberId) {
      workspaceMemberId = await ctx.db.insert("workspaceMembers", {
        workspaceId: args.workspaceId,
        clerkUserId: syntheticUserID,
        email: args.email,
        displayName,
        workspaceRole: "member",
        status: "invited",
        invitationId: args.invitationId,
        createdAt: now,
        updatedAt: now,
      });
    } else {
      await ctx.db.patch(workspaceMemberId, {
        email: args.email,
        displayName,
        status: "invited",
        invitationId: args.invitationId,
        updatedAt: now,
      });
    }

    const existingVaultMember = await ctx.db
      .query("vaultMembers")
      .withIndex("by_vault_user", (q) => q.eq("vaultId", args.vaultId).eq("clerkUserId", syntheticUserID))
      .first();

    if (!existingVaultMember) {
      await ctx.db.insert("vaultMembers", {
        workspaceId: args.workspaceId,
        vaultId: args.vaultId,
        clerkUserId: syntheticUserID,
        vaultRole: normalizeVaultRole(args.vaultRole),
        status: "invited",
        createdAt: now,
        updatedAt: now,
      });
    } else {
      await ctx.db.patch(existingVaultMember._id, {
        vaultRole: normalizeVaultRole(args.vaultRole),
        status: "invited",
        updatedAt: now,
      });
    }

    return { ok: true };
  },
});

export const setVaultRole = mutation({
  args: {
    workspaceId: v.id("workspaces"),
    vaultId: v.id("vaults"),
    clerkUserId: v.string(),
    vaultRole: v.string(),
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const role = normalizeVaultRole(args.vaultRole);
    const existing = await ctx.db
      .query("vaultMembers")
      .withIndex("by_vault_user", (q) => q.eq("vaultId", args.vaultId).eq("clerkUserId", args.clerkUserId))
      .first();

    if (!existing) {
      const inserted = await ctx.db.insert("vaultMembers", {
        workspaceId: args.workspaceId,
        vaultId: args.vaultId,
        clerkUserId: args.clerkUserId,
        vaultRole: role,
        status: "active",
        createdAt: now,
        updatedAt: now,
      });
      return {
        id: inserted,
        clerkUserId: args.clerkUserId,
        vaultId: args.vaultId,
        vaultRole: role,
        status: "active",
      };
    }

    await ctx.db.patch(existing._id, {
      vaultRole: role,
      status: "active",
      updatedAt: now,
    });

    return {
      id: existing._id,
      clerkUserId: args.clerkUserId,
      vaultId: args.vaultId,
      vaultRole: role,
      status: "active",
    };
  },
});

export const deactivateWorkspaceMember = mutation({
  args: {
    workspaceId: v.id("workspaces"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const workspaceMember = await ctx.db
      .query("workspaceMembers")
      .withIndex("by_workspace_user", (q) =>
        q.eq("workspaceId", args.workspaceId).eq("clerkUserId", args.clerkUserId),
      )
      .first();

    if (workspaceMember) {
      await ctx.db.patch(workspaceMember._id, {
        status: "removed",
        updatedAt: now,
      });
    }

    const vaultMembers = await ctx.db
      .query("vaultMembers")
      .withIndex("by_workspace_user", (q) =>
        q.eq("workspaceId", args.workspaceId).eq("clerkUserId", args.clerkUserId),
      )
      .collect();
    for (const member of vaultMembers) {
      await ctx.db.patch(member._id, {
        status: "removed",
        updatedAt: now,
      });
    }

    return { ok: true };
  },
});

export const syncWorkspaceMembership = mutation({
  args: {
    workspaceId: v.id("workspaces"),
    clerkUserId: v.string(),
    email: v.string(),
    displayName: v.string(),
    workspaceRole: v.string(),
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const workspaceRole = normalizeWorkspaceRole(args.workspaceRole);

    const direct = await ctx.db
      .query("workspaceMembers")
      .withIndex("by_workspace_user", (q) =>
        q.eq("workspaceId", args.workspaceId).eq("clerkUserId", args.clerkUserId),
      )
      .first();

    const invited = await ctx.db
      .query("workspaceMembers")
      .withIndex("by_workspace", (q) => q.eq("workspaceId", args.workspaceId))
      .collect();
    const invitedRecord = invited.find(
      (member) => member.status === "invited" && member.email.toLowerCase() === args.email.toLowerCase(),
    );

    if (direct) {
      await ctx.db.patch(direct._id, {
        email: args.email,
        displayName: args.displayName,
        workspaceRole,
        status: "active",
        joinedAt: direct.joinedAt ?? now,
        lastSeenAt: now,
        updatedAt: now,
      });
      return { ok: true };
    }

    if (invitedRecord) {
      const oldSyntheticID = invitedRecord.clerkUserId;
      await ctx.db.patch(invitedRecord._id, {
        clerkUserId: args.clerkUserId,
        displayName: args.displayName,
        workspaceRole,
        status: "active",
        joinedAt: now,
        lastSeenAt: now,
        updatedAt: now,
      });

      const invitedVaultMembers = await ctx.db
        .query("vaultMembers")
        .withIndex("by_workspace_user", (q) =>
          q.eq("workspaceId", args.workspaceId).eq("clerkUserId", oldSyntheticID),
        )
        .collect();
      for (const member of invitedVaultMembers) {
        await ctx.db.patch(member._id, {
          clerkUserId: args.clerkUserId,
          status: "active",
          updatedAt: now,
        });
      }
      return { ok: true };
    }

    await ctx.db.insert("workspaceMembers", {
      workspaceId: args.workspaceId,
      clerkUserId: args.clerkUserId,
      email: args.email,
      displayName: args.displayName,
      workspaceRole,
      status: "active",
      joinedAt: now,
      lastSeenAt: now,
      createdAt: now,
      updatedAt: now,
    });
    return { ok: true };
  },
});
