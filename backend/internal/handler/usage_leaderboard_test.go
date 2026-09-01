package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	rows      []usagestats.UserBreakdownItem
	called    bool
	startTime time.Time
	endTime   time.Time
	dimension usagestats.UserBreakdownDimension
	limit     int
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
	usageService := service.NewUsageService(repo, nil, nil, nil)
	usageHandler := NewUsageHandler(usageService, nil, nil, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
		c.Next()
	})
	router.GET("/usage/leaderboard", usageHandler.PublicTokenLeaderboard)
	return router
}

func TestPublicTokenLeaderboardMasksIdentityAndUsesTokenSort(t *testing.T) {
	repo := &leaderboardUsageRepoStub{rows: []usagestats.UserBreakdownItem{
		{UserID: 42, Email: "alice@example.com", Requests: 12, InputTokens: 100, OutputTokens: 20, CacheTokens: 30, TotalTokens: 150},
		{UserID: 7, Email: "赵@example.cn", Requests: 4, InputTokens: 40, OutputTokens: 5, CacheTokens: 0, TotalTokens: 45},
	}}
	router := newLeaderboardTestRouter(repo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/leaderboard?period=day&timezone=Asia/Shanghai", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.True(t, repo.called)
	require.Equal(t, "total_tokens", repo.dimension.SortBy)
	require.Equal(t, 20, repo.limit)
	require.Contains(t, recorder.Body.String(), `"masked_email":"a***e@example.com"`)
	require.Contains(t, recorder.Body.String(), `"masked_email":"赵***@example.cn"`)
	require.Contains(t, recorder.Body.String(), `"is_current_user":true`)
	require.NotContains(t, recorder.Body.String(), "alice@example.com")
	require.NotContains(t, recorder.Body.String(), `"user_id"`)
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

func TestPublicLeaderboardWeekStartsMondayAndEndsTomorrow(t *testing.T) {
	period, startTime, endTime, ok := publicLeaderboardRange("week", "Asia/Shanghai")
	require.True(t, ok)
	require.Equal(t, "week", period)
	require.Equal(t, time.Monday, startTime.Weekday())
	require.Equal(t, 0, startTime.Hour())
	require.Equal(t, 24*time.Hour, endTime.Sub(time.Date(endTime.Year(), endTime.Month(), endTime.Day()-1, 0, 0, 0, 0, endTime.Location())))
	require.True(t, startTime.Before(endTime))
}
