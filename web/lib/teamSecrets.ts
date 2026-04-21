import fs from "fs";
import path from "path";
import { createCipheriv, createDecipheriv, createHash, randomBytes } from "crypto";

let cachedDevSecret: string | null = null;

function getDevSecretPath(): string {
  return path.resolve(process.cwd(), "..", ".local", "sshthing-team-secret.key");
}

function getOrCreateDevSecret(): string {
  if (cachedDevSecret) {
    return cachedDevSecret;
  }

  const secretPath = getDevSecretPath();
  const secretDir = path.dirname(secretPath);
  if (!fs.existsSync(secretDir)) {
    fs.mkdirSync(secretDir, { recursive: true });
  }

  if (!fs.existsSync(secretPath)) {
    const generated = randomBytes(32).toString("base64url");
    fs.writeFileSync(secretPath, `${generated}\n`, { encoding: "utf8", mode: 0o600 });
    cachedDevSecret = generated;
    return generated;
  }

  const existing = fs.readFileSync(secretPath, "utf8").trim();
  if (!existing) {
    const regenerated = randomBytes(32).toString("base64url");
    fs.writeFileSync(secretPath, `${regenerated}\n`, { encoding: "utf8", mode: 0o600 });
    cachedDevSecret = regenerated;
    return regenerated;
  }

  cachedDevSecret = existing;
  return existing;
}

function getKey(): Buffer {
  const raw = process.env.SSHTHING_TEAM_SECRET_KEY?.trim();
  const secret = raw || (process.env.NODE_ENV === "production" ? "" : getOrCreateDevSecret());
  if (!secret) {
    throw new Error("missing_team_secret_key");
  }
  return createHash("sha256").update(secret, "utf8").digest();
}

export function encryptTeamSecret(secret: string): string {
  const iv = randomBytes(12);
  const cipher = createCipheriv("aes-256-gcm", getKey(), iv);
  const encrypted = Buffer.concat([cipher.update(secret, "utf8"), cipher.final()]);
  const tag = cipher.getAuthTag();
  return `v1:${iv.toString("base64url")}:${tag.toString("base64url")}:${encrypted.toString("base64url")}`;
}

export function decryptTeamSecret(payload: string): string {
  const [version, ivRaw, tagRaw, cipherRaw] = payload.split(":");
  if (version !== "v1" || !ivRaw || !tagRaw || !cipherRaw) {
    throw new Error("invalid_secret_payload");
  }

  const decipher = createDecipheriv("aes-256-gcm", getKey(), Buffer.from(ivRaw, "base64url"));
  decipher.setAuthTag(Buffer.from(tagRaw, "base64url"));
  const decrypted = Buffer.concat([
    decipher.update(Buffer.from(cipherRaw, "base64url")),
    decipher.final(),
  ]);
  return decrypted.toString("utf8");
}
