package httpapi

import (
	"context"
	"errors"
	"net/http"

	"cpa-usage-keeper/internal/modelranking"
	"github.com/gin-gonic/gin"
)

type Provider interface {
	Leaderboard(context.Context, modelranking.Period, modelranking.Metric) (modelranking.Response, error)
}

type ScopedProvider interface {
	Provider
	LeaderboardWithOptions(context.Context, modelranking.Period, modelranking.Metric, modelranking.LeaderboardOptions) (modelranking.Response, error)
}

type ViewerScope struct{ ScopeKey string }
type ViewerScopeResolver func(*gin.Context) (ViewerScope, bool)

func RegisterRoutes(router gin.IRoutes, provider Provider) {
	router.GET("/model-ranking/leaderboards", func(c *gin.Context) {
		setNoStoreHeaders(c)
		if provider == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model_ranking_unavailable"})
			return
		}
		query := c.Request.URL.Query()
		for key := range query {
			if key != "period" && key != "metric" && key != "provider" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_model_ranking_selection"})
				return
			}
		}
		if len(query["period"]) != 1 || len(query["metric"]) != 1 || len(query["provider"]) > 1 {
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
		var board modelranking.Response
		if scoped, ok := provider.(ScopedProvider); ok {
			board, err = scoped.LeaderboardWithOptions(c.Request.Context(), period, metric, modelranking.LeaderboardOptions{Provider: query.Get("provider"), IncludeTrend: true})
		} else {
			board, err = provider.Leaderboard(c.Request.Context(), period, metric)
		}
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

func RegisterKeyViewerRoutes(router gin.IRoutes, provider ScopedProvider, resolve ViewerScopeResolver) {
	router.GET("/key-model-ranking/leaderboards", func(c *gin.Context) {
		setNoStoreHeaders(c)
		if provider == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model_ranking_unavailable"})
			return
		}
		if resolve == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		scope, ok := resolve(c)
		if !ok || scope.ScopeKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		query := c.Request.URL.Query()
		for key := range query {
			if key != "period" && key != "metric" && key != "provider" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_model_ranking_selection"})
				return
			}
		}
		if len(query["period"]) != 1 || len(query["metric"]) != 1 || len(query["provider"]) > 1 {
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
		board, err := provider.LeaderboardWithOptions(c.Request.Context(), period, metric, modelranking.LeaderboardOptions{ScopeKey: scope.ScopeKey, Provider: query.Get("provider"), IncludeTrend: true})
		if err != nil {
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
