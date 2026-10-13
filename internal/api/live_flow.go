package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var liveFlowUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
}

func registerLiveFlowRoutes(router gin.IRoutes, client LiveFlowProvider) {
	router.GET("/live-flow/ws", func(ctx *gin.Context) {
		if client == nil {
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "live flow is unavailable"})
			return
		}

		browserConn, err := liveFlowUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}
		defer func() { _ = browserConn.Close() }()

		upstreamConn, response, err := client.DialLiveFlow(ctx.Request.Context())
		if err != nil {
			if response != nil {
				_ = browserConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, http.StatusText(response.StatusCode)))
			}
			return
		}
		defer func() { _ = upstreamConn.Close() }()

		go func() {
			for {
				if _, _, errRead := browserConn.ReadMessage(); errRead != nil {
					_ = upstreamConn.Close()
					return
				}
			}
		}()

		for {
			messageType, payload, errRead := upstreamConn.ReadMessage()
			if errRead != nil {
				return
			}
			if messageType != websocket.TextMessage {
				continue
			}
			if errWrite := browserConn.WriteMessage(websocket.TextMessage, payload); errWrite != nil {
				return
			}
		}
	})
}
