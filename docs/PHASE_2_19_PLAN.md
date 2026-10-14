# Phase 2.19: Enhanced API Key Viewer Capabilities — Implementation Plan

**Status:** ⬜ Not started  
**Estimated scope:** Large (5-7 days for a single developer)  
**Isolation:** Suitable for worktree execution  
**Created:** 2026-10-14  

## 1. Overview

Expand API key viewer capabilities to include:
- **Local Ranking** (visible by default)
- **Overview** (own stats) — already available
- **Live Flow** (global view of all flows)
- **Analysis** (own stats) — already available
- **Request Events** (own events only)

### Current State
API key viewers have: /key-overview, /key-realtime, /key-activity, /key-analysis, /key-ranking (config-gated)
They do NOT have: Request Events (admin-only), Live Flow (explicitly denied)

### Target State
1. Local Ranking — enabled by default
2. Overview — own stats (already available)
3. Live Flow — global view (metadata-only)
4. Analysis — own stats (already available)
5. Request Events — scoped to their API key only

## 2. Design Decisions (REQUIRED BEFORE IMPLEMENTATION)

### 2.1 Live Flow Scope
**Option A (Recommended):** Global view (all flows, metadata-only)
- Pros: Simple, reuses existing endpoint
- Cons: Privacy concern (viewers see ALL request metadata)

**Option B:** Key-scoped view (only their flows)
- Pros: Better privacy
- Cons: Requires server-side filtering

**Recommendation:** Option A for MVP, document privacy implications

### 2.2 Request Events Page
**Recommendation:** Dedicated /key-events route

### 2.3 Local Ranking
**Recommendation:** Always-on (no config flag)

### 2.4 Menu Ordering
Overview → Realtime → Analysis → Local Ranking → Request Events → Live Flow

## 3. Key Files to Modify

### Backend
- internal/api/router.go — register new routes in keyViewerProtected
- internal/api/usage_events.go — add key-viewer handlers
- internal/api/live_flow.go — accept API key viewer sessions
- internal/api/test/key_viewer_*_test.go — new test files

### Frontend
- web/src/features/key-viewer/navigation.ts — extend KeyViewerPath type
- web/src/pages/KeyEventsPage.tsx — new page
- web/src/pages/KeyLiveFlowPage.tsx — new page (optional)
- web/src/App.tsx — add routing
- web/src/features/key-viewer/KeyViewerShell.tsx — add menu items
- web/src/i18n/locales/*.json — add translations

### Documentation
- docs/SPECIFICATION.md — update §2 Users & Roles table
- docs/ARCHITECTURE.md — document new routes
- README.md / README.zh.md — update feature lists

## 4. Implementation Checklist

### Phase 1: Research & Design
- [ ] Read current implementation files (router.go, usage_events.go, live_flow.go, key-viewer/)
- [ ] Make Live Flow access decision (global vs. scoped)
- [ ] Document decision in session.md or ADR

### Phase 2: Backend (Request Events)
- [ ] Add /usage/events endpoint for key viewers (scoped by api_key_id)
- [ ] Add request log endpoints (scoped access control)
- [ ] Add backend tests for authorization and scoping
- [ ] Run: go test ./internal/api/test/...

### Phase 3: Backend (Live Flow & Ranking)
- [ ] Update Live Flow WebSocket to accept key viewer sessions
- [ ] Remove config flag for local ranking (always-on)
- [ ] Add Live Flow authorization tests
- [ ] Run: go test ./internal/api/test/...

### Phase 4: Frontend
- [ ] Extend KeyViewerPath type (add /key-events, /key-live-flow)
- [ ] Create KeyEventsPage component
- [ ] Create KeyLiveFlowPage component (if enabled)
- [ ] Update KeyViewerShell menu
- [ ] Update App.tsx routing
- [ ] Add API functions for key-viewer endpoints
- [ ] Add frontend tests
- [ ] Run: npm --prefix ./web run test

### Phase 5: i18n
- [ ] Add keys to en.json, zh.json, zh-TW.json

### Phase 6: Documentation
- [ ] Update SPECIFICATION.md (Users & Roles table, add UC-2b)
- [ ] Update ARCHITECTURE.md (routes, module layout)
- [ ] Update README.md and README.zh.md
- [ ] Document Live Flow decision
- [ ] Mark APIKeyViewerLocalRankingEnabled as deprecated

### Phase 7: Verification
- [ ] Run: make verify
- [ ] Fix any failures
- [ ] Manual testing: login → all pages → verify scoping
- [ ] Test export (CSV/JSON)
- [ ] Test Live Flow connection
- [ ] Update session.md

## 5. Backend Implementation Details

### Request Events Endpoint
Register in keyViewerProtected group:
  registerKeyRequestEventsRoute(keyViewerProtected, usageProvider, ...)

Handler scopes by session.api_key_id:
  query.APIKeyID = &apiKeyID
  events, err := usageProvider.GetUsageEvents(ctx, query)

### Request Log Authorization
Verify event belongs to viewer:
  event, err := usageProvider.GetUsageEventByID(ctx, eventID)
  if err || event.APIKeyID != apiKeyID { return 404 }

### Live Flow Authorization
Accept both admin and API key viewer sessions:
  adminSession, hasAdmin := auth.GetAdminSession(c, sessions)
  viewerSession, hasViewer := auth.GetAPIKeyViewerSession(c, sessions)
  if !hasAdmin && !hasViewer { return 401 }

## 6. Security Checklist
- [ ] Request events MUST be filtered by session.api_key_id
- [ ] Request logs MUST be redacted (prompts, bodies, headers)
- [ ] Cross-key access MUST return 404 or empty
- [ ] Rate limiting MUST apply
- [ ] Live Flow privacy MUST be documented in UI

## 7. Testing Requirements

### Backend Tests (new files)
- internal/api/test/key_viewer_request_events_test.go
- internal/api/test/key_viewer_live_flow_test.go

Test cases:
1. Key viewer can fetch their own request events
2. Key viewer CANNOT fetch other keys' events
3. Key viewer can fetch request logs for their events
4. Key viewer CANNOT fetch logs for other keys' events
5. Key viewer can connect to Live Flow WebSocket
6. Export is scoped correctly
7. Rate limiting applies

### Frontend Tests
- web/src/pages/test/KeyEventsPage.test.tsx

## 8. Timeline: 5-7 days

Day 1: Research, design decisions, backend scaffolding
Day 2: Backend (request events, request logs)
Day 3: Backend (Live Flow auth, tests)
Day 4: Frontend (pages, routing, API)
Day 5: i18n, documentation
Day 6: Testing, verification, fixes
Day 7: Buffer

## 9. Success Criteria

✅ API key viewers can:
1. Access Local Ranking (no config)
2. View their own request events with filters
3. Export their events (CSV/JSON)
4. View request logs for their events
5. Connect to Live Flow WebSocket

✅ Security:
1. Events strictly scoped
2. Logs redacted
3. Cross-key access blocked
4. Rate limiting enforced

✅ Tests: make verify passes

✅ Documentation: SPECIFICATION.md, ARCHITECTURE.md, README updated

## 10. Related Documents
- ROADMAP.md — Phase 2.19 entry
- SPECIFICATION.md — §2 Users & Roles
- ARCHITECTURE.md — module layout
- session.md — decision documentation

