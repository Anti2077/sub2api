package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestRedeemReceiptIsReturnedWithoutInternalNotes(t *testing.T) {
	for _, kind := range []string{"model_day", "model_tokens", "group_rate"} {
		t.Run(kind, func(t *testing.T) {
			card := &rewards.Card{ID: 42, UserID: 7, Status: "unused", Template: rewards.Template{Name: "Issued snapshot", Version: 3, Kind: kind, Model: "luna", Tokens: 12345, Rate: .25, Hours: 24, GroupIDs: []int64{2}}}
			response := RedeemCodeFromService(&service.RedeemCode{Type: "card", Status: "used", Card: card, Notes: "internal note"})
			raw, err := json.Marshal(response)
			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(raw, &decoded))
			require.NotContains(t, decoded, "notes")
			receipt, ok := decoded["card"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, float64(42), receipt["id"])
			template, ok := receipt["template"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, kind, template["kind"])
			require.Equal(t, float64(12345), template["tokens"])
			require.Equal(t, .25, template["rate"])
		})
	}
}
func TestLegacyRedeemResponseOmitsCardReceipt(t *testing.T) {
	for _, kind := range []string{"balance", "concurrency", "subscription"} {
		raw, err := json.Marshal(RedeemCodeFromService(&service.RedeemCode{Type: kind, Value: 10}))
		require.NoError(t, err)
		var decoded map[string]any
		require.NoError(t, json.Unmarshal(raw, &decoded))
		require.NotContains(t, decoded, "card")
		require.Equal(t, float64(10), decoded["value"])
	}
}
