# SSHThing Teams Setup

## Required services

- Clerk project for authentication and organizations
- Convex deployment for application data and HTTP actions
- Web app deployment for browser auth handoff

## Current local state

The Teams worktree is already attached to a Convex dev deployment:

- `CONVEX_DEPLOYMENT=dev:frugal-leopard-172`
- `CONVEX_URL=https://frugal-leopard-172.convex.cloud`
- `CONVEX_SITE_URL=https://frugal-leopard-172.convex.site`

The local browser app env file at `web/.env.local` has already been prefilled
with the Convex values above. Clerk keys still need to be added there.

## Local AI tooling

The local Codex MCP config is expected to include:

- Clerk MCP as a remote URL server
- Convex MCP as a stdio server scoped to `D:\Code\SSHThing-teams`

Example `C:\Users\UrAvgProGamer\.codex\config.toml` entries:

```toml
[beta]
rmcp = true

[mcp_servers.clerk]
type = "url"
url = "https://mcp.clerk.com/mcp"

[mcp_servers.convex]
type = "stdio"
command = "npx"
args = ["-y", "convex@latest", "mcp", "start", "--project-dir", "D:\\Code\\SSHThing-teams"]
```

Notes:

- Clerk MCP requires remote MCP support (`rmcp = true`).
- Convex MCP does not need a global install when started through `npx`.
- Convex MCP should stay pointed at the Teams worktree, not the main repo root.
- If Codex does not pick up MCP config changes immediately, restart the Codex session.

## Expected environment variables

### Go TUI

- `SSHTHING_TEAMS_ENABLED`
- `SSHTHING_TEAMS_API_BASE_URL`
- `SSHTHING_TEAMS_BROWSER_BASE_URL`

### Web app

- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
- `CLERK_SECRET_KEY`
- `NEXT_PUBLIC_CONVEX_URL`
- `CONVEX_DEPLOYMENT`

### Convex

- Clerk auth integration secrets
- any future encryption key material for shared team credentials

## First implementation notes

- Billing is intentionally deferred.
- Shared team credential caching is disabled by default.
- Teams remains cloud-only in v1.

## Useful commands

From `D:\Code\SSHThing-teams`:

```powershell
npm install
npm run convex:dev
npm run web:dev
```

If Convex on Windows complains about temp directories being on different
filesystems, set:

```powershell
$env:CONVEX_TMPDIR='D:\Code\SSHThing-teams\.convex-tmp'
```
