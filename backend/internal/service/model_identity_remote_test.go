package service

import (
	"context"
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
	require.Equal(t, "[redacted]", payload["items"].([]any)[0].(map[string]any)["Authorization"])
	require.Equal(t, 3, payload["items"].([]any)[0].(map[string]any)["usage"].(map[string]any)["totalTokens"])
}

func TestRemoteIdentityRequiresConfirmedSpecificModel(t *testing.T) {
	for _, tc := range []struct{ verdict, detected, want string }{
		{"clean_match", "openai/gpt-6-astra", "matched"},
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
	require.Empty(t, remoteDetectedModel(map[string]any{}, map[string]any{"trueFamily": "openai"}))
}

func TestRemoteBaselineCatalogAndVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/probe/baselines", r.URL.Path)
		require.Empty(t, r.URL.RawQuery)
		_ = json.NewEncoder(w).Encode(map[string]any{"models": []string{"openai/gpt-6-astra", "anthropic/claude-opus-5"}})
	}))
	defer server.Close()
	svc := &ModelIdentityService{remoteURL: server.URL + "/api/probe/run?unused=1", client: server.Client()}
	models, err := svc.Models(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 2)
	require.Equal(t, "openai/gpt-6-astra", models[0].ID)
	require.Equal(t, "openai", models[0].Family)
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
	defer res.Body.Close()
	require.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
	require.False(t, forwarded.Load())
}
