import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

function randomCode(length: number): string {
  const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
  let value = "";
  for (let i = 0; i < length; i++) {
    value += alphabet[Math.floor(Math.random() * alphabet.length)];
  }
  return value;
}

export const start = mutation({
  args: {
    deviceName: v.string()
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const deviceCode = randomCode(8);
    const pollSecret = crypto.randomUUID();

    const sessionId = await ctx.db.insert("cliAuthSessions", {
      status: "pending",
      requestedAt: now,
      deviceName: args.deviceName,
      deviceCode,
      pollSecret
    });

    return {
      sessionId,
      deviceCode,
      pollSecret,
      authUrl: `/cli-auth/complete?session=${sessionId}`,
      pollIntervalMs: 2000
    };
  }
});

export const poll = query({
  args: {
    sessionId: v.id("cliAuthSessions"),
    pollSecret: v.string()
  },
  handler: async (ctx, args) => {
    const session = await ctx.db.get(args.sessionId);
    if (!session || session.pollSecret !== args.pollSecret) {
      return { status: "expired" as const };
    }

    return {
      status: session.status,
      completedAt: session.completedAt ?? null,
      teamId: session.teamId ?? null,
      userId: session.userId ?? null
    };
  }
});

export const complete = mutation({
  args: {
    sessionId: v.id("cliAuthSessions"),
    userId: v.string(),
    teamId: v.optional(v.id("teams"))
  },
  handler: async (ctx, args) => {
    await ctx.db.patch(args.sessionId, {
      status: "completed",
      completedAt: Date.now(),
      userId: args.userId,
      teamId: args.teamId
    });

    return { ok: true };
  }
});

export const expire = mutation({
  args: {
    sessionId: v.id("cliAuthSessions")
  },
  handler: async (ctx, args) => {
    await ctx.db.patch(args.sessionId, {
      status: "expired"
    });

    return { ok: true };
  }
});
