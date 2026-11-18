// Package httpapi exposes the read-only model ranking leaderboard over HTTP.
// The payload is metadata only: alias display names, counters, providers, and
// the selected metric's ranked value.
package httpapi

import (
	"context"
	"errors"
	"net/http"

	"cpa-usage-keeper/internal/modelranking"

	"github.com/gin-gonic/gin"
)

// Provider is the model ranking source consumed by the routes; defined here so
// the API layer never imports service internals for this feature.
type Provider interface {
	Leaderboard(context.Context, modelranking.Period, modelranking.Metric) (modelranking.Response, error)
}

// RegisterRoutes mounts GET /model-ranking/leaderboards?period=<value>&metric=<value>.
// The route is always mounted so an unconfigured provider yields a stable 503.
func RegisterRoutes(router gin.IRoutes, provider Provider) {
	router.GET("/model-ranking/leaderboards", func(c *gin.Context) {
		setNoStoreHeaders(c)
		if provider == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model_ranking_unavailable"})
			return
		}
		query := c.Request.URL.Query()
		if len(query) != 2 || len(query["period"]) != 1 || len(query["metric"]) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_model_ranking_selection"})
			return
		}
		period, err := modelranking.ParsePeriod(query.Get("period"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_model_ranking_selection"})
			return
		}
		metric, err := modelranking.ParseMetric(query.Get("metric"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_model_ranking_selection"})
			return
		}
		board, err := provider.Leaderboard(c.Request.Context(), period, metric)
		if err != nil {
			if errors.Is(err, modelranking.ErrInvalidSelection) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_model_ranking_selection"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "model_ranking_failed"})
			return
		}
		c.JSON(http.StatusOK, board)
	})
}

func setNoStoreHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}