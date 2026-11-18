package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"cpa-usage-keeper/internal/adminsettings"
	"github.com/gin-gonic/gin"
)

type telegramUpdateRequest struct {
	BotToken        *string `json:"bot_token"`
	ChatID          *string `json:"chat_id"`
	MessageThreadID *int64  `json:"message_thread_id"`
}

type updateRequest struct {
	APIKey                *string                `json:"api_key"`
	RequestTimeoutSeconds *int64                 `json:"request_timeout_seconds"`
	Telegram              *telegramUpdateRequest `json:"telegram"`
}

// toUpdate maps the wire payload onto the service update, keeping omitted
// Telegram fields untouched.
func (r updateRequest) toUpdate() adminsettings.Update {
	update := adminsettings.Update{APIKey: r.APIKey, RequestTimeoutSeconds: r.RequestTimeoutSeconds}
	if r.Telegram != nil {
		update.Telegram = &adminsettings.TelegramUpdate{
			BotToken:        r.Telegram.BotToken,
			ChatID:          r.Telegram.ChatID,
			MessageThreadID: r.Telegram.MessageThreadID,
		}
	}
	return update
}

func RegisterRoutes(router gin.IRoutes, service *adminsettings.Service) {
	router.GET("/admin-settings", func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "admin settings are unavailable", "code": "admin_settings_unavailable"})
			return
		}
		settings, err := service.Get(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "admin settings are unavailable", "code": "admin_settings_unavailable"})
			return
		}
		c.JSON(http.StatusOK, settings)
	})
	router.PUT("/admin-settings", func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "admin settings are unavailable", "code": "admin_settings_unavailable"})
			return
		}
		var request updateRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin settings", "code": "invalid_admin_settings"})
			return
		}
		settings, err := service.Save(c.Request.Context(), request.toUpdate())
		if err != nil {
			if errors.Is(err, adminsettings.ErrProtectionKeyRequired) {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "admin settings protection is not configured", "code": "admin_settings_protection_unavailable"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "invalid_admin_settings"})
			return
		}
		c.JSON(http.StatusOK, settings)
	})
}

func FormatTimeout(seconds int64) string {
	return strconv.FormatInt(seconds, 10)
}
