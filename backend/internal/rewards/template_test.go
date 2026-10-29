package rewards

import (
	"strings"
	"testing"
)

func TestTemplateModelsValidationAndCompatibility(t *testing.T) {
	base := Template{Name: "multi", Kind: "model_day", Model: "legacy", Models: []string{"gpt-a", "gpt-b"}, GroupIDs: []int64{1}, Hours: 168, ClaimDays: 30}
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	if !base.matchesModel("gpt-b") || base.matchesModel("legacy") || base.matchesModel("gpt") {
		t.Fatal("model list must match exactly and supersede legacy model")
	}
	for _, ids := range [][]string{{""}, {"gpt-*"}, {"gpt?"}, {"gpt a"}, {"gpt\tb"}, {"gpt\nb"}, {strings.Repeat("x", 201)}, make([]string, 101)} {
		v := base
		v.Models = ids
		if v.Validate() == nil {
			t.Fatalf("accepted invalid IDs: %q", ids)
		}
	}
	base.Models = nil
	if !base.matchesModel("legacy") {
		t.Fatal("legacy snapshots must still work")
	}
	for _, hours := range []int{24, 48, 168, 8760} {
		base.Hours = hours
		if err := base.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, hours := range []int{0, 1, 25, 8761} {
		base.Hours = hours
		if base.Validate() == nil {
			t.Fatalf("accepted duration %d", hours)
		}
	}
	base.Kind = "group_rate"
	base.Hours = 72
	base.Rate = .5
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	base.Kind = "model_tokens"
	base.Hours = 5
	base.Tokens = 100
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTemplateOverlapChecksEveryModelAndGroup(t *testing.T) {
	multi := Template{Kind: "model_day", Models: []string{"gpt-a", "gpt-b"}, GroupIDs: []int64{1, 2}}
	for _, tc := range []struct {
		other Template
		want  bool
	}{
		{Template{Kind: "model_day", Model: "gpt-b", GroupIDs: []int64{2}}, true},
		{Template{Kind: "model_day", Models: []string{"gpt-c", "gpt-a"}, GroupIDs: []int64{1}}, true},
		{Template{Kind: "model_day", Model: "gpt-c", GroupIDs: []int64{1}}, false},
		{Template{Kind: "model_day", Model: "gpt-a", GroupIDs: []int64{3}}, false},
		{Template{Kind: "model_tokens", Model: "gpt-a", GroupIDs: []int64{1}}, false},
	} {
		if multi.overlaps(tc.other) != tc.want || tc.other.overlaps(multi) != tc.want {
			t.Fatalf("overlap mismatch: %+v", tc)
		}
	}
}
