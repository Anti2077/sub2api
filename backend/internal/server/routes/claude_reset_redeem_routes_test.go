package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// The Claude redeem route consumes an irreversible credit just like the Codex
// reset-quota route, so it must sit behind exactly the same security middleware
// chain before the account routes' additional folder filter.
func TestClaudeResetRedeemRouteMatchesCodexResetProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{Account: &adminhandler.AccountHandler{}, OpenAIOAuth: &adminhandler.OpenAIOAuthHandler{}}}
	chains := map[string][]string{}
	adminAuth := servermiddleware.AdminAuthMiddleware(func(c *gin.Context) {
		names := c.HandlerNames()
		chains[c.FullPath()] = names[:len(names)-1]
		if c.GetHeader("Authorization") == "" {
			servermiddleware.AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization required")
			return
		}
		servermiddleware.AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
	})
	auditLog := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })
	stepUp := servermiddleware.StepUpAuthMiddleware(func(c *gin.Context) { c.Next() })
	RegisterAdminRoutes(router.Group("/api/v1"), handlers, adminAuth, auditLog, stepUp, nil, nil)

	codex := "/api/v1/admin/openai/accounts/1/reset-quota"
	claude := "/api/v1/admin/accounts/1/claude/reset-credits/redeem"
	for _, path := range []string{codex, claude} {
		for _, query := range []string{"", "?folder=invalid"} {
			for _, auth := range []string{"", "Bearer user-token"} {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodPost, path+query, nil)
				if auth != "" {
					request.Header.Set("Authorization", auth)
				}
				router.ServeHTTP(recorder, request)
				if auth == "" {
					require.Equal(t, http.StatusUnauthorized, recorder.Code, path)
				} else {
					require.Equal(t, http.StatusForbidden, recorder.Code, path)
				}
			}
		}
	}
	codexChain := chains["/api/v1/admin/openai/accounts/:id/reset-quota"]
	claudeChain := chains["/api/v1/admin/accounts/:id/claude/reset-credits/redeem"]
	require.NotEmpty(t, codexChain)
	// Keep the exact order and every security layer; the folder filter must run
	// after them, including when its query is invalid.
	expectedClaudeChain := append(append([]string{}, codexChain...),
		"github.com/Wei-Shaw/sub2api/internal/server/routes.registerAccountRoutes.func1")
	require.Equal(t, strings.Join(expectedClaudeChain, "\n"), strings.Join(claudeChain, "\n"))
}
