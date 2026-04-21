# SSHThing Self-Hosted Deployment

This directory packages the SSHThing web app together with a self-hosted Convex backend, Convex dashboard, and Postgres for persistence.

The first rollout target is the testing stack only:

- Compose project: `sshthing-test`
- Web UI: `http://127.0.0.1:3001`
- Convex API: `http://127.0.0.1:3220`
- Convex site / HTTP actions: `http://127.0.0.1:3221`
- Convex dashboard: `http://127.0.0.1:6792`
- Postgres: `127.0.0.1:5433`

Production is templated but should not be started during the first rollout.

## Files

- `compose.yml`: parameterized multi-service stack
- `deploy_remote.sh`: server-side branch deploy helper using a dedicated git worktree
- `deploy_via_sshthing.sh`: local wrapper that uploads the helper and deploys over SSHThing
- `env/test/stack.env.example`: test stack template
- `env/prod/stack.env.example`: prod stack template
- `../../.env.selfhosted.test.example`: local CLI template for targeting the remote test Convex deployment
- `../../.env.selfhosted.prod.example`: local CLI template for the future prod deployment

## Branch-based deploy flow

Use the local wrapper to deploy a specific branch to a server:

```bash
./deploy/sshthing/deploy_via_sshthing.sh \
  --target "Main Deployment Server" \
  --branch feat/teams-live-foundation \
  --env test
```

If the branch already exists on GitHub and the server only needs to update, the
command above is enough. If you want the wrapper to push the branch first, use:

```bash
./deploy/sshthing/deploy_via_sshthing.sh \
  --target "Main Deployment Server" \
  --branch feat/teams-live-foundation \
  --env test \
  --push
```

`--push` intentionally refuses to run when the target branch is dirty on the
local machine because Git cannot deploy uncommitted work.

The branch you deploy must already contain the deployment assets, including:

- `deploy/sshthing/compose.yml`
- `deploy/sshthing/env/<env>/stack.env.example`
- the `web/` app sources if you want the `web` service built on the server

The remote helper defaults to `main`, so production can use the same script
without changing the branch flag:

```bash
./deploy/sshthing/deploy_via_sshthing.sh --env prod --branch main
```

The remote deploy helper keeps persistent state outside the branch worktree:

- checkout state lives in `/home/ubuntu/Code/SSHThing/.deploy-worktrees`
- stack env files live in `/home/ubuntu/Code/SSHThing/.deploy-state/<env>`
- admin keys live in `/home/ubuntu/Code/SSHThing/.deploy-state/<env>/convex_admin_key.txt`

## Server rollout

Target host layout:

- repo checkout: `/home/ubuntu/Code/SSHThing`
- deployed stack env file: `/home/ubuntu/Code/SSHThing/deploy/sshthing/env/test/stack.env`

Suggested first-time setup on the server:

```bash
cd /home/ubuntu/Code/SSHThing
cp deploy/sshthing/env/test/stack.env.example deploy/sshthing/env/test/stack.env
chmod 600 deploy/sshthing/env/test/stack.env
```

Fill in real values for:

- `POSTGRES_PASSWORD`
- `INSTANCE_SECRET`
- `SSHTHING_TEAM_SECRET_KEY`
- Clerk test keys
- final public test URLs that Cloudflare will point at later

The Convex images should use a multi-arch tag or manifest-list digest.
Do not pin an amd64-only image digest if the target host is arm64.

Start the test stack:

```bash
cd /home/ubuntu/Code/SSHThing
docker compose \
  --env-file deploy/sshthing/env/test/stack.env \
  -f deploy/sshthing/compose.yml \
  up -d --build
```

Stop the test stack:

```bash
cd /home/ubuntu/Code/SSHThing
docker compose \
  --env-file deploy/sshthing/env/test/stack.env \
  -f deploy/sshthing/compose.yml \
  down
```

Check service state:

```bash
cd /home/ubuntu/Code/SSHThing
docker compose \
  --env-file deploy/sshthing/env/test/stack.env \
  -f deploy/sshthing/compose.yml \
  ps
```

Read logs:

```bash
cd /home/ubuntu/Code/SSHThing
docker compose \
  --env-file deploy/sshthing/env/test/stack.env \
  -f deploy/sshthing/compose.yml \
  logs -f convex convex-dashboard web postgres
```

Generate the Convex admin key after the backend is healthy:

```bash
cd /home/ubuntu/Code/SSHThing
docker compose \
  --env-file deploy/sshthing/env/test/stack.env \
  -f deploy/sshthing/compose.yml \
  exec convex ./generate_admin_key.sh
```

## Smoke checks on the server

```bash
curl -fsS http://127.0.0.1:3001
curl -fsS http://127.0.0.1:3220/version
curl -fsS http://127.0.0.1:3221
curl -fsS http://127.0.0.1:6792
psql "postgresql://sshthing:<password>@127.0.0.1:5433/sshthing_test" -c 'select 1'
```

## Local CLI targeting the remote self-hosted test backend

Create a non-committed local file from the template:

```bash
cp .env.selfhosted.test.example .env.selfhosted.test
```

Then populate:

- `CONVEX_SELF_HOSTED_URL`
- `CONVEX_SELF_HOSTED_ADMIN_KEY`

Example usage:

```bash
set -a
source .env.selfhosted.test
set +a
npx convex dev
```

## Cloudflare and Clerk follow-up

This stack binds only to `127.0.0.1`, so it is not public by itself. After the test stack is healthy, point your Cloudflare tunnel/public domains at:

- `127.0.0.1:3001` for the test app
- `127.0.0.1:3220` for the Convex API
- `127.0.0.1:3221` for Convex site / HTTP actions
- `127.0.0.1:6792` for the Convex dashboard

Once the public test domains are live, make sure the matching Clerk test application allows those origins and redirect URLs.
