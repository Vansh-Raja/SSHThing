export function getConvexURL(): string {
  return process.env.NEXT_PUBLIC_CONVEX_URL || process.env.CONVEX_URL || "";
}

export function getClerkIssuerDomain(): string {
  return process.env.CLERK_FRONTEND_API_URL || process.env.CLERK_JWT_ISSUER_DOMAIN || "";
}

export function getRequiredEnv(name: "NEXT_PUBLIC_CONVEX_URL" | "CLERK_FRONTEND_API_URL"): string {
  const value = name === "NEXT_PUBLIC_CONVEX_URL" ? getConvexURL() : getClerkIssuerDomain();
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

export function getBrowserBaseURL(): string {
  return process.env.SSHTHING_BROWSER_BASE_URL || "http://localhost:3000";
}

export function hasBrowserTeamsEnv(): boolean {
  return Boolean(
    process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
      process.env.CLERK_SECRET_KEY &&
      getConvexURL() &&
      getClerkIssuerDomain(),
  );
}
