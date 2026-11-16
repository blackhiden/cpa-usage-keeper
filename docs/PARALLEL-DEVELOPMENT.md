# Parallel Feature Development with Git Worktrees

Reusable procedure for executing a roadmap phase with structured planning and,
where it is genuinely safe, parallel development through Git worktrees and
independent agent instances.

The driving instructions live in `docs/roadmap-worktree.md`. This document is
the reusable summary plus the operating script.

## 1. Phase planning procedure

1. Identify submodules (`git submodule status`). Submodules are part of the
   project and are always in scope.
2. Read `docs/ROADMAP.md` completely, then the target phase in full.
3. Read the documents the phase references, plus `AGENTS.md`.
4. Inspect the actual implementation: frontend modules, backend modules,
   schema, tests, Git state. Verify every claim in the roadmap against source.
5. Produce the plan: objective, current-state analysis, workstreams, tasks,
   dependencies, expected file impact, UI/UX work, backend/API work, database
   work, testing, risks.
6. Present the plan and **wait for explicit approval**. No application code
   before approval.

## 2. Parallel task selection rules

Split into parallel tasks only when all of these hold:

- No task consumes another task's public surface.
- No two tasks edit the same file.
- Each task is independently testable, reviewable, and mergeable.
- Each task can start from the same trunk commit.

If the phase is a dependency chain, run it sequentially. A strict chain in
worktrees produces N branches blocked on the same trunk plus guaranteed
conflicts on shared wiring files. **Fewer parallel tasks is correct when the
work is a chain.** Maximum 10; fewer is normal.

A single prerequisite (Wave 0 / Task 0) may be implemented before branching when
the approved plan says so.

## 3. Worktree creation

All feature branches must originate from the same trunk commit.

```bash
git fetch --all
BASE=$(git rev-parse <trunk>)
git worktree add -b feature/<phase>-<task> ../<repo>-wt-<task> "$BASE"
```

Layout:

```text
<repo>/                          # trunk checkout, never used for feature work
<repo>-wt-<task-a>/              # worktree for task A
<repo>-wt-<task-b>/              # worktree for task B
```

## 4. Branch naming convention

```text
feature/<phase-slug>-<task-slug>
```

Example: `feature/phase-2-22-live-flow-history`.

## 5. Agent assignment

One task per worktree, one agent per worktree. Give each agent: the task id, the
branch, the worktree path, the exact acceptance criteria, the test command, and
the rule that it may not touch any other worktree.

## 6. Worktree isolation

- Never work directly on the trunk branch.
- Never modify another feature worktree.
- Never reset, clean, or force-push.
- Never rewrite Git history.
- Never discard existing changes, including another session's staged files.

## 7. Shared-resource handling

Files that every task wants to touch (router registration, app wiring, i18n
bundles, shared component entry points) are the main reason to serialize. When
they cannot be avoided:

- Assign the shared file to exactly one task.
- Let other tasks expose an interface and have the owner wire it.
- Rebase onto the trunk before requesting review.

## 8. Testing

- Backend: `go test ./cmd/... ./internal/...`
- Frontend: `npm --prefix ./web run test`, `lint`, `typecheck`, `build`
- Equivalent aggregate: `make verify`
- Each task must be green on its own before integration analysis.

## 9. Review

Review per branch: diff against the base commit, acceptance criteria, tests
added, and confirmation that protected features in
`docs/MERGE-PRESERVATION.md` are untouched.

## 10. Integration analysis

Before merging, check for: overlapping edits, duplicated helpers, conflicting
i18n keys, duplicated route registration, and semantic conflicts that compile
but disagree.

## 11. Merge

Merging is a separate approval gate. Feature branches are not merged
automatically and are not deleted automatically.

## 12. Cleanup

```bash
git worktree remove ../<repo>-wt-<task>
git branch -d feature/<phase>-<task>
```

## 13. Common failure scenarios

| Symptom | Cause | Response |
|---|---|---|
| Every branch conflicts on one file | Shared wiring file | Serialize, or give the file a single owner |
| Task cannot compile alone | Dependency chain treated as parallel | Run sequentially |
| Worktree has uncommitted changes | Agent edited without committing | Stop; do not remove the worktree |
| Tests pass alone, fail combined | Semantic conflict | Integration analysis before merge |
| Branch diverges from trunk | Late rebase | Rebase before review, never force-push |

## 14. Script

`scripts/parallel-feature-worktrees.sh` implements creation and cleanup safely.
It never deletes existing work, never overwrites an existing worktree or branch,
and refuses to remove a worktree that still has uncommitted changes.

```bash
./scripts/parallel-feature-worktrees.sh create --base main \
  --repo . --branch feature/phase-2-22-a --path ../wt-a
./scripts/parallel-feature-worktrees.sh list --repo .
./scripts/parallel-feature-worktrees.sh remove --repo . --path ../wt-a
```
