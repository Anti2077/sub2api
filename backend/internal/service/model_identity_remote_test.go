package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRemoteRunStatus(t *testing.T) {
	for _, test := range []struct {
		status string
		want   string
	}{
		{"completed", "completed"},
		{"aborted", "cancelled"},
		{"cancelled", "cancelled"},
		{"failed", "request_error"},
		{"running", "service_error"},
	} {
		t.Run(test.status, func(t *testing.T) {
			require.Equal(t, test.want, remoteRunStatus(map[string]any{"status": test.status}))
		})
	}
}

func TestIdentityHTTPClientsUseTLS12ForHostedProbe(t *testing.T) {
	for _, client := range []*http.Client{identityRemoteCatalogClient(), identityRunHTTPClient()} {
		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.TLSClientConfig)
		require.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
		require.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MaxVersion)
	}
}

func TestRemoteVerdictAndDetectedModel(t *testing.T) {
	assessment := map[string]any{
		"resolvedIdentity": map[string]any{"modelId": "openai/gpt-6-astra"},
	}
	verdict := map[string]any{"status": "clean_match", "trueModel": "openai/gpt-6-astra", "trueFamily": "openai"}
	require.Equal(t, "matched", remoteVerdict(verdict["status"]))
	require.Equal(t, "openai/gpt-6-astra", remoteDetectedModel(assessment, verdict))

	verdict = map[string]any{"status": "clean_match_submodel_mismatch", "trueFamily": "openai"}
	require.Equal(t, "mismatched", remoteVerdict(verdict["status"]))
	require.Equal(t, "openai/gpt-6-astra", remoteDetectedModel(assessment, verdict))
	// The website can expose only the human label in trueModel. The canonical
	// resolvedIdentity.modelId must win when both fields are present.
	verdict = map[string]any{"status": "clean_match", "trueModel": "GPT 6 Astra"}
	require.Equal(t, "openai/gpt-6-astra", remoteDetectedModel(assessment, verdict))

	verdict = map[string]any{"status": "insufficient_data"}
	require.Equal(t, "inconclusive", remoteVerdict(verdict["status"]))
	require.Empty(t, remoteDetectedModel(map[string]any{}, verdict))
}

func TestRedactRemoteSecrets(t *testing.T) {
	payload := map[string]any{
		"userKey": "secret-key",
		"items":   []any{map[string]any{"Authorization": "Bearer secret", "usage": map[string]any{"totalTokens": 3}}},
	}
	redactRemoteSecrets(payload)
	require.Equal(t, "[redacted]", payload["userKey"])
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "[redacted]", item["Authorization"])
	usage, ok := item["usage"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, 3, usage["totalTokens"])
}

func TestRemoteIdentityRequiresConfirmedSpecificModel(t *testing.T) {
	for _, tc := range []struct{ verdict, detected, want string }{
		{"clean_match", "openai/gpt-6-astra", "matched"},
		{"clean_match", "GPT 6 Astra", "matched"},
		{"clean_match", "openai/gpt-5.5", "inconclusive"},
		{"clean_match_family_only", "", "inconclusive"},
		{"clean_match_submodel_mismatch", "openai/gpt-5.5", "mismatched"},
		{"clean_match_submodel_mismatch", "", "inconclusive"},
		{"ambiguous", "openai/gpt-6-astra", "inconclusive"},
		{"insufficient_data", "openai/gpt-6-astra", "inconclusive"},
		{"plain_mismatch", "", "inconclusive"},
	} {
		t.Run(tc.verdict+tc.detected, func(t *testing.T) {
			require.Equal(t, tc.want, remoteIdentityVerdict(tc.verdict, tc.detected, "openai/gpt-6-astra"))
		})
	}
	require.Equal(t, "matched", remoteIdentityVerdict("match", "GPT 6 Astra", "openai/gpt-6-astra"))
	require.Equal(t, "mismatched", remoteIdentityVerdict("mismatch", "GPT 5.5", "openai/gpt-6-astra"))
	// When a provider omits the nested verdict status, the outer assessment
	// status is still an exact family-level signal and must be interpreted.
	require.Equal(t, "matched", remoteIdentityVerdict("match", "openai/gpt-6-astra", "openai/gpt-6-astra"))
	require.Equal(t, "inconclusive", remoteIdentityVerdict("match", "anthropic/gpt-6-astra", "openai/gpt-6-astra"))
	require.Empty(t, remoteDetectedModel(map[string]any{}, map[string]any{"trueFamily": "openai"}))
}

func TestRemoteBaselineCatalogAndVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.URL.RawQuery)
		switch r.URL.Path {
		case "/api/probe/baselines":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []string{"openai/gpt-6-astra", "anthropic/claude-opus-5"}})
		case "/api/probe/suggested-models":
			_, _ = w.Write([]byte(`{"models":[{"modelId":"openai/gpt-6-astra","identification":{"v3h":true}},{"modelId":"openai/gpt-6.1-sol","identification":{"v3h":true,"v3":false}},{"modelId":"openai/gpt-6-luna","identification":{"v3":true}},{"modelId":"openai/unsupported","identification":{"v3":false,"v3h":false}}]}`))
		default:
			t.Errorf("unexpected catalog path: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	svc := &ModelIdentityService{remoteURL: server.URL + "/api/probe/run?unused=1", client: server.Client()}
	models, err := svc.Models(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 4)
	require.Equal(t, "openai/gpt-6-astra", models[1].ID)
	require.Equal(t, "openai", models[1].Family)
	require.Equal(t, "openai/gpt-6-luna", models[2].ID)
	require.Equal(t, "openai/gpt-6.1-sol", models[3].ID)
	require.Equal(t, "bazaarlink-online", svc.EngineCommit())
}

func TestRemoteHeaderSecretsAreRedacted(t *testing.T) {
	payload := map[string]any{"responseHeaders": map[string]any{"x-api-key": "upstream-secret", "Set-Cookie": "session=private", "x-request-id": "request-id"}, "startedByIp": "private-ip", "usage": map[string]any{"promptTokens": 7}}
	redactRemoteSecrets(payload)
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "upstream-secret")
	require.NotContains(t, string(encoded), "session=private")
	require.NotContains(t, string(encoded), "private-ip")
	require.Contains(t, string(encoded), "request-id")
	require.Contains(t, string(encoded), `"promptTokens":7`)
}

type identityPauseRepo struct {
	IdentityRepository
	plan  IdentityPlan
	saved bool
}

func (r *identityPauseRepo) Plan(context.Context, int64) (*IdentityPlan, error) { return &r.plan, nil }
func (r *identityPauseRepo) SavePlan(_ context.Context, p *IdentityPlan, _ time.Time) error {
	r.plan = *p
	r.saved = true
	return nil
}

func TestIdentityPauseWithoutWorkingKeyOrCatalog(t *testing.T) {
	repo := &identityPauseRepo{plan: IdentityPlan{ID: 1, AccountID: 42, RequestModel: "gpt-6", ExpectedModel: "openai/gpt-6-astra", IntervalMinutes: 120, Enabled: true}}
	svc := &ModelIdentityService{repo: repo}
	plan := repo.plan
	plan.Enabled = false
	plan.AccountID = 43 // The saved owner always wins.
	require.NoError(t, svc.SavePlan(context.Background(), &plan))
	require.True(t, repo.saved)
	require.False(t, repo.plan.Enabled)
	require.EqualValues(t, 42, repo.plan.AccountID)
	plan.Enabled = true
	require.Error(t, svc.SavePlan(context.Background(), &plan))
}

func TestIdentityRunDoesNotForwardCredentialThroughRedirect(t *testing.T) {
	var forwarded atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { forwarded.Store(true) }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	res, err := identityRunHTTPClient().Post(origin.URL, "application/json", strings.NewReader(`{"apiKey":"test-only"}`))
	require.NoError(t, err)
	defer func() { require.NoError(t, res.Body.Close()) }()
	require.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
	require.False(t, forwarded.Load())
}
