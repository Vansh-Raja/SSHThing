import type { Doc, Id } from "./_generated/dataModel";
import type { MutationCtx, QueryCtx } from "./_generated/server";
import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

import { hasTeamPermission, requireHostPermission, requireTeamPermission } from "./teamAccess";

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

function normalizeNotes(value: string | undefined): string {
  return (value ?? "").trim();
}

type TeamCtx = QueryCtx | MutationCtx;

async function getMemberRecord(
  ctx: TeamCtx,
  teamId: Id<"teams">,
  clerkUserId: string,
): Promise<Doc<"teamMembers"> | null> {
  return ctx.db
    .query("teamMembers")
    .withIndex("by_team_and_user", (q) => q.eq("teamId", teamId).eq("clerkUserId", clerkUserId))
    .first();
}

function memberDisplayName(
  member: Pick<Doc<"teamMembers">, "displayName" | "email" | "clerkUserId"> | null,
  fallbackUserId: string,
): string {
  return member?.displayName?.trim() || member?.email?.trim() || fallbackUserId;
}

function roleOrder(role: string): number {
  switch (role) {
    case "owner":
      return 0;
    case "admin":
      return 1;
    default:
      return 2;
  }
}

async function writeAuditEvent(
  ctx: MutationCtx,
  args: {
    teamId: Id<"teams">;
    actorClerkUserId: string;
    actorDisplayName: string;
    entityType: string;
    entityId: string;
    eventType: string;
    targetClerkUserId?: string;
    targetDisplayName?: string;
    summary: string;
    metadata?: {
      hostLabel?: string;
      credentialMode?: string;
      credentialType?: string;
    };
  },
) {
  await ctx.db.insert("teamAuditEvents", {
    teamId: args.teamId,
    actorClerkUserId: args.actorClerkUserId,
    actorDisplayName: args.actorDisplayName,
    entityType: args.entityType,
    entityId: args.entityId,
    eventType: args.eventType,
    targetClerkUserId: args.targetClerkUserId,
    targetDisplayName: args.targetDisplayName,
    summary: args.summary,
    metadata: args.metadata,
    createdAt: Date.now(),
  });
}

export const listForTeam = query({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "read");
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
      notes: normalizeNotes(host.notes),
      authMode: host.authMode ?? host.credentialType,
      credentialMode: host.credentialMode,
      credentialType: host.credentialType,
      secretVisibility: host.secretVisibility,
      lastConnectedAt: host.lastConnectedAt ?? null,
      createdAt: host.createdAt,
      updatedAt: host.updatedAt,
      canManageHosts: hasTeamPermission(access.role, "manage_hosts"),
      canRevealSecrets: hasTeamPermission(access.role, "reveal_secret"),
      canEditNotes: hasTeamPermission(access.role, "edit_notes"),
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
    notes: v.optional(v.string()),
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
      notes: normalizeNotes(args.notes),
      authMode: credentialType,
      credentialMode,
      credentialType,
      secretVisibility: normalizeVisibility(args.secretVisibility),
      createdByClerkUserId: args.clerkUserId,
      updatedByClerkUserId: args.clerkUserId,
      createdAt: now,
      updatedAt: now,
    });

    // Store a shared credential whenever ciphertext is supplied and the host
    // has a meaningful credential type. Allowed in per_member mode too, where
    // the stored row becomes the fallback used when a member has no personal
    // credential.
    if (credentialType !== "none" && args.sharedCredentialCiphertext) {
      await ctx.db.insert("teamHostSharedCredentials", {
        hostId,
        credentialType,
        ciphertext: args.sharedCredentialCiphertext,
        updatedByClerkUserId: args.clerkUserId,
        createdAt: now,
        updatedAt: now,
      });
    }

    return {
      id: hostId,
      teamId: args.teamId,
      label: args.label.trim(),
      hostname: args.hostname.trim(),
      username: args.username.trim(),
      port: args.port > 0 ? args.port : 22,
      group: args.group.trim(),
      tags: args.tags.map((tag) => tag.trim()).filter(Boolean),
      notes: normalizeNotes(args.notes),
      authMode: credentialType,
      credentialMode,
      credentialType,
      secretVisibility: normalizeVisibility(args.secretVisibility),
      lastConnectedAt: null,
      createdAt: now,
      updatedAt: now,
      canManageHosts: true,
      canRevealSecrets: true,
      canEditNotes: true,
    };
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
      notes: normalizeNotes(access.host.notes),
      authMode: access.host.authMode ?? access.host.credentialType,
      credentialMode: access.host.credentialMode,
      credentialType: access.host.credentialType,
      secretVisibility: access.host.secretVisibility,
      lastConnectedAt: access.host.lastConnectedAt ?? null,
      sharedCredential: null,
      sharedCredentialConfigured: Boolean(shared?.ciphertext),
      createdAt: access.host.createdAt,
      updatedAt: access.host.updatedAt,
      canManageHosts: hasTeamPermission(access.role, "manage_hosts"),
      canRevealSecrets: hasTeamPermission(access.role, "reveal_secret"),
      canEditNotes: hasTeamPermission(access.role, "edit_notes"),
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
    notes: v.optional(v.string()),
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
    const actorDisplayName = memberDisplayName(access.member, args.clerkUserId);

    await ctx.db.patch(args.hostId, {
      label: args.label.trim(),
      hostname: args.hostname.trim(),
      username: args.username.trim(),
      port: args.port > 0 ? args.port : 22,
      group: args.group.trim(),
      tags: args.tags.map((tag) => tag.trim()).filter(Boolean),
      notes: normalizeNotes(args.notes),
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

    // Shared credential lifecycle:
    // - Delete if credentialType=="none" (no secret is meaningful).
    // - Delete if the caller explicitly passes sharedCredentialCiphertext=null.
    // - Upsert if the caller passes a non-empty ciphertext string. Allowed in
    //   both "shared" mode and "per_member" mode (where the stored row acts as
    //   a fallback used when a member has no personal credential).
    // - Otherwise (undefined/omitted): preserve what's there. This makes mode
    //   transitions non-destructive by default — an admin flipping a host from
    //   shared to per_member no longer wipes the secret.

    if (credentialType === "none") {
      if (existingShared) {
        await ctx.db.delete(existingShared._id);
        await writeAuditEvent(ctx, {
          teamId: access.host.teamId,
          actorClerkUserId: args.clerkUserId,
          actorDisplayName,
          entityType: "team_host_shared_credential",
          entityId: args.hostId,
          eventType: "shared_credential_deleted",
          summary: `Deleted shared credential for ${access.host.label || access.host.hostname}`,
          metadata: {
            hostLabel: access.host.label || access.host.hostname,
            credentialMode,
            credentialType: existingShared.credentialType,
          },
        });
      }
      return { ok: true };
    }

    if (args.sharedCredentialCiphertext === null) {
      if (existingShared) {
        await ctx.db.delete(existingShared._id);
        await writeAuditEvent(ctx, {
          teamId: access.host.teamId,
          actorClerkUserId: args.clerkUserId,
          actorDisplayName,
          entityType: "team_host_shared_credential",
          entityId: args.hostId,
          eventType: "shared_credential_deleted",
          summary: `Deleted shared credential for ${access.host.label || access.host.hostname}`,
          metadata: {
            hostLabel: access.host.label || access.host.hostname,
            credentialMode,
            credentialType: existingShared.credentialType,
          },
        });
      }
      return { ok: true };
    }

    if (
      typeof args.sharedCredentialCiphertext === "string" &&
      args.sharedCredentialCiphertext !== ""
    ) {
      if (existingShared) {
        await ctx.db.patch(existingShared._id, {
          credentialType,
          ciphertext: args.sharedCredentialCiphertext,
          updatedByClerkUserId: args.clerkUserId,
          updatedAt: now,
        });
        await writeAuditEvent(ctx, {
          teamId: access.host.teamId,
          actorClerkUserId: args.clerkUserId,
          actorDisplayName,
          entityType: "team_host_shared_credential",
          entityId: args.hostId,
          eventType: "shared_credential_replaced",
          summary: `Replaced shared credential for ${access.host.label || access.host.hostname}`,
          metadata: {
            hostLabel: access.host.label || access.host.hostname,
            credentialMode,
            credentialType,
          },
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
        await writeAuditEvent(ctx, {
          teamId: access.host.teamId,
          actorClerkUserId: args.clerkUserId,
          actorDisplayName,
          entityType: "team_host_shared_credential",
          entityId: args.hostId,
          eventType: "shared_credential_replaced",
          summary: `Configured shared credential for ${access.host.label || access.host.hostname}`,
          metadata: {
            hostLabel: access.host.label || access.host.hostname,
            credentialMode,
            credentialType,
          },
        });
      }
    }

    // Ciphertext undefined → preserve existingShared row untouched. This is the
    // "Keep as fallback" path from the shared→per_member mode-switch prompt.

    return { ok: true };
  },
});

export const updateNotes = mutation({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
    notes: v.string(),
  },
  handler: async (ctx, args) => {
    await requireHostPermission(ctx, args.hostId, args.clerkUserId, "edit_notes");
    await ctx.db.patch(args.hostId, {
      notes: normalizeNotes(args.notes),
      updatedAt: Date.now(),
    });
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
      updatedAt: credential?.updatedAt ?? null,
      viewerCanEdit: true,
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

export const listCredentialRoster = query({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireHostPermission(ctx, args.hostId, args.clerkUserId, "manage_hosts");
    const members = await ctx.db
      .query("teamMembers")
      .withIndex("by_team", (q) => q.eq("teamId", access.host.teamId))
      .collect();

    const activeMembers = members.filter((member) => member.status === "active");
    const ownerRecord = activeMembers.find((member) => member.clerkUserId === access.team.ownerClerkUserId);
    const roster = ownerRecord
      ? [...activeMembers]
      : [
          ...activeMembers,
          {
            _id: "owner_virtual" as Id<"teamMembers">,
            _creationTime: 0,
            teamId: access.host.teamId,
            clerkUserId: access.team.ownerClerkUserId,
            email: "",
            displayName: access.team.ownerClerkUserId,
            role: "owner",
            status: "active",
            joinedAt: access.team.createdAt,
            createdAt: access.team.createdAt,
            updatedAt: access.team.updatedAt,
          },
        ];

    // Fetch the shared credential row once so we can compute
    // usingSharedFallback per roster entry without N+1 queries.
    const shared = await ctx.db
      .query("teamHostSharedCredentials")
      .withIndex("by_host", (q) => q.eq("hostId", args.hostId))
      .first();
    const sharedFallbackAvailable =
      access.host.credentialMode === "per_member" &&
      access.host.credentialType !== "none" &&
      Boolean(shared?.ciphertext);

    const entries = await Promise.all(
      roster.map(async (member) => {
        const credential = await ctx.db
          .query("teamHostPersonalCredentials")
          .withIndex("by_host_and_user", (q) => q.eq("hostId", args.hostId).eq("clerkUserId", member.clerkUserId))
          .first();

        const role = member.clerkUserId === access.team.ownerClerkUserId ? "owner" : member.role;
        const hasCredential = Boolean(credential?.ciphertext);
        return {
          memberId: member.clerkUserId,
          displayName: memberDisplayName(member, member.clerkUserId),
          email: member.email,
          role,
          isOwner: member.clerkUserId === access.team.ownerClerkUserId,
          isCurrentUser: member.clerkUserId === args.clerkUserId,
          hasCredential,
          credentialType: credential?.credentialType ?? "none",
          username: credential?.username ?? null,
          updatedAt: credential?.updatedAt ?? null,
          usingSharedFallback: !hasCredential && sharedFallbackAvailable,
        };
      }),
    );

    return entries.sort((left, right) => {
      const roleDelta = roleOrder(left.role) - roleOrder(right.role);
      if (roleDelta !== 0) {
        return roleDelta;
      }
      return left.displayName.localeCompare(right.displayName, undefined, { sensitivity: "base" });
    });
  },
});

export const revealSharedCredential = mutation({
  args: {
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireHostPermission(ctx, args.hostId, args.clerkUserId, "reveal_secret");
    if (access.host.credentialMode !== "shared") {
      throw new Error("host_not_shared_credential_mode");
    }

    const shared = await ctx.db
      .query("teamHostSharedCredentials")
      .withIndex("by_host", (q) => q.eq("hostId", args.hostId))
      .first();
    if (!shared) {
      throw new Error("shared_credential_not_configured");
    }

    await writeAuditEvent(ctx, {
      teamId: access.host.teamId,
      actorClerkUserId: args.clerkUserId,
      actorDisplayName: memberDisplayName(access.member, args.clerkUserId),
      entityType: "team_host_shared_credential",
      entityId: args.hostId,
      eventType: "shared_credential_revealed",
      summary: `Revealed shared credential for ${access.host.label || access.host.hostname}`,
      metadata: {
        hostLabel: access.host.label || access.host.hostname,
        credentialMode: access.host.credentialMode,
        credentialType: shared.credentialType,
      },
    });

    return {
      hostId: args.hostId,
      credentialType: shared.credentialType,
      ciphertext: shared.ciphertext,
      updatedAt: shared.updatedAt,
    };
  },
});

export const revealMemberCredential = mutation({
  args: {
    hostId: v.id("teamHosts"),
    memberClerkUserId: v.string(),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireHostPermission(ctx, args.hostId, args.clerkUserId, "reveal_secret");
    if (access.host.credentialMode !== "per_member") {
      throw new Error("host_not_personal_credential_mode");
    }

    const credential = await ctx.db
      .query("teamHostPersonalCredentials")
      .withIndex("by_host_and_user", (q) => q.eq("hostId", args.hostId).eq("clerkUserId", args.memberClerkUserId))
      .first();
    if (!credential) {
      throw new Error("credential_not_configured");
    }

    const targetMember = await getMemberRecord(ctx, access.host.teamId, args.memberClerkUserId);
    await writeAuditEvent(ctx, {
      teamId: access.host.teamId,
      actorClerkUserId: args.clerkUserId,
      actorDisplayName: memberDisplayName(access.member, args.clerkUserId),
      entityType: "team_host_personal_credential",
      entityId: `${args.hostId}:${args.memberClerkUserId}`,
      eventType: "member_credential_revealed",
      targetClerkUserId: args.memberClerkUserId,
      targetDisplayName: memberDisplayName(targetMember, args.memberClerkUserId),
      summary: `Revealed ${memberDisplayName(targetMember, args.memberClerkUserId)}'s credential for ${access.host.label || access.host.hostname}`,
      metadata: {
        hostLabel: access.host.label || access.host.hostname,
        credentialMode: access.host.credentialMode,
        credentialType: credential.credentialType,
      },
    });

    return {
      hostId: args.hostId,
      memberClerkUserId: args.memberClerkUserId,
      credentialType: credential.credentialType,
      username: credential.username ?? null,
      ciphertext: credential.ciphertext,
      updatedAt: credential.updatedAt,
    };
  },
});

export const deleteMemberCredentialAsAdmin = mutation({
  args: {
    hostId: v.id("teamHosts"),
    memberClerkUserId: v.string(),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireHostPermission(ctx, args.hostId, args.clerkUserId, "reveal_secret");
    if (access.host.credentialMode !== "per_member") {
      throw new Error("host_not_personal_credential_mode");
    }

    const credential = await ctx.db
      .query("teamHostPersonalCredentials")
      .withIndex("by_host_and_user", (q) => q.eq("hostId", args.hostId).eq("clerkUserId", args.memberClerkUserId))
      .first();
    if (!credential) {
      return { ok: true };
    }

    await ctx.db.delete(credential._id);

    const targetMember = await getMemberRecord(ctx, access.host.teamId, args.memberClerkUserId);
    await writeAuditEvent(ctx, {
      teamId: access.host.teamId,
      actorClerkUserId: args.clerkUserId,
      actorDisplayName: memberDisplayName(access.member, args.clerkUserId),
      entityType: "team_host_personal_credential",
      entityId: `${args.hostId}:${args.memberClerkUserId}`,
      eventType: "member_credential_deleted",
      targetClerkUserId: args.memberClerkUserId,
      targetDisplayName: memberDisplayName(targetMember, args.memberClerkUserId),
      summary: `Deleted ${memberDisplayName(targetMember, args.memberClerkUserId)}'s credential for ${access.host.label || access.host.hostname}`,
      metadata: {
        hostLabel: access.host.label || access.host.hostname,
        credentialMode: access.host.credentialMode,
        credentialType: credential.credentialType,
      },
    });

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
        usedSharedFallback: false,
      };
    }

    // per_member mode: prefer the caller's personal credential, then silently
    // fall back to the shared credential if one is stored on this host, then
    // error. "shared row present while mode is per_member" is the implicit
    // opt-in for the fallback feature (no separate flag).
    const personal = await ctx.db
      .query("teamHostPersonalCredentials")
      .withIndex("by_host_and_user", (q) =>
        q.eq("hostId", args.hostId).eq("clerkUserId", args.clerkUserId),
      )
      .first();

    if (personal && personal.ciphertext) {
      return {
        hostId: access.host._id,
        teamId: access.host.teamId,
        label: access.host.label,
        hostname: access.host.hostname,
        username: personal.username ?? access.host.username,
        port: access.host.port,
        credentialMode: access.host.credentialMode,
        credentialType: personal.credentialType,
        secret: personal.ciphertext,
        usedSharedFallback: false,
      };
    }

    if (access.host.credentialType === "none") {
      // No credential type configured — nothing to resolve, but return a valid
      // shape so the CLI can still connect (password-prompt / key-agent flow).
      return {
        hostId: access.host._id,
        teamId: access.host.teamId,
        label: access.host.label,
        hostname: access.host.hostname,
        username: personal?.username ?? access.host.username,
        port: access.host.port,
        credentialMode: access.host.credentialMode,
        credentialType: access.host.credentialType,
        secret: "",
        usedSharedFallback: false,
      };
    }

    const sharedFallback = await ctx.db
      .query("teamHostSharedCredentials")
      .withIndex("by_host", (q) => q.eq("hostId", args.hostId))
      .first();

    if (sharedFallback && sharedFallback.ciphertext) {
      return {
        hostId: access.host._id,
        teamId: access.host.teamId,
        label: access.host.label,
        hostname: access.host.hostname,
        username: personal?.username ?? access.host.username,
        port: access.host.port,
        credentialMode: access.host.credentialMode,
        credentialType: sharedFallback.credentialType,
        secret: sharedFallback.ciphertext,
        usedSharedFallback: true,
      };
    }

    throw new Error("personal_credential_not_configured");
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
