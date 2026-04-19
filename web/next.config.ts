import fs from "fs";
import path from "path";
import type { NextConfig } from "next";

const projectDir = process.cwd();
const repoRoot = path.resolve(projectDir, "..");
const rootEnvPath = path.join(repoRoot, ".env.local");

function loadRootEnvFile() {
  if (!fs.existsSync(rootEnvPath)) {
    return;
  }

  const raw = fs.readFileSync(rootEnvPath, "utf8");
  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const separator = trimmed.indexOf("=");
    if (separator <= 0) {
      continue;
    }

    const key = trimmed.slice(0, separator).trim();
    const value = trimmed.slice(separator + 1).trim();
    if (!process.env[key]) {
      process.env[key] = value;
    }
  }
}

// Convex writes .env.local at the repo root. Load that here so the Next app
// can see the same deployment values without duplicating env files.
loadRootEnvFile();

if (!process.env.NEXT_PUBLIC_CONVEX_URL && process.env.CONVEX_URL) {
  process.env.NEXT_PUBLIC_CONVEX_URL = process.env.CONVEX_URL;
}

if (!process.env.CLERK_FRONTEND_API_URL && process.env.CLERK_JWT_ISSUER_DOMAIN) {
  process.env.CLERK_FRONTEND_API_URL = process.env.CLERK_JWT_ISSUER_DOMAIN;
}

const nextConfig: NextConfig = {
  reactStrictMode: true,
};

export default nextConfig;
