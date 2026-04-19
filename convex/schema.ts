import { defineSchema, defineTable } from "convex/server";
import { v } from "convex/values";

export default defineSchema({
  teams: defineTable({
    ownerClerkUserId: v.string(),
    name: v.string(),
    slug: v.string(),
    displayOrder: v.number(),
    status: v.string(),
    createdAt: v.number(),
    updatedAt: v.number(),
  })
    .index("by_owner_and_display_order", ["ownerClerkUserId", "displayOrder"])
    .index("by_owner_and_slug", ["ownerClerkUserId", "slug"]),

  teamMembers: defineTable({
    teamId: v.id("teams"),
    clerkUserId: v.string(),
    email: v.string(),
    displayName: v.string(),
    role: v.string(),
    status: v.string(),
    joinedAt: v.optional(v.number()),
    createdAt: v.number(),
    updatedAt: v.number(),
  })
    .index("by_team", ["teamId"])
    .index("by_team_and_user", ["teamId", "clerkUserId"])
    .index("by_user", ["clerkUserId"]),

  teamHosts: defineTable({
    teamId: v.id("teams"),
    label: v.string(),
    hostname: v.string(),
    username: v.string(),
    port: v.number(),
    group: v.string(),
    tags: v.array(v.string()),
    authMode: v.optional(v.string()),
    lastConnectedAt: v.optional(v.number()),
    createdAt: v.number(),
    updatedAt: v.number(),
  }).index("by_team", ["teamId"]),

  workspaces: defineTable({
    clerkOrganizationId: v.string(),
    name: v.string(),
    slug: v.string(),
    status: v.string(),
    createdByUserId: v.string(),
    createdAt: v.number(),
  })
    .index("by_clerk_org", ["clerkOrganizationId"])
    .index("by_slug", ["slug"]),

  workspaceMembers: defineTable({
    workspaceId: v.id("workspaces"),
    clerkUserId: v.string(),
    email: v.string(),
    displayName: v.string(),
    workspaceRole: v.string(),
    status: v.string(),
    invitationId: v.optional(v.string()),
    joinedAt: v.optional(v.number()),
    lastSeenAt: v.optional(v.number()),
    createdAt: v.number(),
    updatedAt: v.number(),
  })
    .index("by_workspace", ["workspaceId"])
    .index("by_workspace_user", ["workspaceId", "clerkUserId"])
    .index("by_clerk_user", ["clerkUserId"]),

  vaults: defineTable({
    workspaceId: v.id("workspaces"),
    name: v.string(),
    slug: v.string(),
    description: v.string(),
    createdAt: v.number(),
    updatedAt: v.number(),
  })
    .index("by_workspace", ["workspaceId"])
    .index("by_workspace_slug", ["workspaceId", "slug"]),

  vaultMembers: defineTable({
    workspaceId: v.id("workspaces"),
    vaultId: v.id("vaults"),
    clerkUserId: v.string(),
    vaultRole: v.string(),
    status: v.string(),
    createdAt: v.number(),
    updatedAt: v.number(),
  })
    .index("by_vault", ["vaultId"])
    .index("by_vault_user", ["vaultId", "clerkUserId"])
    .index("by_workspace_user", ["workspaceId", "clerkUserId"]),

  resources: defineTable({
    vaultId: v.id("vaults"),
    label: v.string(),
    group: v.string(),
    tags: v.array(v.string()),
    hostname: v.string(),
    username: v.string(),
    port: v.number(),
    shareMode: v.string(),
    notes: v.array(v.string()),
    createdBy: v.string(),
    createdAt: v.number(),
    updatedBy: v.string(),
    updatedAt: v.number(),
  }).index("by_vault", ["vaultId"]),

  cliAuthSessions: defineTable({
    deviceName: v.string(),
    deviceCode: v.string(),
    pollSecret: v.string(),
    status: v.string(),
    requestedAt: v.number(),
    completedAt: v.optional(v.number()),
    clerkUserId: v.optional(v.string()),
    workspaceId: v.optional(v.id("workspaces")),
    teamId: v.optional(v.id("teams")),
    expiresAt: v.number(),
  }).index("by_device_code", ["deviceCode"]),

  tuiSessions: defineTable({
    clerkUserId: v.string(),
    workspaceId: v.optional(v.id("workspaces")),
    teamId: v.optional(v.id("teams")),
    accessTokenHash: v.string(),
    refreshTokenHash: v.string(),
    deviceName: v.string(),
    accessExpiresAt: v.number(),
    refreshExpiresAt: v.number(),
    lastSeenAt: v.number(),
    revokedAt: v.optional(v.number()),
    createdAt: v.number(),
  })
    .index("by_access_hash", ["accessTokenHash"])
    .index("by_refresh_hash", ["refreshTokenHash"])
    .index("by_workspace", ["workspaceId"]),

  auditEvents: defineTable({
    workspaceId: v.id("workspaces"),
    actorUserId: v.string(),
    eventType: v.string(),
    targetType: v.string(),
    targetId: v.string(),
    metadata: v.any(),
    createdAt: v.number(),
  }).index("by_workspace", ["workspaceId"]),
});
