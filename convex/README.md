# SSHThing Convex Backend

This package hosts the SSHThing Teams backend model:

- Clerk-backed auth integration
- team, member, host, and invite state
- CLI device-flow session issuance
- TUI session storage

Run `pnpm convex:dev` from the repo root after configuring your Convex environment.

Required environment for Clerk auth:

- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
- `CLERK_SECRET_KEY`
- `CLERK_FRONTEND_API_URL` or `CLERK_JWT_ISSUER_DOMAIN`

Local Convex development writes `CONVEX_URL` and `CONVEX_DEPLOYMENT` to the
repo-root `.env.local`. The Next app in `web/` loads those values automatically.
