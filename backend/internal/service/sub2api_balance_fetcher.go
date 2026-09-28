package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

const (
	sub2apiBalanceRequestTimeout = 15 * time.Second
	sub2apiBalanceMaxBodyBytes   = 256 * 1024
)

// Sub2APIBalanceFetcherService reads the wallet balance from an upstream
// Sub2API instance. The account API key is never returned to the caller.
type Sub2APIBalanceFetcherService struct {
	httpUpstream HTTPUpstream
	cfg          *config.Config
}

func NewSub2APIBalanceFetcherService(httpUpstream HTTPUpstream, cfg *config.Config) *Sub2APIBalanceFetcherService {
	return &Sub2APIBalanceFetcherService{httpUpstream: httpUpstream, cfg: cfg}
}

type sub2APIUsageResponse struct {
	Mode      string          `json:"mode"`
	Balance   json.RawMessage `json:"balance"`
	Remaining json.RawMessage `json:"remaining"`
	Quota     struct {
		Remaining json.RawMessage `json:"remaining"`
	} `json:"quota"`
}

func (s *Sub2APIBalanceFetcherService) FetchBalance(ctx context.Context, account *Account) (float64, error) {
	if account == nil || !account.IsPoolMode() {
		return 0, fmt.Errorf("account is not in pool mode")
	}
	if s == nil || s.httpUpstream == nil {
		return 0, fmt.Errorf("sub2api balance transport is not configured")
	}
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return 0, fmt.Errorf("pool mode account api_key is empty")
	}
	baseURL := strings.TrimSpace(account.GetCredential("base_url"))
	if baseURL == "" {
		return 0, fmt.Errorf("pool mode account base_url is empty")
	}
	normalizedBaseURL, err := validateSub2APIBalanceBaseURL(baseURL, s.cfg)
	if err != nil {
		return 0, fmt.Errorf("invalid pool mode base_url: %w", err)
	}
	endpoint := buildOpenAIEndpointURL(normalizedBaseURL, "/v1/usage")
	requestCtx, cancel := context.WithTimeout(ctx, sub2apiBalanceRequestTimeout)
	defer cancel()
	requestCtx = WithHTTPUpstreamRedirectsDisabled(requestCtx)
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("build sub2api usage request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	account.ApplyHeaderOverrides(req.Header)
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, maxInt(account.Concurrency, 1))
	if err != nil {
		return 0, fmt.Errorf("request upstream sub2api usage: %w", err)
	}
	if resp == nil || resp.Body == nil {
		return 0, fmt.Errorf("upstream sub2api usage returned an empty response")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("upstream sub2api usage returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, sub2apiBalanceMaxBodyBytes+1))
	if err != nil {
		return 0, fmt.Errorf("read upstream sub2api usage: %w", err)
	}
	if len(body) > sub2apiBalanceMaxBodyBytes {
		return 0, fmt.Errorf("upstream sub2api usage response is too large")
	}
	var payload sub2APIUsageResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("decode upstream sub2api usage: %w", err)
	}
	for _, raw := range []json.RawMessage{payload.Balance, payload.Remaining, payload.Quota.Remaining} {
		if balance, ok := parseSub2APIBalanceNumber(raw); ok {
			return balance, nil
		}
	}
	return 0, fmt.Errorf("upstream sub2api usage response does not include a numeric balance")
}

func validateSub2APIBalanceBaseURL(raw string, cfg *config.Config) (string, error) {
	if cfg == nil {
		return urlvalidator.ValidateURLFormat(raw, false)
	}
	if !cfg.Security.URLAllowlist.Enabled {
		return urlvalidator.ValidateURLFormat(raw, cfg.Security.URLAllowlist.AllowInsecureHTTP)
	}
	return urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: true,
		AllowPrivate:     cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
}

func parseSub2APIBalanceNumber(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil && !math.IsNaN(number) && !math.IsInf(number, 0) {
		return number, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0) {
			return parsed, true
		}
	}
	return 0, false
}
