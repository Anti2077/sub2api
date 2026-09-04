package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageEquivalenceRepoStub struct {
	service.UsageLogRepository
	stats   *usagestats.UsageStats
	called  bool
	filters usagestats.UsageLogFilters
}

type usageEquivalenceUserRepoStub struct {
	service.UserRepository
	user   *service.User
	err    error
	called bool
	userID int64
}

type usageEquivalenceSettingRepoStub struct {
	service.SettingRepository
	values map[string]string
	err    error
}

func (s *usageEquivalenceSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *usageEquivalenceUserRepoStub) GetByID(_ context.Context, userID int64) (*service.User, error) {
	s.called = true
	s.userID = userID
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func (s *usageEquivalenceRepoStub) GetStatsWithFilters(_ context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	s.called = true
	s.filters = filters
	if s.stats == nil {
		return &usagestats.UsageStats{}, nil
	}
	return s.stats, nil
}

func newUsageEquivalenceTestRouter(repo *usageEquivalenceRepoStub, userID int64) *gin.Engine {
	return newUsageEquivalenceTestRouterWithSettings(repo, nil, userID, map[string]string{
		service.SettingKeyUsageEquivalenceEnabled:         "true",
		service.SettingKeyUsageEquivalencePlus7DLimitUSD:  "100",
		service.SettingKeyUsageEquivalencePlus30DLimitUSD: "400",
	})
}

func newUsageEquivalenceTestRouterWithUserRepo(repo *usageEquivalenceRepoStub, userRepo service.UserRepository, userID int64) *gin.Engine {
	return newUsageEquivalenceTestRouterWithSettings(repo, userRepo, userID, map[string]string{
		service.SettingKeyUsageEquivalenceEnabled:         "true",
		service.SettingKeyUsageEquivalencePlus7DLimitUSD:  "100",
		service.SettingKeyUsageEquivalencePlus30DLimitUSD: "400",
	})
}

func newUsageEquivalenceTestRouterWithSettings(repo *usageEquivalenceRepoStub, userRepo service.UserRepository, userID int64, values map[string]string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageService := service.NewUsageService(repo, userRepo, nil, nil)
	settingService := service.NewSettingService(&usageEquivalenceSettingRepoStub{values: values}, &config.Config{})
	handler := NewUsageHandler(usageService, nil, nil, settingService)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
		c.Next()
	})
	router.GET("/usage/equivalence", handler.UsageEquivalence)
	return router
}

func TestUsageEquivalenceRangeUsesExactRollingWindows(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, time.September, 4, 15, 30, 0, 0, location)

	for _, testCase := range []struct {
		period   string
		duration time.Duration
	}{
		{period: usageEquivalencePeriodLast24Hours, duration: 24 * time.Hour},
		{period: usageEquivalencePeriodLast7Days, duration: 7 * 24 * time.Hour},
		{period: usageEquivalencePeriodLast30Days, duration: 30 * 24 * time.Hour},
	} {
		t.Run(testCase.period, func(t *testing.T) {
			period, timezoneName, startTime, endTime, ok := usageEquivalenceRange(testCase.period, "Asia/Shanghai", now, nil)
			require.True(t, ok)
			require.Equal(t, testCase.period, period)
			require.Equal(t, "Asia/Shanghai", timezoneName)
			require.Equal(t, testCase.duration, endTime.Sub(startTime))
			require.Equal(t, now, endTime)
		})
	}
}

func TestUsageEquivalenceRangeUsesCalendarMonthInUserTimezone(t *testing.T) {
	location, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	now := time.Date(2026, time.September, 4, 8, 15, 0, 0, location)

	period, timezoneName, startTime, endTime, ok := usageEquivalenceRange("", "America/Los_Angeles", now, nil)

	require.True(t, ok)
	require.Equal(t, usageEquivalencePeriodThisMonth, period)
	require.Equal(t, "America/Los_Angeles", timezoneName)
	require.Equal(t, time.Date(2026, time.September, 1, 0, 0, 0, 0, location), startTime)
	require.Equal(t, now, endTime)
}

func TestUsageEquivalenceRangeRejectsInvalidInput(t *testing.T) {
	_, _, _, _, ok := usageEquivalenceRange("last_year", "Asia/Shanghai", time.Now(), nil)
	require.False(t, ok)

	_, _, _, _, ok = usageEquivalenceRange(usageEquivalencePeriodLast24Hours, "Not/A_Timezone", time.Now(), nil)
	require.False(t, ok)
}

func TestUsageEquivalenceRangeUsesSixCalendarMonths(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, time.August, 31, 15, 30, 0, 0, location)

	period, timezoneName, startTime, endTime, ok := usageEquivalenceRange(
		usageEquivalencePeriodLast6Months,
		"Asia/Shanghai",
		now,
		nil,
	)

	require.True(t, ok)
	require.Equal(t, usageEquivalencePeriodLast6Months, period)
	require.Equal(t, "Asia/Shanghai", timezoneName)
	require.Equal(t, now.AddDate(0, -6, 0), startTime)
	require.Equal(t, now, endTime)
}

func TestUsageEquivalenceRangeUsesRegistrationTimeForAllTime(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, time.September, 4, 15, 30, 0, 0, location)
	registeredAt := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

	period, timezoneName, startTime, endTime, ok := usageEquivalenceRange(
		usageEquivalencePeriodAllTime,
		"Asia/Shanghai",
		now,
		&registeredAt,
	)

	require.True(t, ok)
	require.Equal(t, usageEquivalencePeriodAllTime, period)
	require.Equal(t, "Asia/Shanghai", timezoneName)
	require.True(t, registeredAt.Equal(startTime))
	require.Equal(t, location, startTime.Location())
	require.Equal(t, now, endTime)
}

func TestUsageEquivalenceRangeRequiresRegistrationTimeForAllTime(t *testing.T) {
	_, _, _, _, ok := usageEquivalenceRange(
		usageEquivalencePeriodAllTime,
		"UTC",
		time.Now(),
		nil,
	)
	require.False(t, ok)
}

func TestBuildUsageEquivalenceResponseUsesRecordedStandardCost(t *testing.T) {
	startTime := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(24 * time.Hour)
	got := buildUsageEquivalenceResponse(
		usageEquivalencePeriodLast24Hours,
		"UTC",
		startTime,
		endTime,
		&usagestats.UsageStats{
			TotalCost:       260,
			TotalActualCost: 78,
			TotalRequests:   11,
			TotalTokens:     9000,
		},
		100,
		400,
	)

	require.Equal(t, 260.0, got.StandardCost)
	require.Equal(t, 78.0, got.ActualCost)
	require.InDelta(t, 0.3, got.EffectiveRateMultiplier, 1e-12)
	require.Equal(t, int64(11), got.TotalRequests)
	require.Equal(t, int64(9000), got.TotalTokens)
	require.Len(t, got.Plans, 3)
	require.Equal(t, "chatgpt_plus", got.Plans[0].ID)
	require.Equal(t, 1, got.Plans[0].UsageMultiple)
	require.Equal(t, 100.0, got.Plans[0].Quota7DStandardCost)
	require.Equal(t, 400.0, got.Plans[0].Quota30DStandardCost)
	require.Equal(t, 2.6, got.Plans[0].Equivalent7DWindows)
	require.Equal(t, 0.65, got.Plans[0].Equivalent30DWindows)
	require.Equal(t, "chatgpt_pro_5x", got.Plans[1].ID)
	require.Equal(t, 5, got.Plans[1].UsageMultiple)
	require.Equal(t, 500.0, got.Plans[1].Quota7DStandardCost)
	require.Equal(t, 2000.0, got.Plans[1].Quota30DStandardCost)
	require.Equal(t, 0.52, got.Plans[1].Equivalent7DWindows)
	require.Equal(t, 0.13, got.Plans[1].Equivalent30DWindows)
	require.Equal(t, "chatgpt_pro_20x", got.Plans[2].ID)
	require.Equal(t, 20, got.Plans[2].UsageMultiple)
	require.Equal(t, 2000.0, got.Plans[2].Quota7DStandardCost)
	require.Equal(t, 8000.0, got.Plans[2].Quota30DStandardCost)
	require.Equal(t, 0.13, got.Plans[2].Equivalent7DWindows)
	require.Equal(t, 0.0325, got.Plans[2].Equivalent30DWindows)
	require.Equal(t, openAIPlanReferenceSource, got.ReferenceSource)
	require.Equal(t, usageEquivalenceDisclaimer, got.Disclaimer)
}

func TestBuildUsageEquivalenceResponseHandlesZeroAndNegativeCosts(t *testing.T) {
	got := buildUsageEquivalenceResponse(
		usageEquivalencePeriodThisMonth,
		"UTC",
		time.Time{},
		time.Time{},
		&usagestats.UsageStats{TotalCost: -1, TotalActualCost: -2},
		100,
		400,
	)

	require.Zero(t, got.StandardCost)
	require.Zero(t, got.ActualCost)
	require.Zero(t, got.EffectiveRateMultiplier)
	for _, plan := range got.Plans {
		require.Zero(t, plan.Equivalent7DWindows)
		require.Zero(t, plan.Equivalent30DWindows)
	}
}

func TestUsageEquivalenceReturnsNotFoundWhenDisabled(t *testing.T) {
	repo := &usageEquivalenceRepoStub{}
	router := newUsageEquivalenceTestRouterWithSettings(repo, nil, 42, map[string]string{
		service.SettingKeyUsageEquivalenceEnabled:         "false",
		service.SettingKeyUsageEquivalencePlus7DLimitUSD:  "100",
		service.SettingKeyUsageEquivalencePlus30DLimitUSD: "400",
	})

	req := httptest.NewRequest(http.MethodGet, "/usage/equivalence", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.False(t, repo.called)
}

func TestUsageEquivalenceRejectsMissingQuotaReferences(t *testing.T) {
	repo := &usageEquivalenceRepoStub{}
	router := newUsageEquivalenceTestRouterWithSettings(repo, nil, 42, map[string]string{
		service.SettingKeyUsageEquivalenceEnabled: "true",
	})

	req := httptest.NewRequest(http.MethodGet, "/usage/equivalence", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.False(t, repo.called)
}

func TestUsageEquivalenceScopesAggregationToAuthenticatedUser(t *testing.T) {
	repo := &usageEquivalenceRepoStub{stats: &usagestats.UsageStats{
		TotalCost:       100,
		TotalActualCost: 25,
		TotalRequests:   4,
		TotalTokens:     1200,
	}}
	router := newUsageEquivalenceTestRouter(repo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/equivalence?period=last_24h&timezone=Asia%2FShanghai", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, repo.called)
	require.Equal(t, int64(42), repo.filters.UserID)
	require.NotNil(t, repo.filters.StartTime)
	require.NotNil(t, repo.filters.EndTime)
	require.Equal(t, 24*time.Hour, repo.filters.EndTime.Sub(*repo.filters.StartTime))

	var body struct {
		Data usageEquivalenceResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, usageEquivalencePeriodLast24Hours, body.Data.Period)
	require.Equal(t, "Asia/Shanghai", body.Data.Timezone)
	require.Equal(t, usageEquivalenceScope, body.Data.Scope)
	require.Equal(t, 100.0, body.Data.StandardCost)
	require.Equal(t, 25.0, body.Data.ActualCost)
	require.Equal(t, 0.25, body.Data.EffectiveRateMultiplier)
}

func TestUsageEquivalenceAllTimeStartsAtAuthenticatedUserRegistration(t *testing.T) {
	registeredAt := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	usageRepo := &usageEquivalenceRepoStub{}
	userRepo := &usageEquivalenceUserRepoStub{user: &service.User{
		ID:        42,
		CreatedAt: registeredAt,
	}}
	router := newUsageEquivalenceTestRouterWithUserRepo(usageRepo, userRepo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/equivalence?period=all_time&timezone=Asia%2FShanghai", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, userRepo.called)
	require.Equal(t, int64(42), userRepo.userID)
	require.True(t, usageRepo.called)
	require.True(t, registeredAt.Equal(*usageRepo.filters.StartTime))

	var body struct {
		Data usageEquivalenceResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, usageEquivalencePeriodAllTime, body.Data.Period)
	require.True(t, registeredAt.Equal(body.Data.StartTime))
}

func TestUsageEquivalenceAllTimeStopsWhenUserLookupFails(t *testing.T) {
	usageRepo := &usageEquivalenceRepoStub{}
	userRepo := &usageEquivalenceUserRepoStub{err: errors.New("database unavailable")}
	router := newUsageEquivalenceTestRouterWithUserRepo(usageRepo, userRepo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/equivalence?period=all_time", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.True(t, userRepo.called)
	require.False(t, usageRepo.called)
}

func TestUsageEquivalenceAllTimeRejectsInvalidTimezoneBeforeUserLookup(t *testing.T) {
	usageRepo := &usageEquivalenceRepoStub{}
	userRepo := &usageEquivalenceUserRepoStub{user: &service.User{
		ID:        42,
		CreatedAt: time.Now(),
	}}
	router := newUsageEquivalenceTestRouterWithUserRepo(usageRepo, userRepo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/equivalence?period=all_time&timezone=Not%2FA_Timezone", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.False(t, userRepo.called)
	require.False(t, usageRepo.called)
}

func TestUsageEquivalenceRejectsInvalidPeriodBeforeQuery(t *testing.T) {
	repo := &usageEquivalenceRepoStub{}
	router := newUsageEquivalenceTestRouter(repo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/equivalence?period=year", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.False(t, repo.called)
}

func TestUsageEquivalenceRejectsInvalidTimezoneBeforeQuery(t *testing.T) {
	repo := &usageEquivalenceRepoStub{}
	router := newUsageEquivalenceTestRouter(repo, 42)

	req := httptest.NewRequest(http.MethodGet, "/usage/equivalence?timezone=Not%2FA_Timezone", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.False(t, repo.called)
}
