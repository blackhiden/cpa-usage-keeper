package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var liveFlowUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
}

type liveFlowRelayError struct {
	Kind      string `json:"kind"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func newLiveFlowRelayError(response *http.Response) liveFlowRelayError {
	if response == nil {
		return liveFlowRelayError{
			Kind:      "keeper_live_flow_error",
			Code:      "upstream_unreachable",
			Message:   "Keeper could not connect to CLIProxyAPI Live Flow.",
			Retryable: true,
		}
	}

	status := response.StatusCode
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return liveFlowRelayError{
			Kind:      "keeper_live_flow_error",
			Code:      "upstream_auth_required",
			Message:   "CLIProxyAPI rejected the Live Flow connection. Configure CPA_LIVE_FLOW_API_KEY when root ws-auth is enabled.",
			Retryable: false,
		}
	}
	if status == http.StatusNotFound || status == http.StatusGone {
		return liveFlowRelayError{
			Kind:      "keeper_live_flow_error",
			Code:      "upstream_feature_disabled",
			Message:   "CLIProxyAPI Live Flow is unavailable. Enable cpa-fork-liveflow-enabled and restart CLIProxyAPI.",
			Retryable: false,
		}
	}
	return liveFlowRelayError{
		Kind:      "keeper_live_flow_error",
		Code:      "upstream_rejected",
		Message:   "CLIProxyAPI rejected the Live Flow connection.",
		Retryable: status >= http.StatusInternalServerError,
	}
}

func writeLiveFlowRelayError(conn *websocket.Conn, relayError liveFlowRelayError) {
	payload, err := json.Marshal(relayError)
	if err != nil {
		return
	}
	_ = conn.WriteMessage(websocket.TextMessage, payload)
	closeCode := websocket.CloseTryAgainLater
	if !relayError.Retryable {
		closeCode = websocket.ClosePolicyViolation
	}
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, relayError.Code))
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
			relayError := newLiveFlowRelayError(response)
			if response != nil && response.Body != nil {
				_ = response.Body.Close()
			}
			log.WithFields(log.Fields{
				"code":      relayError.Code,
				"retryable": relayError.Retryable,
			}).WithError(err).Warn("live flow upstream connection failed")
			writeLiveFlowRelayError(browserConn, relayError)
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
