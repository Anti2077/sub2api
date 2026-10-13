package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func localizableGroup() *Group {
	return &Group{
		ID: 1, Name: "标准", Description: "号池",
		I18n: domain.GroupI18n{"en": {Name: "Standard", Description: "Pro + Max pool"}},
	}
}

func TestGroupLocalized_ReplacesTextAndDropsTranslations(t *testing.T) {
	g := localizableGroup().Localized("en")

	require.Equal(t, "Standard", g.Name)
	require.Equal(t, "Pro + Max pool", g.Description)

	raw, err := json.Marshal(g)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.NotContains(t, decoded, "i18n", "user-facing output must not carry the translation table")
}

func TestGroupLocalized_KeepsOriginalWhenLanguageHasNoTranslation(t *testing.T) {
	g := localizableGroup().Localized("zh")

	require.Equal(t, "标准", g.Name)
	require.Equal(t, "号池", g.Description)
	require.Nil(t, g.I18n)
}

func TestGroupLocalized_PreservesCustomPricingAndAdminIsolation(t *testing.T) {
	source := &service.Group{
		ID: 9, Name: "标准", Description: "号池",
		I18n:           domain.GroupI18n{"en": {Name: "Standard", Description: "Pool"}},
		RateMultiplier: 1.2, PeakRateEnabled: true,
		PeakStart: "18:00", PeakEnd: "23:00", PeakRateMultiplier: 1.5,
		ProfitControlEnabled: true, ProfitMinMargin: 0.2, ProfitSafetyBuffer: 0.1,
		DuplicateOperationID: "internal-operation",
	}
	admin := GroupFromServiceAdmin(source)
	user := GroupFromService(source).Localized("en")

	require.Equal(t, "Standard", user.Name)
	require.Equal(t, "Pool", user.Description)
	require.Equal(t, 1.2, user.RateMultiplier)
	require.True(t, user.PeakRateEnabled)
	require.Equal(t, "18:00", user.PeakStart)
	require.Equal(t, "23:00", user.PeakEnd)
	require.Equal(t, 1.5, user.PeakRateMultiplier)

	raw, err := json.Marshal(user)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	for _, private := range []string{"i18n", "profit_control_enabled", "profit_min_margin", "profit_safety_buffer", "duplicate_operation_id"} {
		require.NotContains(t, decoded, private)
	}
	// Localizing a user DTO must preserve canonical admin text and its translation table.
	require.Equal(t, "标准", admin.Name)
	require.Equal(t, source.I18n, admin.I18n)
	require.True(t, admin.ProfitControlEnabled)
	require.Equal(t, 0.2, admin.ProfitMinMargin)
	require.Equal(t, "标准", source.Name)
	require.Equal(t, "Standard", source.I18n["en"].Name)
}

func TestLocalized_NilSafe(t *testing.T) {
	require.Nil(t, (*Group)(nil).Localized("en"))
	require.Nil(t, (*APIKey)(nil).Localized("en"))
	require.Nil(t, (*RedeemCode)(nil).Localized("en"))
	require.Nil(t, (*UsageLog)(nil).Localized("en"))
	require.Nil(t, (*UserSubscription)(nil).Localized("en"))

	// 未携带分组的记录原样返回
	require.Nil(t, (&APIKey{}).Localized("en").Group)
}

func TestLocalized_NestedGroups(t *testing.T) {
	key := (&APIKey{Group: localizableGroup()}).Localized("en")
	require.Equal(t, "Standard", key.Group.Name)

	code := (&RedeemCode{Group: localizableGroup()}).Localized("en")
	require.Equal(t, "Standard", code.Group.Name)

	sub := (&UserSubscription{Group: localizableGroup()}).Localized("en")
	require.Equal(t, "Standard", sub.Group.Name)

	// 使用记录除自身分组外，还带着 API Key 与订阅各自的分组
	log := (&UsageLog{
		Group:        localizableGroup(),
		APIKey:       &APIKey{Group: localizableGroup()},
		Subscription: &UserSubscription{Group: localizableGroup()},
	}).Localized("en")
	require.Equal(t, "Standard", log.Group.Name)
	require.Equal(t, "Standard", log.APIKey.Group.Name)
	require.Equal(t, "Standard", log.Subscription.Group.Name)
	require.Nil(t, log.APIKey.Group.I18n)
	require.Nil(t, log.Subscription.Group.I18n)
}
