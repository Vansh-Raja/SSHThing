import { auth, clerkClient, currentUser } from "@clerk/nextjs/server";

import { convexApi, convexMutation, convexQuery } from "./convex";
import { getBrowserBaseURL } from "./env";
import { createAccessToken, createClaimCode, createDeviceCode, createPollSecret, createRefreshToken, hashToken } from "./tokens";

type TuiSessionRecord = {
  _id: string;
  clerkUserId: string;
  deviceName: string;
  accessExpiresAt: number;
  refreshExpiresAt: number;
  revokedAt?: number | null;
};

export type BrowserIdentity = {
  userId: string;
  email: string;
  displayName: string;
};

export type RequestActor = {
  clerkUserId: string;
  email: string;
  displayName: string;
  source: "browser" | "bearer";
};

export async function requireBrowserIdentity(): Promise<BrowserIdentity> {
  const { isAuthenticated, userId } = await auth();
  if (!isAuthenticated || !userId) {
    throw new Error("not_authenticated");
  }

  const user = await currentUser();
  if (!user) {
    throw new Error("missing_user");
  }

  const primaryEmail = user.emailAddresses.find((email) => email.id === user.primaryEmailAddressId)?.emailAddress
    ?? user.emailAddresses[0]?.emailAddress
    ?? "";

  return {
    userId,
    email: primaryEmail.toLowerCase(),
    displayName: [user.firstName, user.lastName].filter(Boolean).join(" ") || user.username || primaryEmail || userId,
  };
}

async function getClerkUserSummary(clerkUserId: string): Promise<BrowserIdentity> {
  const client = await clerkClient();
  const user = await client.users.getUser(clerkUserId);
  const primaryEmail = user.emailAddresses.find((email) => email.id === user.primaryEmailAddressId)?.emailAddress
    ?? user.emailAddresses[0]?.emailAddress
    ?? "";

  return {
    userId: user.id,
    email: primaryEmail.toLowerCase(),
    displayName: [user.firstName, user.lastName].filter(Boolean).join(" ") || user.username || primaryEmail || user.id,
  };
}

export async function buildCliAuthStart(deviceName: string, headless = false) {
  const started = await convexMutation<{ sessionId: string; deviceCode: string; pollSecret: string; expiresAt: number }>(
    convexApi.sessions.startCliAuth,
    {
      deviceName,
      deviceCode: createDeviceCode(),
      pollSecret: createPollSecret(),
      ttlSeconds: 600,
    },
  );

  const authUrl = new URL("/cli-auth/complete", getBrowserBaseURL());
  authUrl.searchParams.set("session", started.sessionId);
  authUrl.searchParams.set("code", started.deviceCode);
  // Headless flow: the complete page renders a paste-back claim code
  // instead of telling the user to return to an already-polling TUI.
  if (headless) {
    authUrl.searchParams.set("mode", "headless");
  }

  return {
    authUrl: authUrl.toString(),
    deviceCode: started.deviceCode,
    sessionId: started.sessionId,
    pollSecret: started.pollSecret,
    pollIntervalSeconds: 2,
    expiresAt: started.expiresAt,
  };
}

/**
 * Mark a CLI auth session complete. For headless sign-ins, also mints a
 * one-time claim code, stores it on the session, and returns it so the
 * browser can display it for paste-back into the TUI.
 */
export async function completeCliAuth(
  sessionId: string,
  deviceCode?: string | null,
  headless = false,
): Promise<{ ok: boolean; claimCode: string | null }> {
  const identity = await requireBrowserIdentity();
  const claimCode = headless ? createClaimCode() : null;

  await convexMutation<{ ok: boolean }>(convexApi.sessions.completeCliAuth, {
    sessionId,
    deviceCode: deviceCode ?? "",
    clerkUserId: identity.userId,
    ...(claimCode ? { claimCode } : {}),
  });

  return { ok: true, claimCode };
}

/**
 * Mint a fresh access/refresh token pair for a TUI session and persist
 * the hashed tokens. Shared by the poll-based and headless claim flows.
 */
async function mintTuiTokens(clerkUserId: string, deviceName: string) {
  const user = await getClerkUserSummary(clerkUserId);
  const accessToken = createAccessToken();
  const refreshToken = createRefreshToken();
  const session = await convexMutation<{
    sessionId: string;
    accessExpiresAt: number;
    refreshExpiresAt: number;
  }>(convexApi.sessions.createTuiSession, {
    clerkUserId,
    deviceName: deviceName || "SSHThing TUI",
    accessTokenHash: hashToken(accessToken),
    refreshTokenHash: hashToken(refreshToken),
    accessTtlSeconds: 900,
    refreshTtlSeconds: 86400 * 30,
  });

  return {
    status: "completed" as const,
    accessToken,
    refreshToken,
    expiresAt: session.accessExpiresAt,
    user: {
      id: clerkUserId,
      name: user.displayName,
      email: user.email,
    },
  };
}

/**
 * Headless paste-back: exchange the browser-displayed claim code (plus
 * the TUI-held pollSecret) for a token pair.
 */
export async function claimCliAuth(sessionId: string, pollSecret: string, claimCode: string) {
  const record = await convexMutation<{ clerkUserId: string; deviceName: string }>(
    convexApi.sessions.claimCliAuthSession,
    {
      sessionId,
      pollSecret,
      claimCode: claimCode.trim(),
    },
  );

  return mintTuiTokens(record.clerkUserId, record.deviceName);
}

export async function pollCliAuth(sessionId: string, pollSecret: string) {
  const record = await convexQuery<{
    status: string;
    clerkUserId?: string | null;
    deviceName?: string | null;
    expiresAt: number;
    completedAt?: number | null;
  }>(convexApi.sessions.getCliAuthStatus, {
    sessionId,
    pollSecret,
  });

  if (record.status !== "completed" || !record.clerkUserId) {
    return {
      status: record.status,
      expiresAt: record.expiresAt,
    };
  }

  return mintTuiTokens(record.clerkUserId, record.deviceName ?? "SSHThing TUI");
}

export async function refreshTuiAccess(refreshToken: string) {
  const record = await convexQuery<TuiSessionRecord | null>(convexApi.sessions.getTuiSessionByRefreshHash, {
    refreshTokenHash: hashToken(refreshToken),
  });

  if (!record || record.revokedAt || record.refreshExpiresAt <= Date.now()) {
    throw new Error("invalid_refresh_token");
  }

  const accessToken = createAccessToken();
  const updated = await convexMutation<{ accessExpiresAt: number }>(convexApi.sessions.rotateAccessToken, {
    sessionId: record._id,
    accessTokenHash: hashToken(accessToken),
    accessTtlSeconds: 900,
  });

  return {
    accessToken,
    expiresAt: updated.accessExpiresAt,
  };
}

export async function revokeTuiSession(refreshToken: string) {
  const record = await convexQuery<TuiSessionRecord | null>(convexApi.sessions.getTuiSessionByRefreshHash, {
    refreshTokenHash: hashToken(refreshToken),
  });
  if (!record) {
    return { ok: true };
  }
  await convexMutation<{ ok: boolean }>(convexApi.sessions.revokeTuiSession, {
    sessionId: record._id,
  });
  return { ok: true };
}

export async function getTuiSessionFromBearer(authHeader: string | null) {
  const value = authHeader?.trim() ?? "";
  if (!value.startsWith("Bearer ")) {
    throw new Error("missing_bearer_token");
  }
  const accessToken = value.slice("Bearer ".length).trim();
  if (!accessToken) {
    throw new Error("missing_bearer_token");
  }

  const record = await convexQuery<TuiSessionRecord | null>(convexApi.sessions.getTuiSessionByAccessHash, {
    accessTokenHash: hashToken(accessToken),
  });
  if (!record || record.revokedAt || record.accessExpiresAt <= Date.now()) {
    throw new Error("invalid_access_token");
  }

  await convexMutation<{ ok: boolean }>(convexApi.sessions.markTuiSessionSeen, {
    sessionId: record._id,
  });

  return record;
}

export async function getSessionContextFromBearer(authHeader: string | null) {
  const session = await getTuiSessionFromBearer(authHeader);
  return { session };
}

export async function getActorFromRequest(authHeader: string | null): Promise<RequestActor> {
  const trimmed = authHeader?.trim() ?? "";
  if (trimmed.startsWith("Bearer ")) {
    const session = await getTuiSessionFromBearer(authHeader);
    const user = await getClerkUserSummary(session.clerkUserId);
    return {
      clerkUserId: user.userId,
      email: user.email,
      displayName: user.displayName,
      source: "bearer",
    };
  }

  const identity = await requireBrowserIdentity();
  return {
    clerkUserId: identity.userId,
    email: identity.email,
    displayName: identity.displayName,
    source: "browser",
  };
}

export function buildInviteLink(inviteId: string, token: string): string {
  const inviteUrl = new URL(`/teams/invites/${inviteId}`, getBrowserBaseURL());
  inviteUrl.searchParams.set("token", token);
  return inviteUrl.toString();
}
