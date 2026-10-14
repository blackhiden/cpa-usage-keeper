# Phase 2.19: Ready for Parallel Implementation

**Status:** ✅ Planning Complete | **Base Commit:** `bc28682e` | **Date:** 2026-10-14

---

## Summary

Phase 2.19 (Enhanced API Key Viewer Capabilities) has been successfully decomposed into **6 parallelizable tasks** with isolated Git worktrees and comprehensive agent instructions.

---

## Worktrees Created

All worktrees are in `/home/denny/Project/cpa/CLIProxyAPI/`:

| Task | Branch | Worktree | Time | Dependencies |
|------|--------|----------|------|--------------|
| Backend: Request Events | `feature/apikey-request-events-backend` | `cpauk-apikey-request-events-backend` | 2d | None |
| Backend: Live Flow | `feature/apikey-liveflow-backend` | `cpauk-apikey-liveflow-backend` | 0.5d | None |
| Backend: Ranking | `feature/apikey-ranking-default` | `cpauk-apikey-ranking-default` | 0.5d | None |
| Frontend: Events Page | `feature/apikey-request-events-ui` | `cpauk-apikey-request-events-ui` | 2d | Backend (can stub) |
| Frontend: Live Flow Page | `feature/apikey-liveflow-ui` | `cpauk-apikey-liveflow-ui` | 1d | Backend |
| Docs & i18n | `feature/apikey-docs-i18n` | `cpauk-apikey-docs-i18n` | 1d | ALL tasks |

**Total:** ~7 days sequential, ~3-4 days with parallelization

---

## Per-Worktree Files

Each worktree contains:
- **`AGENT_INSTRUCTIONS.md`** - Complete 30+ step implementation checklist
- **`INITIAL_PROMPT.md`** - Quick-start prompt for AI agents

---

## How to Start (Parallel Execution)

Launch independent Cline agents in separate VS Code windows:

```bash
# Agent 1: Backend Request Events
cd /home/denny/Project/cpa/CLIProxyAPI/cpauk-apikey-request-events-backend
code .
# Send INITIAL_PROMPT.md content to Cline

# Agent 2: Backend Live Flow
cd /home/denny/Project/cpa/CLIProxyAPI/cpauk-apikey-liveflow-backend
code .

# Agent 3: Backend Ranking
cd /home/denny/Project/cpa/CLIProxyAPI/cpauk-apikey-ranking-default
code .

# Agent 4: Frontend Events UI
cd /home/denny/Project/cpa/CLIProxyAPI/cpauk-apikey-request-events-ui
code .

# Agent 5: Frontend Live Flow UI
cd /home/denny/Project/cpa/CLIProxyAPI/cpauk-apikey-liveflow-ui
code .

# Agent 6: Documentation (START LAST - after all others complete)
cd /home/denny/Project/cpa/CLIProxyAPI/cpauk-apikey-docs-i18n
code .
```

---

## Design Decisions (Pre-Approved)

1. **Live Flow Scope:** Global view (all requests visible, metadata-only) with privacy notice
2. **Request Events:** Scoped by `api_key_id`, cross-key access returns 404
3. **Local Ranking:** Always-on (deprecate `APIKeyViewerLocalRankingEnabled` flag)

---

## Integration & Merge

**Order:**
1. Backend: Ranking → Live Flow → Request Events
2. Frontend: Events UI → Live Flow UI
3. Documentation (last)

**Test after each merge:** `make verify`

---

## Success Criteria

- [ ] All 6 features implemented and tested independently
- [ ] All features merged to `dev` branch
- [ ] `make verify` passes
- [ ] Manual testing confirms functionality
- [ ] Documentation complete (SPECIFICATION.md, ARCHITECTURE.md, README)
- [ ] i18n complete (en, zh, zh-TW)
- [ ] Worktrees cleaned up

---

## Key Documents

- `docs/PHASE_2_19_PLAN.md` - Full implementation plan
- `PHASE_2_19_COORDINATION.md` - Coordination guide
- `docs/ROADMAP.md` - Updated with Phase 2.19
- Each worktree: `AGENT_INSTRUCTIONS.md`, `INITIAL_PROMPT.md`

---

**Next Action:** Launch parallel agents or begin sequential implementation
