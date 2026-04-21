#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: deploy_remote.sh [options]

Deploy SSHThing from a Git branch on the server using a dedicated git worktree.

Options:
  --branch <name>     Git branch to deploy. Default: main
  --env <name>        Stack environment to deploy. Default: test
  --repo-dir <path>   Bare repo checkout path. Default: /home/ubuntu/Code/SSHThing
  --state-dir <path>  Persistent deploy state directory. Default: <repo-dir>/.deploy-state/<env>
  --skip-web          Deploy backend services only
  --help              Show this help
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

wait_for_url() {
  local url="$1"
  local attempts="${2:-45}"
  local delay="${3:-2}"
  local i

  for ((i = 1; i <= attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done

  echo "Timed out waiting for $url" >&2
  return 1
}

branch="main"
environment="test"
repo_dir="/home/ubuntu/Code/SSHThing"
state_dir=""
skip_web="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --branch)
      branch="${2:?missing value for --branch}"
      shift 2
      ;;
    --env)
      environment="${2:?missing value for --env}"
      shift 2
      ;;
    --repo-dir)
      repo_dir="${2:?missing value for --repo-dir}"
      shift 2
      ;;
    --state-dir)
      state_dir="${2:?missing value for --state-dir}"
      shift 2
      ;;
    --skip-web)
      skip_web="true"
      shift
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

require_cmd git
require_cmd docker
require_cmd python3
require_cmd curl

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose is required" >&2
  exit 1
fi

if [[ ! -d "$repo_dir/.git" ]]; then
  echo "Repo checkout not found at $repo_dir" >&2
  exit 1
fi

if ! git -C "$repo_dir" ls-remote --exit-code --heads origin "$branch" >/dev/null 2>&1; then
  echo "Remote branch origin/$branch does not exist" >&2
  exit 1
fi

branch_slug="$(printf '%s' "$branch" | sed 's#[^A-Za-z0-9._-]#-#g')"
worktrees_root="$repo_dir/.deploy-worktrees"
worktree_dir="$worktrees_root/${environment}-${branch_slug}"
deploy_branch="deploy-${environment}-${branch_slug}"

if [[ -z "$state_dir" ]]; then
  state_dir="$repo_dir/.deploy-state/$environment"
fi

mkdir -p "$worktrees_root" "$state_dir"

git -C "$repo_dir" fetch origin "$branch"

if [[ ! -d "$worktree_dir/.git" ]]; then
  git -C "$repo_dir" worktree add -B "$deploy_branch" "$worktree_dir" "origin/$branch"
else
  git -C "$worktree_dir" fetch origin "$branch"
  git -C "$worktree_dir" checkout "$deploy_branch"
  git -C "$worktree_dir" merge --ff-only "origin/$branch"
fi

compose_file="$worktree_dir/deploy/sshthing/compose.yml"
example_env="$worktree_dir/deploy/sshthing/env/$environment/stack.env.example"
legacy_env="$repo_dir/deploy/sshthing/env/$environment/stack.env"
env_file="$state_dir/stack.env"
admin_key_file="$state_dir/convex_admin_key.txt"

if [[ ! -f "$compose_file" ]]; then
  echo "Compose file not found at $compose_file" >&2
  exit 1
fi

if [[ ! -f "$env_file" ]]; then
  if [[ -f "$legacy_env" ]]; then
    cp "$legacy_env" "$env_file"
  elif [[ -f "$example_env" ]]; then
    cp "$example_env" "$env_file"
    echo "Created $env_file from example. Fill in real values before deploying." >&2
    exit 1
  else
    echo "No env file found and no example available for $environment" >&2
    exit 1
  fi
fi

chmod 600 "$env_file"

set -a
# shellcheck disable=SC1090
source "$env_file"
set +a

services=(postgres convex convex-dashboard)
if [[ "$skip_web" != "true" && -f "$worktree_dir/web/Dockerfile" && -f "$worktree_dir/web/package.json" ]]; then
  services+=(web)
else
  echo "Skipping web deploy. Missing web app sources or --skip-web was set."
fi

docker compose --env-file "$env_file" -f "$compose_file" up -d --build "${services[@]}"

wait_for_url "http://127.0.0.1:${CONVEX_API_PORT}/version"
wait_for_url "http://127.0.0.1:${CONVEX_DASHBOARD_PORT}"

if [[ " ${services[*]} " == *" web "* ]]; then
  wait_for_url "http://127.0.0.1:${WEB_PORT}"
fi

if [[ ! -s "$admin_key_file" ]]; then
  docker compose --env-file "$env_file" -f "$compose_file" exec -T convex \
    sh -lc './generate_admin_key.sh | awk "NF { line = \$0 } END { print line }"' \
    >"$admin_key_file"
  chmod 600 "$admin_key_file"
fi

cat <<EOF
Deploy complete.
branch=$branch
environment=$environment
worktree_dir=$worktree_dir
env_file=$env_file
admin_key_file=$admin_key_file
services=${services[*]}
EOF

docker compose --env-file "$env_file" -f "$compose_file" ps
