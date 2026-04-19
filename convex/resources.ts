import { query } from "./_generated/server";
import { v } from "convex/values";

export const listForVault = query({
  args: {
    vaultId: v.id("vaults"),
  },
  handler: async (ctx, args) => {
    const resources = await ctx.db
      .query("resources")
      .withIndex("by_vault", (q) => q.eq("vaultId", args.vaultId))
      .collect();

    return resources.map((resource) => ({
      id: resource._id,
      vaultId: resource.vaultId,
      label: resource.label,
      group: resource.group,
      tags: resource.tags,
      hostname: resource.hostname,
      username: resource.username,
      port: resource.port,
      shareMode: resource.shareMode,
      notes: resource.notes,
      createdAt: resource.createdAt,
      updatedAt: resource.updatedAt,
    }));
  },
});

export const getResourceById = query({
  args: {
    resourceId: v.id("resources"),
  },
  handler: async (ctx, args) => {
    const resource = await ctx.db.get(args.resourceId);
    if (!resource) {
      return null;
    }
    return {
      id: resource._id,
      vaultId: resource.vaultId,
      label: resource.label,
      group: resource.group,
      tags: resource.tags,
      hostname: resource.hostname,
      username: resource.username,
      port: resource.port,
      shareMode: resource.shareMode,
      notes: resource.notes,
      createdAt: resource.createdAt,
      updatedAt: resource.updatedAt,
    };
  },
});
