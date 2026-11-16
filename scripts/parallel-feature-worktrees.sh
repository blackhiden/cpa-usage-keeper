#!/usr/bin/env bash
# Safe Git worktree helper for parallel feature development.
#
# See docs/PARALLEL-DEVELOPMENT.md. This script never deletes existing work,
# never overwrites an existing worktree or branch, and refuses to remove a
# worktree that still has uncommitted changes.

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  parallel-feature-worktrees.sh create --base <ref> --branch <name> --path <dir> [--repo <dir>]
  parallel-feature-worktrees.sh list [--repo <dir>]
  parallel-feature-worktrees.sh remove --path <dir> [--repo <dir>]

Commands:
  create   Create a new worktree and branch from --base. Fails if the branch
           or path already exists.
  list     List existing worktrees for the repository.
  remove   Remove a worktree. Fails if it has uncommitted changes. The branch
           is never deleted automatically.
EOF
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

repo="."

command="${1:-}"
if [[ -z "$command" ]]; then
  usage
  exit 1
fi
shift

base=""
branch=""
path=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base) base="${2:-}"; shift 2 ;;
    --branch) branch="${2:-}"; shift 2 ;;
    --path) path="${2:-}"; shift 2 ;;
    --repo) repo="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
done

[[ -d "$repo/.git" || -f "$repo/.git" ]] || die "not a git repository: $repo"

case "$command" in
  create)
    [[ -n "$base" ]] || die "create requires --base"
    [[ -n "$branch" ]] || die "create requires --branch"
    [[ -n "$path" ]] || die "create requires --path"

    git -C "$repo" rev-parse --verify --quiet "$base^{commit}" >/dev/null \
      || die "base ref does not exist: $base"

    # Never overwrite an existing branch.
    if git -C "$repo" show-ref --verify --quiet "refs/heads/$branch"; then
      die "branch already exists: $branch"
    fi

    # Never overwrite an existing worktree directory.
    if [[ -e "$path" ]]; then
      die "path already exists: $path"
    fi

    # Never create a duplicate worktree for the same branch.
    if git -C "$repo" worktree list --porcelain | grep -qx "branch refs/heads/$branch"; then
      die "worktree already exists for branch: $branch"
    fi

    git -C "$repo" worktree add -b "$branch" "$path" "$base"
    printf 'created worktree %s on branch %s from %s\n' "$path" "$branch" "$base"
    ;;

  list)
    git -C "$repo" worktree list
    ;;

  remove)
    [[ -n "$path" ]] || die "remove requires --path"
    [[ -e "$path" ]] || die "worktree path does not exist: $path"

    # Refuse to discard uncommitted changes.
    if [[ -n "$(git -C "$path" status --porcelain 2>/dev/null)" ]]; then
      die "worktree has uncommitted changes, refusing to remove: $path"
    fi

    git -C "$repo" worktree remove "$path"
    printf 'removed worktree %s (branch was not deleted)\n' "$path"
    ;;

  *)
    usage
    die "unknown command: $command"
    ;;
esac
