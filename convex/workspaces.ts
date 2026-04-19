import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

function normalizeSlug(raw: string, fallback: string): string {
  const slug = raw
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48);
  return slug || fallback.toLowerCase();
}

function workspaceRoleFromClerkRole(clerkRole: string): string {
  return clerkRole === "org:admin" ? "admin" : "member";
}

export const bootstrapForClerkOrganization = mutation({
  args: {
    clerkOrganizationId: v.string(),
    organizationName: v.string(),
    organizationSlug: v.string(),
    clerkUserId: v.string(),
    userEmail: v.string(),
    displayName: v.string(),
    clerkRole: v.string(),
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const existing = await ctx.db
      .query("workspaces")
      .withIndex("by_clerk_org", (q) => q.eq("clerkOrganizationId", args.clerkOrganizationId))
      .first();

    let workspaceId = existing?._id;
    if (!workspaceId) {
      workspaceId = await ctx.db.insert("workspaces", {
        clerkOrganizationId: args.clerkOrganizationId,
        name: args.organizationName,
        slug: normalizeSlug(args.organizationSlug || args.organizationName, args.clerkOrganizationId),
        status: "active",
        createdByUserId: args.clerkUserId,
        createdAt: now,
      });
    }

    const member = await ctx.db
      .query("workspaceMembers")
      .withIndex("by_workspace_user", (q) =>
        q.eq("workspaceId", workspaceId).eq("clerkUserId", args.clerkUserId),
      )
      .first();

    const workspaceRole =
      existing?.createdByUserId === args.clerkUserId || !existing
        ? "owner"
        : workspaceRoleFromClerkRole(args.clerkRole);

    if (!member) {
      await ctx.db.insert("workspaceMembers", {
        workspaceId,
        clerkUserId: args.clerkUserId,
        email: args.userEmail,
        displayName: args.displayName,
        workspaceRole,
        status: "active",
        joinedAt: now,
        lastSeenAt: now,
        createdAt: now,
        updatedAt: now,
      });
    } else {
      await ctx.db.patch(member._id, {
        email: args.userEmail,
        displayName: args.displayName,
        workspaceRole,
        status: "active",
        joinedAt: member.joinedAt ?? now,
        lastSeenAt: now,
        updatedAt: now,
      });
    }

    let defaultVault = await ctx.db
      .query("vaults")
      .withIndex("by_workspace_slug", (q) => q.eq("workspaceId", workspaceId).eq("slug", "general"))
      .first();
    if (!defaultVault) {
      const defaultVaultId = await ctx.db.insert("vaults", {
        workspaceId,
        name: "General",
        slug: "general",
        description: "Default SSHThing Teams vault.",
        createdAt: now,
        updatedAt: now,
      });
      defaultVault = await ctx.db.get(defaultVaultId);
      await ctx.db.insert("vaultMembers", {
        workspaceId,
        vaultId: defaultVaultId,
        clerkUserId: args.clerkUserId,
        vaultRole: "vault_admin",
        status: "active",
        createdAt: now,
        updatedAt: now,
      });
    }

    return {
      workspaceId,
      defaultVaultId: defaultVault!._id,
    };
  },
});

export const getWorkspaceSummary = query({
  args: {
    workspaceId: v.id("workspaces"),
  },
  handler: async (ctx, args) => {
    const workspace = await ctx.db.get(args.workspaceId);
    if (!workspace) {
      throw new Error("workspace_not_found");
    }

    return {
      id: workspace._id,
      clerkOrganizationId: workspace.clerkOrganizationId,
      name: workspace.name,
      slug: workspace.slug,
      status: workspace.status,
      createdByUserId: workspace.createdByUserId,
      createdAt: workspace.createdAt,
    };
  },
});
