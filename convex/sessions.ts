import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

export const startCliAuth = mutation({
  args: {
    deviceName: v.string(),
    deviceCode: v.string(),
    pollSecret: v.string(),
    ttlSeconds: v.number(),
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const expiresAt = now + args.ttlSeconds * 1000;
    const sessionId = await ctx.db.insert("cliAuthSessions", {
      deviceName: args.deviceName,
      deviceCode: args.deviceCode,
      pollSecret: args.pollSecret,
      status: "pending",
      requestedAt: now,
      expiresAt,
    });

    return {
      sessionId,
      deviceCode: args.deviceCode,
      pollSecret: args.pollSecret,
      expiresAt,
    };
  },
});

export const getCliAuthStatus = query({
  args: {
    sessionId: v.id("cliAuthSessions"),
    pollSecret: v.string(),
  },
  handler: async (ctx, args) => {
    const session = await ctx.db.get(args.sessionId);
    if (!session || session.pollSecret !== args.pollSecret) {
      throw new Error("session_not_found");
    }
    if (session.expiresAt <= Date.now() && session.status !== "completed") {
      return {
        status: "expired",
        expiresAt: session.expiresAt,
      };
    }
    return {
      status: session.status,
      clerkUserId: session.clerkUserId ?? null,
      deviceName: session.deviceName,
      expiresAt: session.expiresAt,
      completedAt: session.completedAt ?? null,
    };
  },
});

export const completeCliAuth = mutation({
  args: {
    sessionId: v.id("cliAuthSessions"),
    deviceCode: v.string(),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const session = await ctx.db.get(args.sessionId);
    if (!session) {
      throw new Error("session_not_found");
    }
    if (session.expiresAt <= Date.now()) {
      throw new Error("session_expired");
    }
    if (args.deviceCode && session.deviceCode !== args.deviceCode) {
      throw new Error("device_code_mismatch");
    }

    await ctx.db.patch(args.sessionId, {
      status: "completed",
      completedAt: Date.now(),
      clerkUserId: args.clerkUserId,
    });

    return { ok: true };
  },
});

export const createTuiSession = mutation({
  args: {
    clerkUserId: v.string(),
    deviceName: v.string(),
    accessTokenHash: v.string(),
    refreshTokenHash: v.string(),
    accessTtlSeconds: v.number(),
    refreshTtlSeconds: v.number(),
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const accessExpiresAt = now + args.accessTtlSeconds * 1000;
    const refreshExpiresAt = now + args.refreshTtlSeconds * 1000;

    const sessionId = await ctx.db.insert("tuiSessions", {
      clerkUserId: args.clerkUserId,
      accessTokenHash: args.accessTokenHash,
      refreshTokenHash: args.refreshTokenHash,
      deviceName: args.deviceName,
      accessExpiresAt,
      refreshExpiresAt,
      lastSeenAt: now,
      createdAt: now,
    });

    return {
      sessionId,
      accessExpiresAt,
      refreshExpiresAt,
    };
  },
});

export const getTuiSessionByAccessHash = query({
  args: {
    accessTokenHash: v.string(),
  },
  handler: async (ctx, args) => {
    const session = await ctx.db
      .query("tuiSessions")
      .withIndex("by_access_hash", (q) => q.eq("accessTokenHash", args.accessTokenHash))
      .first();
    if (!session) {
      return null;
    }
    return {
      _id: session._id,
      clerkUserId: session.clerkUserId,
      deviceName: session.deviceName,
      accessExpiresAt: session.accessExpiresAt,
      refreshExpiresAt: session.refreshExpiresAt,
      lastSeenAt: session.lastSeenAt,
      revokedAt: session.revokedAt ?? null,
      createdAt: session.createdAt,
    };
  },
});

export const getTuiSessionByRefreshHash = query({
  args: {
    refreshTokenHash: v.string(),
  },
  handler: async (ctx, args) => {
    const session = await ctx.db
      .query("tuiSessions")
      .withIndex("by_refresh_hash", (q) => q.eq("refreshTokenHash", args.refreshTokenHash))
      .first();
    if (!session) {
      return null;
    }
    return {
      _id: session._id,
      clerkUserId: session.clerkUserId,
      deviceName: session.deviceName,
      accessExpiresAt: session.accessExpiresAt,
      refreshExpiresAt: session.refreshExpiresAt,
      lastSeenAt: session.lastSeenAt,
      revokedAt: session.revokedAt ?? null,
      createdAt: session.createdAt,
    };
  },
});

export const rotateAccessToken = mutation({
  args: {
    sessionId: v.id("tuiSessions"),
    accessTokenHash: v.string(),
    accessTtlSeconds: v.number(),
  },
  handler: async (ctx, args) => {
    const accessExpiresAt = Date.now() + args.accessTtlSeconds * 1000;
    await ctx.db.patch(args.sessionId, {
      accessTokenHash: args.accessTokenHash,
      accessExpiresAt,
      lastSeenAt: Date.now(),
    });
    return { accessExpiresAt };
  },
});

export const revokeTuiSession = mutation({
  args: {
    sessionId: v.id("tuiSessions"),
  },
  handler: async (ctx, args) => {
    await ctx.db.patch(args.sessionId, {
      revokedAt: Date.now(),
    });
    return { ok: true };
  },
});

export const markTuiSessionSeen = mutation({
  args: {
    sessionId: v.id("tuiSessions"),
  },
  handler: async (ctx, args) => {
    await ctx.db.patch(args.sessionId, {
      lastSeenAt: Date.now(),
    });
    return { ok: true };
  },
});
