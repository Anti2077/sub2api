package rewards

import (
	"github.com/shopspring/decimal"
	"io"
	"math/rand"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIdentity(t *testing.T) {
	cases := []struct {
		path, body, model string
		pure              bool
	}{
		{"/v1/chat/completions", `{"model":"luna","messages":[{"role":"user","content":"hello"}]}`, "luna", true},
		{"/v1/responses", `{"model":"luna","input":[{"role":"user","content":[{"type":"input_image","image_url":"x"}]}]}`, "luna", false},
		{"/v1/responses", `{"model":"luna","input":"hi","tools":[{"type":"web_search"}]}`, "luna", false},
		{"/v1beta/models/gemini-test:generateContent", `{"contents":[{"parts":[{"text":"hi"}]}]}`, "gemini-test", true},
		{"/v1beta/models/gemini-test:generateContent", `{"contents":[{"parts":[{"inlineData":{"data":"image"}}]}]}`, "gemini-test", false},
		{"/v1/images/generations", `{"model":"luna","prompt":"hi"}`, "", false},
	}
	for _, c := range cases {
		t.Run(c.path+c.body, func(t *testing.T) {
			r := httptest.NewRequest("POST", c.path, strings.NewReader(c.body))
			model, pure, e := RequestIdentity(r)
			if e != nil || model != c.model || pure != c.pure {
				t.Fatalf("got %q %v %v", model, pure, e)
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != c.body {
				t.Fatal("request body consumed")
			}
		})
	}
}
func TestAdjustTextIndependentOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(29))
	for i := 0; i < 2000; i++ {
		original := float64(rng.Intn(10000000)) / 1e5
		text := original * float64(rng.Intn(101)) / 100
		current := float64(rng.Intn(90)+10) / 100
		rate := float64(rng.Intn(90)+10) / 100
		c := &Capture{Rate: &rate, Free: i%7 == 0}
		got, _ := AdjustText(original, text, c, current)
		off := decimal.Zero
		if rate < current {
			off = decimal.NewFromFloat(text).Mul(decimal.NewFromInt(1).Sub(decimal.NewFromFloat(rate).Div(decimal.NewFromFloat(current))))
		}
		if c.Free {
			off = decimal.NewFromFloat(text)
		}
		want, _ := decimal.NewFromFloat(original).Sub(off).Round(8).Float64()
		if got != want {
			t.Fatalf("iteration %d got %.8f want %.8f", i, got, want)
		}
	}
}
func TestCardTemplateWholeDays(t *testing.T) {
	base := Template{Name: "test", Kind: "model_day", Model: "test", GroupIDs: []int64{1}, Hours: 24, ClaimDays: 30}
	if e := base.Validate(); e != nil {
		t.Fatal(e)
	}
	base.Hours = 7 * 24
	if e := base.Validate(); e != nil {
		t.Fatal(e)
	}
	base.Hours = 1
	if base.Validate() == nil {
		t.Fatal("day duration must be a whole day")
	}
}
