package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestAuthAndProfileRoutesRejectUnauthenticatedAvatarValidatorsWithoutCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Auth: &handler.AuthHandler{}, Setting: &handler.SettingHandler{}}
	jwtAuth := servermiddleware.NewJWTAuthMiddleware(nil, nil, nil, nil)
	audit := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: time.Millisecond, MaxRetries: -1})
	t.Cleanup(func() { _ = rdb.Close() })
	RegisterAuthRoutes(router.Group("/api/v1"), handlers, jwtAuth, audit, rdb, nil, nil)
	RegisterUserRoutes(router.Group("/api/v1"), handlers, jwtAuth, audit, nil, nil)
	for _, path := range []string{"/api/v1/auth/me", "/api/v1/user/profile"} {
		for _, auth := range []string{"", "Basic invalid", "Bearer "} {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path+"?avatar_version=v1-any-owner", nil)
			request.Header.Set("Authorization", auth)
			request.Header.Set("If-None-Match", "v1-any-owner")
			router.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusUnauthorized, recorder.Code, "path=%s auth=%q", path, auth)
			require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
			require.Contains(t, recorder.Header().Values("Vary"), "Authorization")
			require.NotContains(t, recorder.Body.String(), "avatar_url")
		}
	}

	// Public auth errors also must not be retained by a CDN or browser cache.
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil))
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
}
