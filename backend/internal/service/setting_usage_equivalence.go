package service

import (
	"context"
	"fmt"
)

type UsageEquivalenceSettings struct {
	Enabled         bool
	Plus7DLimitUSD  float64
	Plus30DLimitUSD float64
}

func (s *SettingService) GetUsageEquivalenceSettings(ctx context.Context) (*UsageEquivalenceSettings, error) {
	if s == nil || s.settingRepo == nil {
		return nil, fmt.Errorf("usage equivalence settings are unavailable")
	}

	settings, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyUsageEquivalenceEnabled,
		SettingKeyUsageEquivalencePlus7DLimitUSD,
		SettingKeyUsageEquivalencePlus30DLimitUSD,
	})
	if err != nil {
		return nil, fmt.Errorf("get usage equivalence settings: %w", err)
	}

	return &UsageEquivalenceSettings{
		Enabled:         settings[SettingKeyUsageEquivalenceEnabled] == "true",
		Plus7DLimitUSD:  parseUsageEquivalenceLimitUSD(settings[SettingKeyUsageEquivalencePlus7DLimitUSD]),
		Plus30DLimitUSD: parseUsageEquivalenceLimitUSD(settings[SettingKeyUsageEquivalencePlus30DLimitUSD]),
	}, nil
}
