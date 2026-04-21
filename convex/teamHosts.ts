import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

import { requireHostPermission, requireTeamPermission } from "./teamAccess";

function normalizeCredentialMode(value: string): "shared" | "per_member" {
  return value === "per_member" ? "per_member" : "shared";
}

function normalizeCredentialType(value: string): "none" | "password" | "private_key" {
  switch (value) {
    case "password":
    case "private_key":
    case "none":
      return value;
    default:
      return "none";
  }
}

function normalizeVisibility(value: string): string {
  return value === "revealed_to_access_holders" ? value : "revealed_to_access_holders";
}

export const listForTeam = query({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "read");
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
      authMode: host.authMode ?? host.credentialType,
      credentialMode: host.credentialMode,
      credentialType: host.credentialType,
      secretVisibility: host.secretVisibility,
      lastConnectedAt: host.lastConnectedAt ?? null,
      createdAt: host.createdAt,
      updatedAt: host.updatedAt,
    }));
  },
});

export const create = mutation({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
    label: v.string(),
    hostname: v.string(),
    username: v.string(),
    port: v.number(),
    group: v.string(),
    tags: v.array(v.string()),
    credentialMode: v.string(),
    credentialType: v.string(),
    secretVisibility: v.string(),
    sharedCredentialCiphertext: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_hosts");

    const now = Date.now();
    const credentialMode = normalizeCredentialMode(args.credentialMode);
    const credentialType = normalizeCredentialType(args.credentialType);
    const hostId = await ctx.db.insert("teamHosts", {
      teamId: args.teamId,
      label: args.label.trim(),
      hostname: args.hostname.trim(),
      username: args.username.trim(),
      port: args.port > 0 ? args.port : 22,
      group: args.group.trim(),
      tags: args.tags.map((tag) => tag.trim()).filter(Boolean),
      authMode: credentialType,
      credentialMode,
      credentialType,
      secretVisibility: normalizeVisibility(args.secretVisibility),
      createdByClerkUserId: args.clerkUserId,
      updatedByClerkUserId: args.clerkUserId,
      createdAt: now,
      updatedAt: now,
    });

    if (credentialMode === "shared" && credentialType !== "none" && args.sharedCredentialCiphertext) {
      await ctx.db.insert("teamHostSharedCredentials", {
        hostId,
        credentialType,
        ciphertext: args.sharedCredentialCiphertext,
        updatedByClerkUserId: args.clerkUserId,
        createdAt: now,
        updatedAt: now,
      });
    }

    return { id: hostId };
  },
});

export const getHost = query({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireHostPermission(ctx, args.hostId, args.clerkUserId, "read");
    const shared = await ctx.db
      .query("teamHostSharedCredentials")
      .withIndex("by_host", (q) => q.eq("hostId", args.hostId))
      .first();

    return {
      id: access.host._id,
      teamId: access.host.teamId,
      label: access.host.label,
      hostname: access.host.hostname,
      username: access.host.username,
      port: access.host.port,
      group: access.host.group,
      tags: access.host.tags,
      authMode: access.host.authMode ?? access.host.credentialType,
      credentialMode: access.host.credentialMode,
      credentialType: access.host.credentialType,
      secretVisibility: access.host.secretVisibility,
      lastConnectedAt: access.host.lastConnectedAt ?? null,
      sharedCredential: access.host.credentialMode === "shared" ? shared?.ciphertext ?? null : null,
      createdAt: access.host.createdAt,
      updatedAt: access.host.updatedAt,
    };
  },
});

export const update = mutation({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
    label: v.string(),
    hostname: v.string(),
    username: v.string(),
    port: v.number(),
    group: v.string(),
    tags: v.array(v.string()),
    credentialMode: v.string(),
    credentialType: v.string(),
    secretVisibility: v.string(),
    sharedCredentialCiphertext: v.optional(v.union(v.string(), v.null())),
  },
  handler: async (ctx, args) => {
    const access = await requireHostPermission(ctx, args.hostId, args.clerkUserId, "manage_hosts");
    const now = Date.now();
    const credentialMode = normalizeCredentialMode(args.credentialMode);
    const credentialType = normalizeCredentialType(args.credentialType);

    await ctx.db.patch(args.hostId, {
      label: args.label.trim(),
      hostname: args.hostname.trim(),
      username: args.username.trim(),
      port: args.port > 0 ? args.port : 22,
      group: args.group.trim(),
      tags: args.tags.map((tag) => tag.trim()).filter(Boolean),
      authMode: credentialType,
      credentialMode,
      credentialType,
      secretVisibility: normalizeVisibility(args.secretVisibility),
      updatedByClerkUserId: args.clerkUserId,
      updatedAt: now,
    });

    const existingShared = await ctx.db
      .query("teamHostSharedCredentials")
      .withIndex("by_host", (q) => q.eq("hostId", args.hostId))
      .first();

    if (credentialMode !== "shared" || credentialType === "none") {
      if (existingShared) {
        await ctx.db.delete(existingShared._id);
      }
      return { ok: true };
    }

    if (args.sharedCredentialCiphertext === null) {
      if (existingShared) {
        await ctx.db.delete(existingShared._id);
      }
      return { ok: true };
    }

    if (typeof args.sharedCredentialCiphertext === "string" && args.sharedCredentialCiphertext !== "") {
      if (existingShared) {
        await ctx.db.patch(existingShared._id, {
          credentialType,
          ciphertext: args.sharedCredentialCiphertext,
          updatedByClerkUserId: args.clerkUserId,
          updatedAt: now,
        });
      } else {
        await ctx.db.insert("teamHostSharedCredentials", {
          hostId: args.hostId,
          credentialType,
          ciphertext: args.sharedCredentialCiphertext,
          updatedByClerkUserId: args.clerkUserId,
          createdAt: now,
          updatedAt: now,
        });
      }
    }

    return { ok: true };
  },
});

export const remove = mutation({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    await requireHostPermission(ctx, args.hostId, args.clerkUserId, "manage_hosts");
    const shared = await ctx.db
      .query("teamHostSharedCredentials")
      .withIndex("by_host", (q) => q.eq("hostId", args.hostId))
      .first();
    if (shared) {
      await ctx.db.delete(shared._id);
    }
    const personal = await ctx.db
      .query("teamHostPersonalCredentials")
      .withIndex("by_host_and_user", (q) => q.eq("hostId", args.hostId))
      .collect();
    for (const credential of personal) {
      await ctx.db.delete(credential._id);
    }
    await ctx.db.delete(args.hostId);
    return { ok: true };
  },
});

export const getMyCredential = query({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireHostPermission(ctx, args.hostId, args.clerkUserId, "read");
    const credential = await ctx.db
      .query("teamHostPersonalCredentials")
      .withIndex("by_host_and_user", (q) => q.eq("hostId", args.hostId).eq("clerkUserId", args.clerkUserId))
      .first();

    return {
      hostId: args.hostId,
      credentialMode: access.host.credentialMode,
      credentialType: access.host.credentialType,
      username: credential?.username ?? null,
      ciphertext: credential?.ciphertext ?? null,
      hasCredential: Boolean(credential?.ciphertext),
    };
  },
});

export const upsertMyCredential = mutation({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
    username: v.optional(v.string()),
    credentialType: v.string(),
    ciphertext: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireHostPermission(ctx, args.hostId, args.clerkUserId, "read");
    if (access.host.credentialMode !== "per_member") {
      throw new Error("host_not_personal_credential_mode");
    }

    const existing = await ctx.db
      .query("teamHostPersonalCredentials")
      .withIndex("by_host_and_user", (q) => q.eq("hostId", args.hostId).eq("clerkUserId", args.clerkUserId))
      .first();

    const now = Date.now();
    const credentialType = normalizeCredentialType(args.credentialType);
    if (existing) {
      await ctx.db.patch(existing._id, {
        username: args.username?.trim() || undefined,
        credentialType,
        ciphertext: args.ciphertext,
        updatedAt: now,
      });
    } else {
      await ctx.db.insert("teamHostPersonalCredentials", {
        hostId: args.hostId,
        clerkUserId: args.clerkUserId,
        username: args.username?.trim() || undefined,
        credentialType,
        ciphertext: args.ciphertext,
        createdAt: now,
        updatedAt: now,
      });
    }

    return { ok: true };
  },
});

export const deleteMyCredential = mutation({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    await requireHostPermission(ctx, args.hostId, args.clerkUserId, "read");
    const existing = await ctx.db
      .query("teamHostPersonalCredentials")
      .withIndex("by_host_and_user", (q) => q.eq("hostId", args.hostId).eq("clerkUserId", args.clerkUserId))
      .first();

    if (existing) {
      await ctx.db.delete(existing._id);
    }

    return { ok: true };
  },
});

export const getConnectConfig = query({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireHostPermission(ctx, args.hostId, args.clerkUserId, "read");

    if (access.host.credentialMode === "shared") {
      const shared = await ctx.db
        .query("teamHostSharedCredentials")
        .withIndex("by_host", (q) => q.eq("hostId", args.hostId))
        .first();
      if (access.host.credentialType !== "none" && !shared) {
        throw new Error("shared_credential_not_configured");
      }
      return {
        hostId: access.host._id,
        teamId: access.host.teamId,
        label: access.host.label,
        hostname: access.host.hostname,
        username: access.host.username,
        port: access.host.port,
        credentialMode: access.host.credentialMode,
        credentialType: access.host.credentialType,
        secret: shared?.ciphertext ?? "",
      };
    }

    const personal = await ctx.db
      .query("teamHostPersonalCredentials")
      .withIndex("by_host_and_user", (q) => q.eq("hostId", args.hostId).eq("clerkUserId", args.clerkUserId))
      .first();
    if (access.host.credentialType !== "none" && !personal) {
      throw new Error("personal_credential_not_configured");
    }

    return {
      hostId: access.host._id,
      teamId: access.host.teamId,
      label: access.host.label,
      hostname: access.host.hostname,
      username: personal?.username ?? access.host.username,
      port: access.host.port,
      credentialMode: access.host.credentialMode,
      credentialType: personal?.credentialType ?? access.host.credentialType,
      secret: personal?.ciphertext ?? "",
    };
  },
});

export const markConnected = mutation({
  args: {
    hostId: v.id("teamHosts"),
  },
  handler: async (ctx, args) => {
    await ctx.db.patch(args.hostId, {
      lastConnectedAt: Date.now(),
      updatedAt: Date.now(),
    });
    return { ok: true };
  },
});
