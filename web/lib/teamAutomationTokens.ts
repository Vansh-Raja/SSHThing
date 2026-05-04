import { randomBytes } from "crypto";

import { hashToken } from "./tokens";

export function createTeamAutomationTokenMaterial() {
  const tokenId = randomBytes(10).toString("base64url");
  const secret = randomBytes(32).toString("base64url");
  return {
    tokenId,
    secret,
    rawToken: `stt_${tokenId}_${secret}`,
    tokenHash: hashToken(secret),
  };
}

export function parseTeamAutomationToken(rawToken: string) {
  const parts = rawToken.trim().split("_");
  if (parts.length !== 3 || parts[0] !== "stt" || !parts[1] || !parts[2]) {
    throw new Error("invalid_team_token_format");
  }
  return {
    tokenId: parts[1],
    tokenHash: hashToken(parts[2]),
  };
}
