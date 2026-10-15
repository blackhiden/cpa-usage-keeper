package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cpa-usage-keeper/internal/service"
	servicedto "cpa-usage-keeper/internal/service/dto"
	"cpa-usage-keeper/internal/timeutil"

	"github.com/gin-gonic/gin"
)

// registerKeyRequestEventsRoute registers scoped request events endpoints for API key viewers.
// All endpoints filter by session.CPAAPIKeyID and return 404 for cross-key access.
func registerKeyRequestEventsRoute(
	router gin.IRoutes,
	usageProvider service.UsageProvider,
	usageIdentityProvider service.UsageIdentityProvider,
	requestLogProvider service.RequestLogProvider,
	requestLogAccessEnabled bool,
) {
	exportSlots := make(chan struct{}, usageEventsExportMaxConcurrency)

	// GET /usage/events - List events (scoped by viewer's API key)
	router.GET("/key-events", func(c *gin.Context) {
		session, _, ok := activeAPIKeyViewerContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		filter, err := parseUsageFilterQuery(c.Request, timeutil.NormalizeStorageTime(time.Now()))
		if err != nil {
			writeUsageFilterParseError(c, err)
			return
		}
		// Scope by viewer's API key - overwrite any client-provided api_key_id
		filter.APIKeyID = fmt.Sprintf("%d", session.CPAAPIKeyID)

		// Apply source filter if present
		if err := applyUsageEventsSourceFilter(&filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if usageProvider == nil {
			writeInternalError(c, "usage provider is unavailable", nil)
			return
		}
		rows, err := usageProvider.ListUsageEvents(c.Request.Context(), filter)
		if err != nil {
			writeInternalError(c, "list usage events failed", err)
			return
		}
		identities, err := loadUsageResolutionData(c, usageIdentityProvider)
		if err != nil {
			return
		}
		resolver := newUsageIdentityResolver(identities)
		// For key viewer, we don't expose CPA API Key mappings - use empty map
		apiKeyInfos := map[string]analysisAPIKeyInfo{}
		nextCursor := ""
		if filter.CursorMode && rows.HasMore && len(rows.Events) > 0 {
			lastEvent := rows.Events[len(rows.Events)-1]
			nextCursor = encodeUsageEventsCursor(lastEvent.Timestamp, lastEvent.ID)
		}
		c.JSON(http.StatusOK, usageEventsResponse{
			Events:     buildUsageEventsPayload(rows.Events, resolver, apiKeyInfos),
			TotalCount: rows.TotalCount,
			Page:       rows.Page,
			PageSize:   rows.PageSize,
			TotalPages: rows.TotalPages,
			NextCursor: nextCursor,
			HasMore:    rows.HasMore,
		})
	})

	// GET /usage/events/filter-options - Filter options (scoped)
	router.GET("/key-events/filter-options", func(c *gin.Context) {
		_, _, ok := activeAPIKeyViewerContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		models, err := loadUsageEventModelFilterOptions(c, usageProvider)
		if err != nil {
			writeInternalError(c, "load model filter options failed", err)
			return
		}
		sources, err := loadUsageEventSourceFilterOptions(c, usageIdentityProvider)
		if err != nil {
			writeInternalError(c, "load source filter options failed", err)
			return
		}
		setNoStoreHeaders(c)
		c.JSON(http.StatusOK, usageEventFilterOptionsResponse{
			Models:  models,
			Sources: sources,
		})
	})

	registerKeyEventDetailRoute(router, usageProvider, usageIdentityProvider)
	registerKeyEventRequestLogRoute(router, usageProvider, requestLogProvider, requestLogAccessEnabled)
	registerKeyEventExportRoute(router, usageProvider, usageIdentityProvider, exportSlots)
}

func registerKeyEventDetailRoute(
	router gin.IRoutes,
	usageProvider service.UsageProvider,
	usageIdentityProvider service.UsageIdentityProvider,
) {
	// GET /usage/events/:id - Get event detail (scoped, verify ownership)
	router.GET("/key-events/:id", func(c *gin.Context) {
		session, _, ok := activeAPIKeyViewerContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		eventID, ok := parseUsageEventRequestLogEventID(c)
		if !ok {
			return
		}

		// Verify ownership
		if usageProvider == nil {
			writeInternalError(c, "usage provider is unavailable", nil)
			return
		}

		verifier, ok := usageProvider.(interface {
			VerifyUsageEventOwnership(context.Context, int64, string) (bool, error)
		})
		if !ok {
			writeInternalError(c, "ownership verification not supported", nil)
			return
		}

		owned, err := verifier.VerifyUsageEventOwnership(c.Request.Context(), eventID, fmt.Sprintf("%d", session.CPAAPIKeyID))
		if err != nil {
			writeInternalError(c, "verify event ownership failed", err)
			return
		}
		if !owned {
			c.JSON(http.StatusNotFound, gin.H{"error": "usage event not found"})
			return
		}

		// Fetch event using streaming to find the specific one
		filter := servicedto.UsageFilter{
			APIKeyID: fmt.Sprintf("%d", session.CPAAPIKeyID),
		}
		var matchedEvent *servicedto.UsageEventRecord
		streamErr := usageProvider.StreamUsageEvents(c.Request.Context(), filter, func(record servicedto.UsageEventRecord) error {
			if record.ID == eventID {
				matchedEvent = &record
				return fmt.Errorf("found")
			}
			return nil
		})
		if streamErr != nil && streamErr.Error() != "found" {
			writeInternalError(c, "get usage event failed", streamErr)
			return
		}

		if matchedEvent == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "usage event not found"})
			return
		}

		identities, err := loadUsageResolutionData(c, usageIdentityProvider)
		if err != nil {
			return
		}
		resolver := newUsageIdentityResolver(identities)
		apiKeyInfos := map[string]analysisAPIKeyInfo{}

		// Build single event payload from the record
		payloads := buildUsageEventsPayload([]servicedto.UsageEventRecord{*matchedEvent}, resolver, apiKeyInfos)
		if len(payloads) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "usage event not found"})
			return
		}
		c.JSON(http.StatusOK, payloads[0])
	})
}

func registerKeyEventRequestLogRoute(
	router gin.IRoutes,
	usageProvider service.UsageProvider,
	requestLogProvider service.RequestLogProvider,
	requestLogAccessEnabled bool,
) {
	// FORK-ONLY: The owning API-key viewer receives every request-log section.
	router.GET("/key-events/:id/request-log", func(c *gin.Context) {
		if !requestLogAccessEnabled {
			writeUsageEventRequestLogAccessDisabled(c)
			return
		}

		session, _, ok := activeAPIKeyViewerContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		if requestLogProvider == nil {
			writeInternalError(c, "request log provider is not configured", nil)
			return
		}

		eventID, ok := parseUsageEventRequestLogEventID(c)
		if !ok {
			return
		}

		// Verify ownership before fetching log
		if usageProvider == nil {
			writeInternalError(c, "usage provider is unavailable", nil)
			return
		}

		verifier, ok := usageProvider.(interface {
			VerifyUsageEventOwnership(context.Context, int64, string) (bool, error)
		})
		if !ok {
			writeInternalError(c, "ownership verification not supported", nil)
			return
		}

		owned, err := verifier.VerifyUsageEventOwnership(c.Request.Context(), eventID, fmt.Sprintf("%d", session.CPAAPIKeyID))
		if err != nil {
			writeInternalError(c, "verify event ownership failed", err)
			return
		}
		if !owned {
			c.JSON(http.StatusNotFound, gin.H{"error": "usage event not found"})
			return
		}

		response, err := requestLogProvider.GetUsageEventRequestLog(c.Request.Context(), eventID)
		if err != nil {
			writeUsageEventRequestLogError(c, err)
			return
		}

		setNoStoreHeaders(c)
		c.JSON(http.StatusOK, buildUsageEventRequestLogPayload(response))
	})
}

func registerKeyEventExportRoute(
	router gin.IRoutes,
	usageProvider service.UsageProvider,
	usageIdentityProvider service.UsageIdentityProvider,
	exportSlots chan struct{},
) {
	// POST /usage/events/export - Export events (CSV/JSON, scoped)
	router.POST("/key-events/export", func(c *gin.Context) {
		session, _, ok := activeAPIKeyViewerContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		format := strings.ToLower(strings.TrimSpace(c.Query("format")))
		if format == "" {
			format = "csv"
		}
		if format != "csv" && format != "json" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid export format"})
			return
		}

		filter, err := parseUsageExportFilterQuery(c.Request, timeutil.NormalizeStorageTime(time.Now()))
		if err != nil {
			writeUsageFilterParseError(c, err)
			return
		}

		// Scope by viewer's API key
		filter.APIKeyID = fmt.Sprintf("%d", session.CPAAPIKeyID)

		if err := applyUsageEventsSourceFilter(&filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		select {
		case exportSlots <- struct{}{}:
			defer func() { <-exportSlots }()
		default:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "usage events export capacity is full"})
			return
		}

		filter.Limit = 0
		filter.Page = 0
		filter.PageSize = 0
		filter.Offset = 0

		identities, err := loadUsageResolutionData(c, usageIdentityProvider)
		if err != nil {
			writeInternalError(c, "load usage resolution data failed", err)
			return
		}
		resolver := newUsageIdentityResolver(identities)
		apiKeyInfos := map[string]analysisAPIKeyInfo{}

		streamEvents := func(emit func(servicedto.UsageEventRecord) error) error {
			if usageProvider == nil {
				return nil
			}
			return usageProvider.StreamUsageEvents(c.Request.Context(), filter, emit)
		}

		if format == "json" {
			if err := writeUsageEventsJSONExport(c, streamEvents, resolver, apiKeyInfos); err != nil {
				writeUsageEventsExportError(c, err)
			}
			return
		}
		if err := writeUsageEventsCSVExport(c, streamEvents, resolver, apiKeyInfos); err != nil {
			writeUsageEventsExportError(c, err)
		}
	})
}
