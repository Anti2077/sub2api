package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type leaderboardUsageRepoStub struct {
	service.UsageLogRepository
	rows              []usagestats.UserBreakdownItem
	avatarURLs        map[int64]string
	avatar            *service.UserAvatar
	avatarErr         error
	avatarReads       int
	usernameConfirmed *bool
	avatarUserIDs     []int64
	called            bool
	startTime         time.Time
	endTime           time.Time
	dimension         usagestats.UserBreakdownDimension
	limit             int
}

type leaderboardUserRepoStub struct {
	service.UserRepository
	usageRepo *leaderboardUsageRepoStub
}

func (s *leaderboardUserRepoStub) GetByID(_ context.Context, userID int64) (*service.User, error) {
	confirmed := true
	if s.usageRepo.usernameConfirmed != nil {
		confirmed = *s.usageRepo.usernameConfirmed
	}
	return &service.User{ID: userID, Username: "current", UsernameConfirmed: confirmed}, nil
}

func (s *leaderboardUserRepoStub) GetLeaderboardAvatarReferences(_ context.Context, userIDs []int64) (map[int64]service.LeaderboardAvatarReference, error) {
	s.usageRepo.avatarUserIDs = append([]int64(nil), userIDs...)
	refs := make(map[int64]service.LeaderboardAvatarReference)
	for _, id := range userIDs {
		url := s.usageRepo.avatarURLs[id]
		if strings.HasPrefix(url, "data:") {
			sum := md5.Sum([]byte(url))
			refs[id] = service.LeaderboardAvatarReference{Version: hex.EncodeToString(sum[:])}
		} else {
			refs[id] = service.LeaderboardAvatarReference{URL: url}
		}
	}
	return refs, nil
}

func (s *leaderboardUserRepoStub) GetLeaderboardAvatar(_ context.Context, username, version string) (*service.UserAvatar, error) {
	s.usageRepo.avatarReads++
	return s.usageRepo.avatar, s.usageRepo.avatarErr
}

func (s *leaderboardUsageRepoStub) GetUserBreakdownStats(
	_ context.Context,
	startTime, endTime time.Time,
	dimension usagestats.UserBreakdownDimension,
	limit int,
) ([]usagestats.UserBreakdownItem, error) {
	s.called = true
	s.startTime = startTime
	s.endTime = endTime
	s.dimension = dimension
	s.limit = limit
	return s.rows, nil
}

func newLeaderboardTestRouter(repo *leaderboardUsageRepoStub, userID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	userRepo := &leaderboardUserRepoStub{usageRepo: repo}
	usageService := service.NewUsageService(repo, userRepo, nil, nil)
	usageHandler := NewUsageHandler(usageService, nil, nil, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
		c.Next()
	})
	router.Use(middleware2.PrivateNoStore())
	router.GET("/usage/leaderboard", usageHandler.PublicTokenLeaderboard)
	router.GET("/usage/leaderboard/avatar/:username/:version", usageHandler.LeaderboardAvatar)
	return router
}

func TestPublicTokenLeaderboardUsesUsernameAndHidesPrivateIdentity(t *testing.T) {
	repo := &leaderboardUsageRepoStub{rows: []usagestats.UserBreakdownItem{
		{UserID: 42, Email: "alice@example.com", Username: "alice", UsernameConfirmed: true, Requests: 12, InputTokens: 100, OutputTokens: 20, CacheTokens: 30, TotalTokens: 150, ActualCost: 2.5},
		{UserID: 7, Email: "赵@example.cn", Username: "zhao", UsernameConfirmed: true, LeaderboardAnonymous: true, Requests: 4, InputTokens: 40, OutputTokens: 5, CacheTokens: 0, TotalTokens: 45, ActualCost: 9.75},
	}, avatarURLs: map[int64]string{42: "https://cdn.example.com/alice.png", 7: "https://cdn.example.com/zhao.png"}}
	router := newLeaderboardTestRouter(repo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/leaderboard?period=day&timezone=Asia/Shanghai", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.True(t, repo.called)
	require.Equal(t, "total_tokens", repo.dimension.SortBy)
	require.Equal(t, 20, repo.limit)
	require.Contains(t, recorder.Body.String(), `"username":"alice"`)
	require.Contains(t, recorder.Body.String(), `"avatar_url":"https://cdn.example.com/alice.png"`)
	require.Contains(t, recorder.Body.String(), `"is_anonymous":true`)
	require.Contains(t, recorder.Body.String(), `"is_current_user":true`)
	require.NotContains(t, recorder.Body.String(), "alice@example.com")
	require.NotContains(t, recorder.Body.String(), `"user_id"`)
	require.NotContains(t, recorder.Body.String(), `"masked_email"`)
	require.NotContains(t, recorder.Body.String(), `"email"`)
	require.NotContains(t, recorder.Body.String(), "zhao.png")
	require.Equal(t, []int64{42}, repo.avatarUserIDs)
}

func TestPublicTokenLeaderboardRejectsInvalidPeriod(t *testing.T) {
	repo := &leaderboardUsageRepoStub{}
	router := newLeaderboardTestRouter(repo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/leaderboard?period=quarter", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.False(t, repo.called)
}

func TestPublicLeaderboardSupportsSpendingMode(t *testing.T) {
	repo := &leaderboardUsageRepoStub{rows: []usagestats.UserBreakdownItem{
		{UserID: 42, Username: "alice", UsernameConfirmed: true, ActualCost: 12.5, TotalTokens: 100},
	}}
	router := newLeaderboardTestRouter(repo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/leaderboard?period=month&mode=spending", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "actual_cost", repo.dimension.SortBy)
	require.Contains(t, recorder.Body.String(), `"mode":"spending"`)
	require.Contains(t, recorder.Body.String(), `"actual_cost":12.5`)
}

func TestPublicLeaderboardRejectsInvalidMode(t *testing.T) {
	repo := &leaderboardUsageRepoStub{}
	router := newLeaderboardTestRouter(repo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/leaderboard?mode=money", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.False(t, repo.called)
}

func TestPublicLeaderboardWeekStartsMondayAndEndsTomorrow(t *testing.T) {
	period, startTime, endTime, ok := publicLeaderboardRange("week", "Asia/Shanghai")
	require.True(t, ok)
	require.Equal(t, "week", period)
	require.Equal(t, time.Monday, startTime.Weekday())
	require.Equal(t, 0, startTime.Hour())
	require.Equal(t, 24*time.Hour, endTime.Sub(time.Date(endTime.Year(), endTime.Month(), endTime.Day()-1, 0, 0, 0, 0, endTime.Location())))
	require.True(t, startTime.Before(endTime))
}

func TestPublicLeaderboardDoesNotInlineLargeAvatars(t *testing.T) {
	image := "data:image/gif;base64," + strings.Repeat("AAAA", 200*1024)
	repo := &leaderboardUsageRepoStub{rows: []usagestats.UserBreakdownItem{{UserID: 42, Username: "alice", UsernameConfirmed: true, TotalTokens: 150}}, avatarURLs: map[int64]string{42: image}}
	recorder := httptest.NewRecorder()
	newLeaderboardTestRouter(repo, 42).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/usage/leaderboard", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Less(t, recorder.Body.Len(), 1024)
	require.Contains(t, recorder.Body.String(), `"avatar_version":`)
	require.NotContains(t, recorder.Body.String(), "data:image")
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	t.Logf("ranking response with synthetic 800 KiB avatar: %d bytes", recorder.Body.Len())
}

func TestLeaderboardAvatarCacheAndVisibility(t *testing.T) {
	version := strings.Repeat("a", 32)
	for _, tc := range []struct {
		name, validator string
		missing         bool
		status          int
	}{
		{name: "first fetch", status: 200},
		{name: "unchanged", validator: `"` + version + `"`, status: 304},
		{name: "weak validator list", validator: `"old", W/"` + version + `"`, status: 304},
		{name: "changed", validator: `"old"`, status: 200},
		{name: "hidden or deleted even with matching validator", validator: `"` + version + `"`, missing: true, status: 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &leaderboardUsageRepoStub{}
			if !tc.missing {
				repo.avatar = &service.UserAvatar{URL: "data:image/gif;base64,R0lGODlh"}
			}
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/usage/leaderboard/avatar/alice/"+version, nil)
			req.Header.Set("If-None-Match", tc.validator)
			newLeaderboardTestRouter(repo, 42).ServeHTTP(recorder, req)
			require.Equal(t, tc.status, recorder.Code)
			require.Equal(t, 1, repo.avatarReads)
			if tc.status == 200 {
				require.Equal(t, "image/gif", recorder.Header().Get("Content-Type"))
				require.Equal(t, "GIF89a", recorder.Body.String())
			}
			if tc.status == 304 {
				require.Empty(t, recorder.Body.String())
			}
			if tc.missing {
				require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
			} else {
				require.Equal(t, "private, no-cache", recorder.Header().Get("Cache-Control"))
				require.Equal(t, fmt.Sprintf("%q", version), recorder.Header().Get("ETag"))
			}
			require.Contains(t, recorder.Header().Values("Vary"), "Authorization")
		})
	}
}

func TestLeaderboardAvatarRejectsInvalidContentAndVersions(t *testing.T) {
	for _, image := range []string{"data:image/svg+xml;base64,PHN2Zz4=", "data:image/gif;base64,%%%", "https://example.com/image.png"} {
		repo := &leaderboardUsageRepoStub{avatar: &service.UserAvatar{URL: image}}
		recorder := httptest.NewRecorder()
		newLeaderboardTestRouter(repo, 42).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/usage/leaderboard/avatar/alice/"+strings.Repeat("a", 32), nil))
		require.Equal(t, 404, recorder.Code)
		require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	}
	repo := &leaderboardUsageRepoStub{}
	recorder := httptest.NewRecorder()
	newLeaderboardTestRouter(repo, 42).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/usage/leaderboard/avatar/alice/invalid", nil))
	require.Equal(t, 404, recorder.Code)
	require.Zero(t, repo.avatarReads)
	confirmed := false
	repo.usernameConfirmed = &confirmed
	recorder = httptest.NewRecorder()
	newLeaderboardTestRouter(repo, 42).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/usage/leaderboard/avatar/alice/"+strings.Repeat("a", 32), nil))
	require.NotEqual(t, 200, recorder.Code)
	require.Zero(t, repo.avatarReads)
}
