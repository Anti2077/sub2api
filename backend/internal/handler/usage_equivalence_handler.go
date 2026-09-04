package handler

import (
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	appTimezone "github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const (
	usageEquivalencePeriodLast24Hours = "last_24h"
	usageEquivalencePeriodLast7Days   = "last_7d"
	usageEquivalencePeriodThisMonth   = "this_month"
	usageEquivalencePeriodLast30Days  = "last_30d"
	usageEquivalencePeriodLast6Months = "last_6m"
	usageEquivalencePeriodAllTime     = "all_time"

	openAIPlanReferenceSource  = "https://learn.chatgpt.com/docs/pricing"
	usageEquivalenceBasis      = "configured_plus_quota_standard_cost"
	usageEquivalenceScope      = "all_models"
	usageEquivalenceCurrency   = "USD"
	usageEquivalenceDisclaimer = "configured_quota_reference_not_official_fixed_limit"
)

type usageEquivalencePlanDefinition struct {
	ID            string
	Name          string
	UsageMultiple int
}

var usageEquivalencePlanDefinitions = [...]usageEquivalencePlanDefinition{
	{ID: "chatgpt_plus", Name: "ChatGPT Plus", UsageMultiple: 1},
	{ID: "chatgpt_pro_5x", Name: "ChatGPT Pro 5x", UsageMultiple: 5},
	{ID: "chatgpt_pro_20x", Name: "ChatGPT Pro 20x", UsageMultiple: 20},
}

type usageEquivalencePlan struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	UsageMultiple        int     `json:"usage_multiple"`
	Quota7DStandardCost  float64 `json:"quota_7d_standard_cost"`
	Quota30DStandardCost float64 `json:"quota_30d_standard_cost"`
	Equivalent7DWindows  float64 `json:"equivalent_7d_windows"`
	Equivalent30DWindows float64 `json:"equivalent_30d_windows"`
}

type usageEquivalenceResponse struct {
	Period                  string                 `json:"period"`
	StartTime               time.Time              `json:"start_time"`
	EndTime                 time.Time              `json:"end_time"`
	Timezone                string                 `json:"timezone"`
	Scope                   string                 `json:"scope"`
	Currency                string                 `json:"currency"`
	StandardCost            float64                `json:"standard_cost"`
	ActualCost              float64                `json:"actual_cost"`
	EffectiveRateMultiplier float64                `json:"effective_rate_multiplier"`
	TotalRequests           int64                  `json:"total_requests"`
	TotalTokens             int64                  `json:"total_tokens"`
	Plans                   []usageEquivalencePlan `json:"plans"`
	ReferenceBasis          string                 `json:"reference_basis"`
	ReferenceSource         string                 `json:"reference_source"`
	Disclaimer              string                 `json:"disclaimer"`
}

func resolveUsageEquivalenceLocation(raw string) (*time.Location, string, bool) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return appTimezone.Location(), appTimezone.Name(), true
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, "", false
	}
	return location, name, true
}

func usageEquivalenceRange(rawPeriod, rawTimezone string, now time.Time, registeredAt *time.Time) (string, string, time.Time, time.Time, bool) {
	location, timezoneName, ok := resolveUsageEquivalenceLocation(rawTimezone)
	if !ok {
		return "", "", time.Time{}, time.Time{}, false
	}

	period := strings.ToLower(strings.TrimSpace(rawPeriod))
	if period == "" {
		period = usageEquivalencePeriodThisMonth
	}
	endTime := now.In(location)

	switch period {
	case usageEquivalencePeriodLast24Hours:
		return period, timezoneName, endTime.Add(-24 * time.Hour), endTime, true
	case usageEquivalencePeriodLast7Days:
		return period, timezoneName, endTime.Add(-7 * 24 * time.Hour), endTime, true
	case usageEquivalencePeriodThisMonth:
		startTime := time.Date(endTime.Year(), endTime.Month(), 1, 0, 0, 0, 0, location)
		return period, timezoneName, startTime, endTime, true
	case usageEquivalencePeriodLast30Days:
		return period, timezoneName, endTime.Add(-30 * 24 * time.Hour), endTime, true
	case usageEquivalencePeriodLast6Months:
		return period, timezoneName, endTime.AddDate(0, -6, 0), endTime, true
	case usageEquivalencePeriodAllTime:
		if registeredAt == nil || registeredAt.IsZero() {
			return "", "", time.Time{}, time.Time{}, false
		}
		startTime := registeredAt.In(location)
		if startTime.After(endTime) {
			startTime = endTime
		}
		return period, timezoneName, startTime, endTime, true
	default:
		return "", "", time.Time{}, time.Time{}, false
	}
}

func nonNegativeCost(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func usageQuotaEquivalent(standardCost, quotaStandardCost float64) float64 {
	if standardCost <= 0 || quotaStandardCost <= 0 {
		return 0
	}
	return standardCost / quotaStandardCost
}

func buildUsageEquivalenceResponse(
	period string,
	timezoneName string,
	startTime time.Time,
	endTime time.Time,
	stats *usagestats.UsageStats,
	plus7DLimitUSD float64,
	plus30DLimitUSD float64,
) usageEquivalenceResponse {
	standardCost := 0.0
	actualCost := 0.0
	var totalRequests int64
	var totalTokens int64
	if stats != nil {
		standardCost = nonNegativeCost(stats.TotalCost)
		actualCost = nonNegativeCost(stats.TotalActualCost)
		totalRequests = stats.TotalRequests
		totalTokens = stats.TotalTokens
	}

	effectiveRateMultiplier := 0.0
	if standardCost > 0 {
		effectiveRateMultiplier = actualCost / standardCost
	}

	plans := make([]usageEquivalencePlan, 0, len(usageEquivalencePlanDefinitions))
	for _, definition := range usageEquivalencePlanDefinitions {
		quota7DStandardCost := plus7DLimitUSD * float64(definition.UsageMultiple)
		quota30DStandardCost := plus30DLimitUSD * float64(definition.UsageMultiple)
		plans = append(plans, usageEquivalencePlan{
			ID:                   definition.ID,
			Name:                 definition.Name,
			UsageMultiple:        definition.UsageMultiple,
			Quota7DStandardCost:  quota7DStandardCost,
			Quota30DStandardCost: quota30DStandardCost,
			Equivalent7DWindows:  usageQuotaEquivalent(standardCost, quota7DStandardCost),
			Equivalent30DWindows: usageQuotaEquivalent(standardCost, quota30DStandardCost),
		})
	}

	return usageEquivalenceResponse{
		Period:                  period,
		StartTime:               startTime,
		EndTime:                 endTime,
		Timezone:                timezoneName,
		Scope:                   usageEquivalenceScope,
		Currency:                usageEquivalenceCurrency,
		StandardCost:            standardCost,
		ActualCost:              actualCost,
		EffectiveRateMultiplier: effectiveRateMultiplier,
		TotalRequests:           totalRequests,
		TotalTokens:             totalTokens,
		Plans:                   plans,
		ReferenceBasis:          usageEquivalenceBasis,
		ReferenceSource:         openAIPlanReferenceSource,
		Disclaimer:              usageEquivalenceDisclaimer,
	}
}

// UsageEquivalence compares recorded standard API cost with operator-configured
// Plus 7d/30d quota references. The Pro references apply OpenAI's published 5x
// and 20x plan multipliers; the configured Plus values are estimates, not fixed
// official quota promises.
func (h *UsageHandler) UsageEquivalence(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.settingService == nil {
		response.InternalError(c, "Usage equivalence settings are unavailable")
		return
	}
	settings, err := h.settingService.GetUsageEquivalenceSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !settings.Enabled {
		response.NotFound(c, "Usage equivalence is not enabled")
		return
	}
	if settings.Plus7DLimitUSD <= 0 || settings.Plus30DLimitUSD <= 0 {
		response.InternalError(c, "Usage equivalence quota references are not configured")
		return
	}

	rawPeriod := c.DefaultQuery("period", usageEquivalencePeriodThisMonth)
	rawTimezone := c.Query("timezone")
	var registeredAt *time.Time
	if strings.EqualFold(strings.TrimSpace(rawPeriod), usageEquivalencePeriodAllTime) {
		if _, _, timezoneOK := resolveUsageEquivalenceLocation(rawTimezone); !timezoneOK {
			response.BadRequest(c, "Invalid period or timezone")
			return
		}
		createdAt, err := h.usageService.GetUserCreatedAt(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		registeredAt = &createdAt
	}

	period, timezoneName, startTime, endTime, ok := usageEquivalenceRange(
		rawPeriod,
		rawTimezone,
		time.Now(),
		registeredAt,
	)
	if !ok {
		response.BadRequest(c, "Invalid period or timezone")
		return
	}

	filters := usagestats.UsageLogFilters{
		UserID:    subject.UserID,
		StartTime: &startTime,
		EndTime:   &endTime,
	}
	stats, err := h.usageService.GetStatsWithFilters(c.Request.Context(), filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, buildUsageEquivalenceResponse(
		period,
		timezoneName,
		startTime,
		endTime,
		stats,
		settings.Plus7DLimitUSD,
		settings.Plus30DLimitUSD,
	))
}
