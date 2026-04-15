import type { AuthConfig } from "convex/server";

export default {
  providers: [
    {
      // Placeholder until the real Clerk project is created and its issuer domain is known.
      domain: "https://placeholder.clerk.accounts.dev",
      applicationID: "convex"
    }
  ]
} satisfies AuthConfig;
