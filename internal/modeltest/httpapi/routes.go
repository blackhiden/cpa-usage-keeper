package httpapi

import (
	"net/http"

	"cpa-usage-keeper/internal/modeltest"
	"github.com/gin-gonic/gin"
)

type viewerCredentialResolver func(*gin.Context) string
type authenticatedCredentialResolver func(*gin.Context) (string, bool)

func RegisterRoutes(router gin.IRoutes, service *modeltest.Service) {
	router.POST("/model-test", func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model testing is unavailable", "code": "model_test_unavailable"})
			return
		}
		var request modeltest.Request
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid model test request", "code": "invalid_request"})
			return
		}
		c.JSON(http.StatusOK, service.Test(c.Request.Context(), request, "", false))
	})
}

func RegisterViewerRoutes(router gin.IRoutes, service *modeltest.Service, resolve viewerCredentialResolver) {
	router.POST("/model-test", func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model testing is unavailable", "code": "model_test_unavailable"})
			return
		}
		var request modeltest.Request
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid model test request", "code": "invalid_request"})
			return
		}
		credential := ""
		if resolve != nil {
			credential = resolve(c)
		}
		c.JSON(http.StatusOK, service.Test(c.Request.Context(), request, credential, true))
	})
}

func RegisterAuthenticatedRoutes(router gin.IRoutes, service *modeltest.Service, resolve authenticatedCredentialResolver) {
	router.POST("/model-test", func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model testing is unavailable", "code": "model_test_unavailable"})
			return
		}
		var request modeltest.Request
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid model test request", "code": "invalid_request"})
			return
		}
		credential, viewer := "", false
		if resolve != nil {
			credential, viewer = resolve(c)
		}
		c.JSON(http.StatusOK, service.Test(c.Request.Context(), request, credential, viewer))
	})
}
