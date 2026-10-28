package service

import (
	"math"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestPaginateAPIKeysLargePage(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, 1, 2, 3, 20, 21, 1000, 1001} {
		params := pagination.PaginationParams{Page: math.MaxInt, PageSize: size}
		require.Empty(t, paginateAPIKeys([]APIKey{{ID: 1}}, params), "page_size=%d", size)
	}
}

func TestMergeBalanceHistoryCodesLargePage(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, 1, 2, 3, 20, 21, 1000, 1001} {
		params := pagination.PaginationParams{Page: math.MaxInt, PageSize: size}
		require.Empty(t, mergeBalanceHistoryCodes(params,
			[]RedeemCode{{ID: 1, Type: RedeemTypeBalance}},
			[]RedeemCode{{ID: 2, Type: RedeemTypeAffiliateBalance}},
			[]RedeemCode{{ID: 3, Type: RedeemTypePromoBalance}},
			[]RedeemCode{{ID: 4, Type: RedeemTypeLotteryBalance}},
		), "page_size=%d", size)
	}
}
