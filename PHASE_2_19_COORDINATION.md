# Phase 2.19 Parallel Development Coordination

**Phase:** Enhanced API Key Viewer Capabilities  
**Status:** Ready for parallel implementation  
**Base Commit:** `bc28682e6948802b86bead0a6aa3bf939c6e7695`  
**Created:** 2026-10-14  

---

## Overview

Phase 2.19 has been decomposed into 6 parallelizable tasks, each with its own Git worktree and feature branch.

---

## Worktrees & Branches

All worktrees are created one level above the main repository:

| Task | Branch | Worktree Path | Status |
|------|--------|---------------|--------|
| Backend: Request Events | `feature/apikey-request-events-backend` | `../cpauk-apikey-request-events-backend` | ⬜ |
| Backend: Live Flow Access | `feature/apikey-liveflow-backend` | `../cpauk-apikey-liveflow-backend` | ⬜ |
| Backend: Ranking Always-On | `feature/apikey-ranking-default` | `../cpauk-apikey-ranking-default` | ⬜ |
| Frontend: Request Events | `feature/apikey-request-events-ui` | `../cpauk-apikey-request-events-ui` | ⬜ |
| Frontend: Live Flow | `feature/apikey-liveflow-ui` | `../cpauk-apikey-liveflow-ui` | ⬜ |
| Documentation & i18n | `feature/apikey-docs-i18n` | `../cpauk-apikey-docs-i18n` | ⬜ |

---

## Task Dependencies

Sequence 1 (Parallel):
- All backend tasks (no dependencies)
- All frontend tasks (can stub API initially)

Sequence 2 (Sequential):
- Documentation & i18n (after all features complete)

---

## Starting Work

Each worktree has:
- `AGENT_INSTRUCTIONS.md` - Full implementation checklist
- `INITIAL_PROMPT.md` - Quick-start for AI agents

For AI agents: Send `INITIAL_PROMPT.md` content as first message.

---

## Verification

Backend: `go build ./... && go test ./internal/...`
Frontend: `npm --prefix ./web run test && npm run typecheck && npm run build`
Docs: `make verify`

---

## Integration

1. Complete all tasks independently
2. Verify no conflicts
3. Merge in order: backend → frontend → docs
4. Test after each merge
5. Run `make verify` on final merge

---

See [PHASE_2_19_PLAN.md](docs/PHASE_2_19_PLAN.md) for details.
