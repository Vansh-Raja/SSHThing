#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: deploy_via_sshthing.sh [options]

Push a branch if requested, upload the remote deploy helper, and run the deploy
on a server through SSHThing.

Options:
  --target <label>     SSHThing target host label. Default: Main Deployment Server
  --auth-file <path>   SSHThing token file. Default: ~/.sshthing/tokens/personal-projects.token
  --branch <name>      Git branch to deploy. Default: main
  --env <name>         Stack environment to deploy. Default: test
  --repo-dir <path>    Repo path on the server. Default: /home/ubuntu/Code/SSHThing
  --push               Push the branch to origin before deploying
  --skip-web           Deploy backend services only
  --help               Show this help
EOF
}

target="Main Deployment Server"
auth_file="$HOME/.sshthing/tokens/personal-projects.token"
branch="main"
environment="test"
repo_dir="/home/ubuntu/Code/SSHThing"
push_branch="false"
skip_web="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)
      target="${2:?missing value for --target}"
      shift 2
      ;;
    --auth-file)
      auth_file="${2:?missing value for --auth-file}"
      shift 2
      ;;
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
    --push)
      push_branch="true"
      shift
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

if ! command -v sshthing >/dev/null 2>&1; then
  echo "sshthing must be installed and available in PATH" >&2
  exit 1
fi

if [[ "$push_branch" == "true" ]]; then
  current_branch="$(git rev-parse --abbrev-ref HEAD)"
  if [[ "$current_branch" == "$branch" ]] && [[ -n "$(git status --porcelain)" ]]; then
    echo "Refusing to push dirty branch $branch. Commit or stash your changes first." >&2
    exit 1
  fi
  git push -u origin "$branch"
fi

remote_script="$(cat "$(dirname "$0")/deploy_remote.sh")"
remote_command="$(cat <<EOF
set -euo pipefail
mkdir -p "$repo_dir/.deploy-tmp"
cat > "$repo_dir/.deploy-tmp/deploy_remote.sh" <<'SCRIPT'
$remote_script
SCRIPT
chmod +x "$repo_dir/.deploy-tmp/deploy_remote.sh"
"$repo_dir/.deploy-tmp/deploy_remote.sh" --branch "$branch" --env "$environment" --repo-dir "$repo_dir" $( [[ "$skip_web" == "true" ]] && printf '%s' '--skip-web' )
EOF
)"

sshthing exec -t "$target" --auth-file "$auth_file" "$remote_command"
