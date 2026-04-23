import { query } from "./_generated/server";
import { v } from "convex/values";

import { requireTeamPermission } from "./teamAccess";

export const listForTeam = query({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_hosts");
    const events = await ctx.db
      .query("teamAuditEvents")
      .withIndex("by_team_and_created_at", (q) => q.eq("teamId", args.teamId))
      .order("desc")
      .take(100);

    return events.map((event) => ({
      id: event._id,
      teamId: event.teamId,
      actorClerkUserId: event.actorClerkUserId,
      actorDisplayName: event.actorDisplayName,
      entityType: event.entityType,
      entityId: event.entityId,
      eventType: event.eventType,
      targetClerkUserId: event.targetClerkUserId ?? null,
      targetDisplayName: event.targetDisplayName ?? null,
      summary: event.summary,
      metadata: event.metadata ?? null,
      createdAt: event.createdAt,
    }));
  },
});
