import type { AuthConfig } from "convex/server";

declare const process: {
  env: Record<string, string | undefined>;
};

export default {
  providers: [
    {
      domain: (process.env.CLERK_JWT_ISSUER_DOMAIN ?? process.env.CLERK_FRONTEND_API_URL) as string,
      applicationID: "convex",
    },
  ],
} satisfies AuthConfig;
