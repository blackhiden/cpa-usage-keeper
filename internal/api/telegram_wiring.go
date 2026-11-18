package api

import (
	"context"
	"fmt"
	"time"

	"cpa-usage-keeper/internal/auth"
	"cpa-usage-keeper/internal/helper"
	"cpa-usage-keeper/internal/modeltest"
	modeltesthttpapi "cpa-usage-keeper/internal/modeltest/httpapi"
	"cpa-usage-keeper/internal/telegram"
	"github.com/gin-gonic/gin"
)

const telegramNotifyTimeout = 15 * time.Second

// telegramReporterResolver derives the reporter label and a non-secret failure
// scope from the authenticated session. The scope never contains the session
// token, so no credential is retained as a lookup key.
func telegramReporterResolver(c *gin.Context) (string, string, bool) {
	sessionValue, hasSession := c.Get(authSessionContextKey)
	session, sessionOK := sessionValue.(auth.Session)
	if !hasSession || !sessionOK {
		// Authentication disabled: single-operator deployment.
		return telegram.AdminReporterLabel, "operator", true
	}
	scope := telegramSessionScope(session)
	if _, key, ok := activeAPIKeyViewerContext(c); ok {
		return helper.CPAAPIKeyDisplayName(key), scope, true
	}
	return telegram.AdminReporterLabel, scope, true
}

func telegramSessionScope(session auth.Session) string {
	return fmt.Sprintf("%s:%d:%d", session.Role, session.CPAAPIKeyID, session.CreatedAt.Unix())
}

// modelTestFailureObserver records every failed probe and triggers the
// automatic notification (REC-010) off the response path.
func modelTestFailureObserver(service *telegram.Service) modeltesthttpapi.ResultObserver {
	if service == nil {
		return nil
	}
	return func(c *gin.Context, request modeltest.Request, result modeltest.Result) {
		if result.Success {
			return
		}
		reporter, scope, ok := telegramReporterResolver(c)
		if !ok {
			return
		}
		service.RecordFailure(scope, telegram.Failure{
			Model:      request.Model,
			Mode:       string(result.Mode),
			StatusCode: result.StatusCode,
			ErrorCode:  result.ErrorCode,
			Reason:     result.Reason,
		})
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), telegramNotifyTimeout)
		go func() {
			defer cancel()
			_ = service.Notify(ctx, telegram.NotifyInput{
				Reporter: reporter,
				Scope:    scope,
				Model:    request.Model,
				Mode:     string(result.Mode),
			})
		}()
	}
}
