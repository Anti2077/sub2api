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

	openAIPlanPricingAsOf      = "2026-09-04"
	openAIPlanPricingSource    = "https://learn.chatgpt.com/docs/pricing"
	usageEquivalenceBasis      = "recorded_standard_cost"
	usageEquivalenceScope      = "all_models"
	usageEquivalenceCurrency   = "USD"
	usageEquivalenceDisclaimer = "api_price_equivalent_not_quota_measurement"
)

type usageEquivalencePlanDefinition struct {
	ID            string
	Name          string
	MonthlyPrice  float64
	UsageMultiple int
}

var usageEquivalencePlanDefinitions = [...]usageEquivalencePlanDefinition{
	{ID: "chatgpt_plus", Name: "ChatGPT Plus", MonthlyPrice: 20, UsageMultiple: 1},
	{ID: "chatgpt_pro_5x", Name: "ChatGPT Pro 5x", MonthlyPrice: 100, UsageMultiple: 5},
	{ID: "chatgpt_pro_20x", Name: "ChatGPT Pro 20x", MonthlyPrice: 200, UsageMultiple: 20},
}

type usageEquivalencePlan struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	MonthlyPrice     float64 `json:"monthly_price"`
	UsageMultiple    int     `json:"usage_multiple"`
	EquivalentMonths float64 `json:"equivalent_months"`
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
	PricingBasis            string                 `json:"pricing_basis"`
	PricingAsOf             string                 `json:"pricing_as_of"`
	PricingSource           string                 `json:"pricing_source"`
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

func usageEquivalenceRange(rawPeriod, rawTimezone string, now time.Time) (string, string, time.Time, time.Time, bool) {
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

func buildUsageEquivalenceResponse(
	period string,
	timezoneName string,
	startTime time.Time,
	endTime time.Time,
	stats *usagestats.UsageStats,
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
		plans = append(plans, usageEquivalencePlan{
			ID:               definition.ID,
			Name:             definition.Name,
			MonthlyPrice:     definition.MonthlyPrice,
			UsageMultiple:    definition.UsageMultiple,
			EquivalentMonths: standardCost / definition.MonthlyPrice,
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
		PricingBasis:            usageEquivalenceBasis,
		PricingAsOf:             openAIPlanPricingAsOf,
		PricingSource:           openAIPlanPricingSource,
		Disclaimer:              usageEquivalenceDisclaimer,
	}
}

// UsageEquivalence returns the current user's recorded standard cost expressed
// as OpenAI ChatGPT/Codex plan monthly-price equivalents. OpenAI describes the
// Pro tiers as 5x and 20x Plus usage, but this endpoint compares API list-price
// value rather than claiming a fixed number of subscription messages.
func (h *UsageHandler) UsageEquivalence(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	period, timezoneName, startTime, endTime, ok := usageEquivalenceRange(
		c.DefaultQuery("period", usageEquivalencePeriodThisMonth),
		c.Query("timezone"),
		time.Now(),
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

	response.Success(c, buildUsageEquivalenceResponse(period, timezoneName, startTime, endTime, stats))
}
