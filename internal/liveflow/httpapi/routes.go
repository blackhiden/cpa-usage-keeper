// Package liveflowhttpapi exposes the Live Flow model history over HTTP.
// The payload is metadata only: model id, counts, status, latency, last-seen.
package liveflowhttpapi

import (
	"context"
	"net/http"

	"cpa-usage-keeper/internal/liveflow"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Provider is the Live Flow history source consumed by the routes; defined
// here so the API layer never imports service internals for this feature.
type Provider interface {
	Models(ctx context.Context, period liveflow.Period) (liveflow.ModelsResponse, error)
}

// DefaultPeriod is served when the request omits the period query parameter.
const DefaultPeriod = liveflow.PeriodToday

// RegisterRoutes exposes GET /live-flow/models?period=<value>.
func RegisterRoutes(router gin.IRoutes, provider Provider) {
	router.GET("/live-flow/models", func(c *gin.Context) {
		if provider == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "live flow history is unavailable"})
			return
		}

		period := DefaultPeriod
		if raw := c.Query("period"); raw != "" {
			parsed, err := liveflow.ParsePeriod(raw)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unknown period"})
				return
			}
			period = parsed
		}

		response, err := provider.Models(c.Request.Context(), period)
		if err != nil {
			logrus.WithError(err).Error("live flow model history failed")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, response)
	})
}
