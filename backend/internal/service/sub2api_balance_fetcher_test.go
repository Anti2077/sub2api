package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

type sub2APIBalanceHTTPStub struct {
	response *http.Response
	request  *http.Request
}

func (s *sub2APIBalanceHTTPStub) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	s.request = req.Clone(req.Context())
	return s.response, nil
}

func (s *sub2APIBalanceHTTPStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, concurrency)
}

func TestSub2APIBalanceFetcherFetchesPoolWalletBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()

	upstream := &sub2APIBalanceHTTPStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"mode":"unrestricted","balance":12.34,"remaining":12.34}`)),
	}}
	fetcher := NewSub2APIBalanceFetcherService(upstream, &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{AllowInsecureHTTP: true}}})
	balance, err := fetcher.FetchBalance(context.Background(), &Account{
		ID: 42, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"pool_mode": true, "api_key": "pool-key", "base_url": server.URL},
	})
	if err != nil {
		t.Fatalf("FetchBalance() error = %v", err)
	}
	if balance != 12.34 {
		t.Fatalf("balance = %v, want 12.34", balance)
	}
	if upstream.request.URL.Path != "/v1/usage" {
		t.Fatalf("request path = %q, want /v1/usage", upstream.request.URL.Path)
	}
	if got := upstream.request.Header.Get("Authorization"); got != "Bearer pool-key" {
		t.Fatalf("authorization = %q, want bearer token", got)
	}
}

func TestSub2APIBalanceFetcherSupportsRemainingFallback(t *testing.T) {
	for _, test := range []struct {
		responseBody string
		want         float64
	}{
		{responseBody: `{"mode":"unrestricted","remaining":"7.5"}`, want: 7.5},
		{responseBody: `{"mode":"quota_limited","quota":{"remaining":6.25}}`, want: 6.25},
	} {
		upstream := &sub2APIBalanceHTTPStub{response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(test.responseBody)),
		}}
		fetcher := NewSub2APIBalanceFetcherService(upstream, &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{AllowInsecureHTTP: true}}})
		balance, err := fetcher.FetchBalance(context.Background(), &Account{
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"pool_mode": true, "api_key": "pool-key", "base_url": "http://pool.example.test"},
		})
		if err != nil {
			t.Fatalf("FetchBalance(%s) error = %v", test.responseBody, err)
		}
		if balance != test.want {
			t.Fatalf("FetchBalance(%s) = %v, want %v", test.responseBody, balance, test.want)
		}
	}
}

func TestSub2APIBalanceFetcherKeepsStaticOverridesWithoutReportingUserIdentity(t *testing.T) {
	groupID := int64(303)
	for _, ctx := range []context.Context{
		context.Background(),
		WithHeaderOverrideRequestScope(context.Background(), 101, 202, &groupID),
	} {
		upstream := &sub2APIBalanceHTTPStub{response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"balance":7.5}`)),
		}}
		fetcher := NewSub2APIBalanceFetcherService(upstream, &config.Config{
			Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{AllowInsecureHTTP: true}},
		})
		balance, err := fetcher.FetchBalance(ctx, &Account{
			ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Credentials: map[string]any{
				"pool_mode": true, "api_key": "pool-key", "base_url": "http://pool.example.test",
				credKeyHeaderOverrideEnabled: true,
				credKeyHeaderOverrides: map[string]string{
					"x-pool-route": "static-route",
					"x-pool-user":  "{{user_id}}",
				},
			},
		})
		if err != nil || balance != 7.5 {
			t.Fatalf("FetchBalance() = %v, %v; want 7.5, nil", balance, err)
		}
		headers := make(http.Header)
		for name, values := range upstream.request.Header {
			for _, value := range values {
				headers.Add(name, value)
			}
		}
		if got := headers.Values("X-Pool-Route"); len(got) != 1 || got[0] != "static-route" {
			t.Fatalf("static override = %q, want [static-route]", got)
		}
		if got := headers.Values("X-Pool-User"); len(got) != 0 {
			t.Fatalf("shared balance probe reported a user identity")
		}
		if headerOverrideScopeFromContext(upstream.request.Context()) != nil {
			t.Fatal("shared balance probe retained an authenticated request scope")
		}
	}
}

func TestSub2APIBalanceFetcherRejectsNonPoolAccount(t *testing.T) {
	fetcher := NewSub2APIBalanceFetcherService(&sub2APIBalanceHTTPStub{}, &config.Config{})
	if _, err := fetcher.FetchBalance(context.Background(), &Account{Type: AccountTypeAPIKey}); err == nil {
		t.Fatal("expected non-pool account to be rejected")
	}
}

type sub2APIBalanceFetcherStub struct{}

func (sub2APIBalanceFetcherStub) FetchBalance(context.Context, *Account) (float64, error) {
	return 8.75, nil
}

func TestAccountUsageServicePoolModeUsesSub2APIBalance(t *testing.T) {
	account := &Account{
		ID: 7, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"pool_mode": true},
	}
	repo := &stubOpenAIAccountRepo{accounts: []Account{*account}}
	svc := &AccountUsageService{accountRepo: repo}
	svc.SetSub2APIBalanceFetcher(sub2APIBalanceFetcherStub{})
	usage, err := svc.GetUsage(context.Background(), account.ID)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if usage.Balance == nil || *usage.Balance != 8.75 || usage.BalanceSource != "sub2api" {
		t.Fatalf("usage balance = %#v, source=%q; want 8.75/sub2api", usage.Balance, usage.BalanceSource)
	}
}
