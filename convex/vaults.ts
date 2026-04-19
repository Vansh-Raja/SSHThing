import { query } from "./_generated/server";
import { v } from "convex/values";

export const listForWorkspace = query({
  args: {
    workspaceId: v.id("workspaces"),
  },
  handler: async (ctx, args) => {
    const vaults = await ctx.db
      .query("vaults")
      .withIndex("by_workspace", (q) => q.eq("workspaceId", args.workspaceId))
      .collect();

    return vaults.map((vault) => ({
      id: vault._id,
      workspaceId: vault.workspaceId,
      name: vault.name,
      slug: vault.slug,
      description: vault.description,
      createdAt: vault.createdAt,
      updatedAt: vault.updatedAt,
    }));
  },
});
