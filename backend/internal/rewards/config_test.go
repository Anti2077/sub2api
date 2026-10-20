package rewards

import (
	"fmt"
	"testing"
)

func TestFarmPlotLimits(t *testing.T) {
	for _, plots := range []int{1, 6, 12, 16} {
		t.Run(fmt.Sprint(plots), func(t *testing.T) {
			if err := (Config{Plots: plots, StealPercent: 10, MaxStolenPercent: 40}).Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, plots := range []int{0, 17} {
		if (Config{Plots: plots, StealPercent: 10, MaxStolenPercent: 40}).Validate() == nil {
			t.Fatalf("invalid plot count accepted: %d", plots)
		}
	}
}
