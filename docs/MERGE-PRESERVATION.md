# Merge Preservation

## API-key viewer full request logs

This fork requires `GET /api/v1/key-events/:id/request-log` to return every
CPA request-log section for the event owner when
`CPA_REQUEST_LOG_ACCESS_ENABLED=true`.

Do not add request-log redaction, filtering, or an allowlist after the route's
ownership verification. Full log access is intentionally limited to the event
owner: cross-key access returns `404`, and disabled request-log access returns
`403`.

The route carries a `FORK-ONLY` comment and
`TestAPIKeyViewerRequestLogFullContentPersistence` compares every returned
section with its original title and content. Preserve both during upstream
merges.

After a merge, run:

```bash
grep -n 'FORK-ONLY: The owning API-key viewer receives every request-log section' internal/api/key_viewer_request_events.go
go test ./internal/api/test/ -run 'TestAPIKeyViewerRequestLogFullContentPersistence|TestAPIKeyViewerRequestLogRejectsCrossKeyAccess|TestAPIKeyViewerRequestLogRejectsDisabledAccess' -count=1
```
