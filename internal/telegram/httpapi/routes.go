// Package telegramhttpapi exposes model-test failure notifications over HTTP.
// The reporter identity is always resolved from the authenticated session; the
// request body cannot carry it.
package telegramhttpapi

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"cpa-usage-keeper/internal/telegram"
	"github.com/gin-gonic/gin"
)

// ReporterResolver derives the reporter label from the request context.
type ReporterResolver func(*gin.Context) (string, bool)

// RegisterRoutes exposes POST /model-test/notify.
func RegisterRoutes(router gin.IRoutes, service *telegram.Service, resolve ReporterResolver) {
	router.POST("/model-test/notify", func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "telegram notifications are unavailable", "code": "telegram_unavailable"})
			return
		}
		var request telegram.Request
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification request", "code": telegram.CodeInvalidRequest})
			return
		}
		reporter := ""
		if resolve != nil {
			var ok bool
			reporter, ok = resolve(c)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
				return
			}
		}
		if err := service.Notify(c.Request.Context(), reporter, request); err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"sent": true})
	})
}

// writeError maps domain errors to stable codes. No domain error text is
// echoed, so a Bot API response or token fragment cannot reach the client.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, telegram.ErrInvalidRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification request", "code": telegram.CodeInvalidRequest})
	case errors.Is(err, telegram.ErrNotConfigured):
		c.JSON(http.StatusConflict, gin.H{"error": "telegram notifications are not configured", "code": telegram.CodeNotConfigured})
	case errors.Is(err, telegram.ErrRateLimited):
		wait := time.Minute
		var rateLimit *telegram.RateLimitError
		if errors.As(err, &rateLimit) && rateLimit.RetryAfter > 0 {
			wait = rateLimit.RetryAfter
		}
		c.Header("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many notifications", "code": telegram.CodeRateLimited})
	case errors.Is(err, telegram.ErrSendFailed):
		c.JSON(http.StatusBadGateway, gin.H{"error": "telegram delivery failed", "code": telegram.CodeSendFailed})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
