import type { Doc, Id } from "./_generated/dataModel";
import type { MutationCtx, QueryCtx } from "./_generated/server";
import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

import { hasTeamPermission, requireTeamPermission } from "./teamAccess";

type TeamCtx = QueryCtx | MutationCtx;

function normalizeStatus(value: string): "completed" | "failed" {
  return value === "completed" ? "completed" : "failed";
}

function trimCommand(value: string): string {
  const trimmed = value.trim();
  return trimmed.length > 4096 ? trimmed.slice(0, 4096) : trimmed;
}

function tokenIsUsable(token: Doc<"teamAutomationTokens">): boolean {
  const now = Date.now();
  if (token.status !== "active" || token.revokedAt) return false;
  if (token.expiresAt && token.expiresAt <= now) return false;
  if (token.maxUses && token.maxUses > 0 && token.useCount >= token.maxUses) return false;
  return true;
}

async function getTokenByID(ctx: TeamCtx, tokenId: string) {
  return ctx.db
    .query("teamAutomationTokens")
    .withIndex("by_token_id", (q) => q.eq("tokenId", tokenId))
    .first();
}

async function writeTeamAuditEvent(
  ctx: MutationCtx,
  args: {
    teamId: Id<"teams">;
    actorClerkUserId: string;
    actorDisplayName: string;
    entityId: string;
    eventType: string;
    summary: string;
    hostLabel?: string;
    tokenName?: string;
    command?: string;
    status?: string;
    exitCode?: number;
  },
) {
  await ctx.db.insert("teamAuditEvents", {
    teamId: args.teamId,
    actorClerkUserId: args.actorClerkUserId,
    actorDisplayName: args.actorDisplayName,
    entityType: "team_automation_token",
    entityId: args.entityId,
    eventType: args.eventType,
    summary: args.summary,
    metadata: {
      hostLabel: args.hostLabel,
      tokenName: args.tokenName,
      command: args.command,
      status: args.status,
      exitCode: args.exitCode,
    },
    createdAt: Date.now(),
  });
}

async function logExecution(
  ctx: MutationCtx,
  token: Doc<"teamAutomationTokens">,
  args: {
    hostId?: Id<"teamHosts">;
    hostLabel?: string;
    command: string;
    clientDevice?: string;
    status: string;
    error?: string;
  },
) {
  const now = Date.now();
  return ctx.db.insert("teamAutomationTokenExecutions", {
    teamId: token.teamId,
    tokenDocId: token._id,
    tokenId: token.tokenId,
    tokenName: token.name,
    createdByClerkUserId: token.createdByClerkUserId,
    createdByDisplayName: token.createdByDisplayName,
    hostId: args.hostId,
    hostLabel: args.hostLabel,
    command: trimCommand(args.command),
    clientDevice: args.clientDevice?.trim() || undefined,
    status: args.status,
    error: args.error?.trim() || undefined,
    startedAt: now,
    finishedAt: args.status === "denied" ? now : undefined,
  });
}

export const listForTeam = query({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const access = await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_hosts");
    if (!hasTeamPermission(access.role, "manage_hosts")) {
      throw new Error("forbidden");
    }
    const tokens = await ctx.db
      .query("teamAutomationTokens")
      .withIndex("by_team", (q) => q.eq("teamId", args.teamId))
      .collect();
    const rows = await Promise.all(
      tokens.map(async (token) => {
        const hosts = await ctx.db
          .query("teamAutomationTokenHosts")
          .withIndex("by_token", (q) => q.eq("tokenDocId", token._id))
          .collect();
        return {
          id: token._id,
          teamId: token.teamId,
          tokenId: token.tokenId,
          name: token.name,
          status: token.status,
          hostCount: hosts.length,
          hosts: hosts.map((host) => ({
            hostId: host.hostId,
            hostLabel: host.hostLabel,
          })),
          createdByClerkUserId: token.createdByClerkUserId,
          createdByDisplayName: token.createdByDisplayName,
          createdAt: token.createdAt,
          updatedAt: token.updatedAt,
          lastUsedAt: token.lastUsedAt ?? null,
          useCount: token.useCount,
          expiresAt: token.expiresAt ?? null,
          maxUses: token.maxUses ?? null,
          revokedAt: token.revokedAt ?? null,
        };
      }),
    );
    return rows.sort((left, right) => right.createdAt - left.createdAt);
  },
});

export const create = mutation({
  args: {
    teamId: v.id("teams"),
    clerkUserId: v.string(),
    actorDisplayName: v.string(),
    name: v.string(),
    tokenId: v.string(),
    tokenHash: v.string(),
    hostIds: v.array(v.id("teamHosts")),
    expiresAt: v.optional(v.union(v.number(), v.null())),
    maxUses: v.optional(v.union(v.number(), v.null())),
  },
  handler: async (ctx, args) => {
    await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_hosts");
    const name = args.name.trim();
    if (!name) throw new Error("token_name_required");
    if (name.length > 80) throw new Error("token_name_too_long");
    if (!args.tokenId.trim() || !args.tokenHash.trim()) throw new Error("invalid_token_material");
    if (args.hostIds.length === 0) throw new Error("token_hosts_required");

    const existing = await getTokenByID(ctx, args.tokenId);
    if (existing) throw new Error("token_id_collision");

    const now = Date.now();
    const tokenDocId = await ctx.db.insert("teamAutomationTokens", {
      teamId: args.teamId,
      name,
      tokenId: args.tokenId.trim(),
      tokenHash: args.tokenHash.trim(),
      createdByClerkUserId: args.clerkUserId,
      createdByDisplayName: args.actorDisplayName.trim() || args.clerkUserId,
      status: "active",
      expiresAt: args.expiresAt ?? undefined,
      maxUses: args.maxUses ?? undefined,
      useCount: 0,
      createdAt: now,
      updatedAt: now,
    });

    const uniqueHostIds = Array.from(new Set(args.hostIds));
    for (const hostId of uniqueHostIds) {
      const host = await ctx.db.get(hostId);
      if (!host || host.teamId !== args.teamId) {
        throw new Error("invalid_token_host");
      }
      await ctx.db.insert("teamAutomationTokenHosts", {
        tokenDocId,
        teamId: args.teamId,
        hostId,
        hostLabel: host.label || host.hostname,
        createdAt: now,
      });
    }

    await writeTeamAuditEvent(ctx, {
      teamId: args.teamId,
      actorClerkUserId: args.clerkUserId,
      actorDisplayName: args.actorDisplayName,
      entityId: tokenDocId,
      eventType: "team_token_created",
      summary: `Created team automation token ${name}`,
      tokenName: name,
    });

    return { id: tokenDocId, tokenId: args.tokenId, name, createdAt: now };
  },
});

export const revoke = mutation({
  args: {
    teamId: v.id("teams"),
    tokenDocId: v.id("teamAutomationTokens"),
    clerkUserId: v.string(),
    actorDisplayName: v.string(),
  },
  handler: async (ctx, args) => {
    await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_hosts");
    const token = await ctx.db.get(args.tokenDocId);
    if (!token || token.teamId !== args.teamId) throw new Error("token_not_found");
    if (token.status === "revoked") return { ok: true };
    const now = Date.now();
    await ctx.db.patch(args.tokenDocId, {
      status: "revoked",
      revokedAt: now,
      updatedAt: now,
    });
    await writeTeamAuditEvent(ctx, {
      teamId: args.teamId,
      actorClerkUserId: args.clerkUserId,
      actorDisplayName: args.actorDisplayName,
      entityId: args.tokenDocId,
      eventType: "team_token_revoked",
      summary: `Revoked team automation token ${token.name}`,
      tokenName: token.name,
    });
    return { ok: true };
  },
});

export const deleteRevoked = mutation({
  args: {
    teamId: v.id("teams"),
    tokenDocId: v.id("teamAutomationTokens"),
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    await requireTeamPermission(ctx, args.teamId, args.clerkUserId, "manage_hosts");
    const token = await ctx.db.get(args.tokenDocId);
    if (!token || token.teamId !== args.teamId) throw new Error("token_not_found");
    if (token.status !== "revoked") throw new Error("token_must_be_revoked");
    const grants = await ctx.db
      .query("teamAutomationTokenHosts")
      .withIndex("by_token", (q) => q.eq("tokenDocId", args.tokenDocId))
      .collect();
    for (const grant of grants) {
      await ctx.db.delete(grant._id);
    }
    await ctx.db.delete(args.tokenDocId);
    return { ok: true };
  },
});

export const resolveForExecution = mutation({
  args: {
    tokenId: v.string(),
    tokenHash: v.string(),
    teamId: v.optional(v.union(v.id("teams"), v.null())),
    target: v.optional(v.string()),
    targetId: v.optional(v.union(v.id("teamHosts"), v.null())),
    command: v.string(),
    clientDevice: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const token = await getTokenByID(ctx, args.tokenId.trim());
    if (!token || token.tokenHash !== args.tokenHash.trim()) {
      throw new Error("invalid_team_token");
    }
    const command = trimCommand(args.command);
    if (!command) {
      await logExecution(ctx, token, {
        command,
        clientDevice: args.clientDevice,
        status: "denied",
        error: "command_required",
      });
      throw new Error("command_required");
    }
    if (args.teamId && token.teamId !== args.teamId) {
      await logExecution(ctx, token, {
        command,
        clientDevice: args.clientDevice,
        status: "denied",
        error: "team_mismatch",
      });
      throw new Error("team_mismatch");
    }
    if (!tokenIsUsable(token)) {
      await logExecution(ctx, token, {
        command,
        clientDevice: args.clientDevice,
        status: "denied",
        error: "team_token_inactive",
      });
      throw new Error("team_token_inactive");
    }

    const grants = await ctx.db
      .query("teamAutomationTokenHosts")
      .withIndex("by_token", (q) => q.eq("tokenDocId", token._id))
      .collect();

    let grant: Doc<"teamAutomationTokenHosts"> | undefined;
    if (args.targetId) {
      grant = grants.find((candidate) => candidate.hostId === args.targetId);
    } else {
      const target = args.target?.trim() ?? "";
      if (!target) {
        await logExecution(ctx, token, {
          command,
          clientDevice: args.clientDevice,
          status: "denied",
          error: "target_required",
        });
        throw new Error("target_required");
      }
      const matches = grants.filter((candidate) => candidate.hostLabel === target);
      if (matches.length > 1) {
        await logExecution(ctx, token, {
          command,
          clientDevice: args.clientDevice,
          status: "denied",
          error: "target_label_ambiguous",
        });
        throw new Error("target_label_ambiguous");
      }
      grant = matches[0];
    }

    if (!grant) {
      await logExecution(ctx, token, {
        command,
        clientDevice: args.clientDevice,
        status: "denied",
        error: "target_not_allowed_by_team_token",
      });
      throw new Error("target_not_allowed_by_team_token");
    }

    const host = await ctx.db.get(grant.hostId);
    if (!host || host.teamId !== token.teamId) {
      await logExecution(ctx, token, {
        hostId: grant.hostId,
        hostLabel: grant.hostLabel,
        command,
        clientDevice: args.clientDevice,
        status: "denied",
        error: "host_not_found",
      });
      throw new Error("host_not_found");
    }

    let credentialType = host.credentialType;
    let username = host.username;
    let ciphertext = "";
    let usedSharedFallback = false;

    if (host.credentialMode === "shared") {
      const shared = await ctx.db
        .query("teamHostSharedCredentials")
        .withIndex("by_host", (q) => q.eq("hostId", host._id))
        .first();
      if (host.credentialType !== "none" && !shared?.ciphertext) {
        await logExecution(ctx, token, {
          hostId: host._id,
          hostLabel: host.label || host.hostname,
          command,
          clientDevice: args.clientDevice,
          status: "denied",
          error: "shared_credential_not_configured",
        });
        throw new Error("shared_credential_not_configured");
      }
      credentialType = shared?.credentialType ?? host.credentialType;
      ciphertext = shared?.ciphertext ?? "";
    } else {
      const personal = await ctx.db
        .query("teamHostPersonalCredentials")
        .withIndex("by_host_and_user", (q) =>
          q.eq("hostId", host._id).eq("clerkUserId", token.createdByClerkUserId),
        )
        .first();
      if (personal?.ciphertext) {
        credentialType = personal.credentialType;
        username = personal.username ?? host.username;
        ciphertext = personal.ciphertext;
      } else {
        const sharedFallback = await ctx.db
          .query("teamHostSharedCredentials")
          .withIndex("by_host", (q) => q.eq("hostId", host._id))
          .first();
        if (sharedFallback?.ciphertext) {
          credentialType = sharedFallback.credentialType;
          ciphertext = sharedFallback.ciphertext;
          usedSharedFallback = true;
        } else if (host.credentialType !== "none") {
          await logExecution(ctx, token, {
            hostId: host._id,
            hostLabel: host.label || host.hostname,
            command,
            clientDevice: args.clientDevice,
            status: "denied",
            error: "personal_credential_not_configured",
          });
          throw new Error("personal_credential_not_configured");
        }
      }
    }

    const executionId = await logExecution(ctx, token, {
      hostId: host._id,
      hostLabel: host.label || host.hostname,
      command,
      clientDevice: args.clientDevice,
      status: "allowed",
    });
    const now = Date.now();
    await ctx.db.patch(token._id, {
      useCount: token.useCount + 1,
      lastUsedAt: now,
      updatedAt: now,
    });
    await ctx.db.patch(host._id, {
      lastConnectedAt: now,
      updatedAt: now,
    });

    return {
      executionId,
      host: {
        hostId: host._id,
        teamId: host.teamId,
        label: host.label,
        hostname: host.hostname,
        username,
        port: host.port,
        credentialMode: host.credentialMode,
        credentialType,
        secret: ciphertext,
        usedSharedFallback,
      },
    };
  },
});

export const finishExecution = mutation({
  args: {
    tokenId: v.string(),
    tokenHash: v.string(),
    executionId: v.id("teamAutomationTokenExecutions"),
    status: v.string(),
    exitCode: v.optional(v.union(v.number(), v.null())),
    error: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const token = await getTokenByID(ctx, args.tokenId.trim());
    if (!token || token.tokenHash !== args.tokenHash.trim()) {
      throw new Error("invalid_team_token");
    }
    const execution = await ctx.db.get(args.executionId);
    if (!execution || execution.tokenDocId !== token._id) {
      throw new Error("execution_not_found");
    }
    const status = normalizeStatus(args.status);
    const error = args.error?.trim();
    await ctx.db.patch(args.executionId, {
      status,
      exitCode: args.exitCode ?? undefined,
      error: error || undefined,
      finishedAt: Date.now(),
    });
    await writeTeamAuditEvent(ctx, {
      teamId: execution.teamId,
      actorClerkUserId: token.createdByClerkUserId,
      actorDisplayName: token.createdByDisplayName,
      entityId: token._id,
      eventType: "team_token_command_" + status,
      summary: `${token.name} ran ${execution.command} on ${execution.hostLabel ?? "unknown host"}`,
      hostLabel: execution.hostLabel,
      tokenName: token.name,
      command: execution.command,
      status,
      exitCode: args.exitCode ?? undefined,
    });
    return { ok: true };
  },
});
