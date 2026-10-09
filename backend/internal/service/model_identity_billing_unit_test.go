//go:build unit

package service

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIdentityOpenAIProbesUsePinnedAccountModelMappingAndNormalBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(7)
	target := &Account{ID: 42, Name: "target", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 4, GroupIDs: []int64{groupID}, Credentials: map[string]any{"api_key": "simulation-upstream-key", "base_url": "https://simulation.example", "api_protocol": "responses", "model_mapping": map[string]any{"public-test-model": "gpt-5.1"}}}
	other := &Account{ID: 43, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, GroupIDs: []int64{groupID}}
	accountRepo := &mockAccountRepoForPlatform{accounts: []Account{*target, *other}, accountsByID: map[int64]*Account{42: target, 43: other}}
	logs := &openAIRecordUsageLogRepoStub{inserted: true}
	billing := &openAIRecordUsageBillingRepoStub{}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logs, billing, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	svc.accountRepo = accountRepo
	ctx := WithIdentityTarget(context.Background(), target.ID, groupID)
	key := &APIKey{ID: 9, Name: "dedicated test", UserID: 5, GroupID: &groupID, Group: &Group{ID: groupID, Platform: PlatformOpenAI, Status: StatusActive, RateMultiplier: 1}}
	for i := 0; i < 3; i++ {
		selected, _, e := svc.SelectAccountWithSchedulerForCapability(ctx, &groupID, "", "", "public-test-model", nil, OpenAIUpstreamTransportHTTPSSE, OpenAIEndpointCapabilityChatCompletions, false, false, true)
		require.NoError(t, e)
		require.EqualValues(t, 42, selected.Account.ID)
		upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}, "X-Request-Id": []string{"simulation-request"}}, Body: io.NopCloser(strings.NewReader(`data: {"type":"response.completed","response":{"id":"resp_simulated","object":"response","model":"gpt-5.1","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"simulated identity probe"}]}],"usage":{"input_tokens":100,"output_tokens":20}}}` + "\n\ndata: [DONE]\n\n"))}}
		svc.httpUpstream = upstream
		body := []byte(`{"model":"public-test-model","stream":false,"messages":[{"role":"user","content":"hello"}]}`)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body)).WithContext(ctx)
		result, e := svc.ForwardAsChatCompletions(ctx, c, selected.Account, body, "", "")
		selected.ReleaseFunc()
		require.NoError(t, e)
		require.Equal(t, "gpt-5.1", result.UpstreamModel)
		require.NotNil(t, upstream.lastReq)
		require.Equal(t, "Bearer simulation-upstream-key", upstream.lastReq.Header.Get("Authorization"))
		require.NoError(t, svc.RecordUsage(ctx, &OpenAIRecordUsageInput{Result: result, Account: selected.Account, APIKey: key, User: &User{ID: 5}, ChannelUsageFields: ChannelUsageFields{OriginalModel: "public-test-model"}}))
		require.EqualValues(t, 42, logs.lastLog.AccountID)
		require.EqualValues(t, 9, logs.lastLog.APIKeyID)
		require.EqualValues(t, 5, logs.lastLog.UserID)
		require.EqualValues(t, 7, *logs.lastLog.GroupID)
		require.NotNil(t, billing.lastCmd)
		require.EqualValues(t, 9, billing.lastCmd.APIKeyID)
	}
	target.Status = StatusError
	selected, _, e := svc.SelectAccountWithSchedulerForCapability(ctx, &groupID, "", "", "public-test-model", nil, OpenAIUpstreamTransportHTTPSSE, OpenAIEndpointCapabilityChatCompletions, false, false, true)
	require.Error(t, e)
	require.Nil(t, selected)
	target.Status = StatusActive
	selected, _, e = svc.SelectAccountWithSchedulerForCapability(ctx, &groupID, "", "", "public-test-model", map[int64]struct{}{42: {}}, OpenAIUpstreamTransportHTTPSSE, OpenAIEndpointCapabilityChatCompletions, false, false, true)
	require.Error(t, e)
	require.Nil(t, selected)
}
