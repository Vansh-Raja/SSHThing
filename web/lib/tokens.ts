import { createHash, randomBytes, timingSafeEqual } from "crypto";

function encodeToken(prefix: string): string {
  return `${prefix}${randomBytes(32).toString("base64url")}`;
}

export function createAccessToken(): string {
  return encodeToken("ssta_");
}

export function createRefreshToken(): string {
  return encodeToken("sstr_");
}

export function createDeviceCode(): string {
  const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
  const bytes = randomBytes(8);
  let out = "";
  for (let i = 0; i < bytes.length; i++) {
    out += alphabet[bytes[i] % alphabet.length];
  }
  return out;
}

export function createPollSecret(): string {
  return randomBytes(24).toString("base64url");
}

export function hashToken(token: string): string {
  return createHash("sha256").update(token, "utf8").digest("hex");
}

export function safeTokenEquals(a: string, b: string): boolean {
  const aBuf = Buffer.from(a, "utf8");
  const bBuf = Buffer.from(b, "utf8");
  if (aBuf.length !== bBuf.length) {
    return false;
  }
  return timingSafeEqual(aBuf, bBuf);
}
