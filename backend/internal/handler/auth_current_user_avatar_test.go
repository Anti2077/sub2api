//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func requestCurrentUserAvatar(t *testing.T, handler *AuthHandler, userID int64, version string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware2.PrivateNoStore())
	router.GET("/api/v1/auth/me", func(c *gin.Context) {
		if userID != 0 {
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
		}
		handler.GetCurrentUser(c)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me?avatar_version="+url.QueryEscape(version), nil)
	// The validator is for the avatar alone, never an HTTP 304 for user data.
	request.Header.Set("If-None-Match", version)
	router.ServeHTTP(recorder, request)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Contains(t, recorder.Header().Values("Vary"), "Authorization")
	require.Empty(t, recorder.Header().Get("ETag"))
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	return recorder, envelope.Data
}

func TestAuthHandlerGetCurrentUserAvatarNegotiation(t *testing.T) {
	// Synthetic 600 KiB image payload (800 KiB data URI), never real user data.
	avatar := "data:image/gif;base64," + strings.Repeat("AAAA", 200*1024)
	repo := &userHandlerRepoStub{user: &service.User{
		ID: 31, Email: "me@example.com", Username: "me",
		Role: service.RoleUser, Status: service.StatusActive, Balance: 100,
		AvatarURL: avatar, AvatarSource: "inline",
	}}
	handler := &AuthHandler{userService: service.NewUserService(repo, nil, nil, nil)}

	first, data := requestCurrentUserAvatar(t, handler, 31, "")
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, avatar, data["avatar_url"])
	version := data["avatar_version"].(string)
	require.Regexp(t, `^v1-[a-f0-9]{64}$`, version)

	// Balance, permissions, run mode and identity summaries must still refresh.
	repo.user.Balance = 42
	repo.user.Role = service.RoleAdmin
	repo.user.UpdatedAt = time.Now()
	unchanged, data := requestCurrentUserAvatar(t, handler, 31, version)
	require.Equal(t, http.StatusOK, unchanged.Code)
	require.NotContains(t, data, "avatar_url")
	require.Equal(t, version, data["avatar_version"])
	require.Equal(t, float64(42), data["balance"])
	require.Equal(t, service.RoleAdmin, data["role"])
	require.Equal(t, "standard", data["run_mode"])
	require.Contains(t, data, "identities")
	require.Less(t, unchanged.Body.Len(), 16*1024)
	require.Less(t, unchanged.Body.Len()*100, first.Body.Len())
	t.Logf("synthetic /auth/me response: full=%d bytes, unchanged-avatar=%d bytes, saved=%.2f%%",
		first.Body.Len(), unchanged.Body.Len(), 100*(1-float64(unchanged.Body.Len())/float64(first.Body.Len())))

	// Missing/invalid validators preserve compatibility, even after negotiation.
	for _, legacyVersion := range []string{"", "unknown", `"` + version + `"`} {
		response, full := requestCurrentUserAvatar(t, handler, 31, legacyVersion)
		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, avatar, full["avatar_url"])
	}

	// The same bytes under another owner/source cannot hit this validator.
	for _, change := range []func(){
		func() { repo.user.ID = 32 },
		func() { repo.user.ID = 31; repo.user.AvatarSource = "remote_url" },
		func() { repo.user.AvatarURL = "https://cdn.example.com/new.png" },
	} {
		change()
		response, changed := requestCurrentUserAvatar(t, handler, repo.user.ID, version)
		require.Equal(t, http.StatusOK, response.Code)
		require.NotEqual(t, version, changed["avatar_version"])
		require.Equal(t, repo.user.AvatarURL, changed["avatar_url"])
	}

	// A remote device deleting the image must invalidate the old cached value.
	_, beforeDelete := requestCurrentUserAvatar(t, handler, 31, "")
	beforeDeleteVersion := beforeDelete["avatar_version"].(string)
	repo.user.AvatarURL, repo.user.AvatarSource = "", ""
	deleted, data := requestCurrentUserAvatar(t, handler, 31, beforeDeleteVersion)
	require.Equal(t, http.StatusOK, deleted.Code)
	require.NotContains(t, data, "avatar_url")
	require.NotEqual(t, beforeDeleteVersion, data["avatar_version"])

	emptyVersion := data["avatar_version"].(string)
	for _, cachedVersion := range []string{"", emptyVersion} {
		response, noAvatar := requestCurrentUserAvatar(t, handler, 31, cachedVersion)
		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, emptyVersion, noAvatar["avatar_version"])
		require.NotContains(t, noAvatar, "avatar_url")
	}
}

type currentUserAvatarFailureRepo struct {
	userHandlerRepoStub
	userError       error
	identitiesError error
}

func (r *currentUserAvatarFailureRepo) GetByID(ctx context.Context, id int64) (*service.User, error) {
	if r.userError != nil {
		return nil, r.userError
	}
	return r.userHandlerRepoStub.GetByID(ctx, id)
}

func (r *currentUserAvatarFailureRepo) ListUserAuthIdentities(ctx context.Context, id int64) ([]service.UserAuthIdentityRecord, error) {
	if r.identitiesError != nil {
		return nil, r.identitiesError
	}
	return r.userHandlerRepoStub.ListUserAuthIdentities(ctx, id)
}

func TestAuthHandlerGetCurrentUserAvatarErrors(t *testing.T) {
	for _, tc := range []struct {
		name            string
		userID          int64
		userError       error
		identitiesError error
		status          int
	}{
		{name: "no subject", status: http.StatusUnauthorized},
		{name: "user lookup failed", userID: 31, userError: errors.New("lookup failed"), status: http.StatusInternalServerError},
		{name: "identities failed", userID: 31, identitiesError: errors.New("identities failed"), status: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &currentUserAvatarFailureRepo{
				userHandlerRepoStub: userHandlerRepoStub{user: &service.User{ID: 31, AvatarURL: "private-avatar"}},
				userError:           tc.userError, identitiesError: tc.identitiesError,
			}
			handler := &AuthHandler{userService: service.NewUserService(repo, nil, nil, nil)}
			response, _ := requestCurrentUserAvatar(t, handler, tc.userID, "v1-cached-version")
			require.Equal(t, tc.status, response.Code)
			require.NotContains(t, response.Body.String(), "private-avatar")
			require.NotContains(t, response.Body.String(), "avatar_version")
		})
	}
}
