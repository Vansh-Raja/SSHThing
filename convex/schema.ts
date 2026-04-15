import { defineSchema, defineTable } from "convex/server";
import { v } from "convex/values";

export default defineSchema({
  teams: defineTable({
    clerkOrganizationId: v.string(),
    name: v.string(),
    slug: v.string(),
    status: v.string(),
    billingStatus: v.optional(v.string()),
    createdByUserId: v.string(),
    createdAt: v.number()
  }).index("by_clerk_org", ["clerkOrganizationId"]),

  teamMembers: defineTable({
    teamId: v.id("teams"),
    clerkUserId: v.string(),
    email: v.string(),
    displayName: v.string(),
    role: v.string(),
    status: v.string(),
    joinedAt: v.optional(v.number()),
    lastSeenAt: v.optional(v.number())
  })
    .index("by_team", ["teamId"])
    .index("by_team_user", ["teamId", "clerkUserId"])
    .index("by_clerk_user", ["clerkUserId"]),

  teamHosts: defineTable({
    teamId: v.id("teams"),
    label: v.string(),
    group: v.optional(v.string()),
    tags: v.array(v.string()),
    hostname: v.string(),
    username: v.string(),
    port: v.number(),
    shareMode: v.string(),
    notes: v.array(v.string()),
    lastActivityAt: v.optional(v.number()),
    createdBy: v.string(),
    createdAt: v.number(),
    updatedBy: v.string(),
    updatedAt: v.number(),
    rotationRecommended: v.optional(v.boolean()),
    rotationReason: v.optional(v.string())
  }).index("by_team", ["teamId"]),

  teamHostSecrets: defineTable({
    teamHostId: v.id("teamHosts"),
    secretType: v.string(),
    keyType: v.string(),
    encryptedSecret: v.string(),
    encryptionVersion: v.number(),
    createdBy: v.string(),
    createdAt: v.number(),
    updatedBy: v.string(),
    updatedAt: v.number(),
    rotationRecommended: v.optional(v.boolean()),
    rotationReason: v.optional(v.string())
  }).index("by_team_host", ["teamHostId"]),

  teamInvites: defineTable({
    teamId: v.id("teams"),
    email: v.string(),
    role: v.string(),
    invitedBy: v.string(),
    createdAt: v.number(),
    status: v.string()
  }).index("by_team", ["teamId"]),

  cliAuthSessions: defineTable({
    status: v.string(),
    requestedAt: v.number(),
    completedAt: v.optional(v.number()),
    userId: v.optional(v.string()),
    teamId: v.optional(v.id("teams")),
    deviceName: v.string(),
    deviceCode: v.string(),
    pollSecret: v.string()
  }).index("by_device_code", ["deviceCode"]),

  cliAccessGrants: defineTable({
    teamId: v.id("teams"),
    teamHostId: v.id("teamHosts"),
    userId: v.string(),
    issuedAt: v.number(),
    expiresAt: v.number(),
    grantType: v.string(),
    redeemedAt: v.optional(v.number()),
    revokedAt: v.optional(v.number())
  }).index("by_team_host", ["teamHostId"]).index("by_user", ["userId"]),

  auditEvents: defineTable({
    teamId: v.id("teams"),
    actorUserId: v.string(),
    eventType: v.string(),
    targetType: v.string(),
    targetId: v.string(),
    metadata: v.any(),
    createdAt: v.number()
  }).index("by_team", ["teamId"])
});
