package routes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type identityRouteRepo struct {
	service.IdentityRepository
	run      *service.IdentityRun
	token    string
	probes   []json.RawMessage
	mu       sync.Mutex
	inflight int
}

func (r *identityRouteRepo) Authorize(_ context.Context, id int64, hash string, _ time.Time) (*service.IdentityRun, error) {
	if id != r.run.ID || hash != service.HashIdentityToken(r.token) || r.run.Status != "running" {
		return nil, errors.New("unauthorized")
	}
	return r.run, nil
}
func (r *identityRouteRepo) Run(_ context.Context, id int64) (*service.IdentityRun, error) {
	if id != r.run.ID {
		return nil, errors.New("not found")
	}
	return r.run, nil
}
func (r *identityRouteRepo) ProbeSlot(_ context.Context, _ int64, acquire bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if acquire {
		if r.inflight >= service.IdentityProbeConcurrency {
			return service.ErrIdentityProbeSlotsFull
		}
		r.inflight++
	} else {
		r.inflight--
	}
	return nil
}
func (r *identityRouteRepo) AppendProbe(_ context.Context, _ int64, p json.RawMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.probes = append(r.probes, p)
	return nil
}

type identityRouteAccountRepo struct {
	service.AccountRepository
	a *service.Account
}

func (r *identityRouteAccountRepo) GetByID(context.Context, int64) (*service.Account, error) {
	return r.a, nil
}

type identityRouteKeyRepo struct {
	service.APIKeyRepository
	k *service.APIKey
}

func (r *identityRouteKeyRepo) GetByID(context.Context, int64) (*service.APIKey, error) {
	// Real repository reads return a fresh Key; derived fields are request-local.
	key := *r.k
	return &key, nil
}

type identityRouteUserRepo struct{ service.UserRepository }

func (r *identityRouteUserRepo) GetByID(context.Context, int64) (*service.User, error) {
	return &service.User{ID: 5, Status: service.StatusActive}, nil
}

type identityRouteGroupRepo struct {
	service.GroupRepository
	exclusive bool
}

func (r *identityRouteGroupRepo) GetByID(context.Context, int64) (*service.Group, error) {
	return &service.Group{ID: 7, Platform: service.PlatformOpenAI, Status: service.StatusActive, IsExclusive: r.exclusive}, nil
}

func TestIdentityProbeCapabilityInjectsBillingKeyAndIgnoresClientTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g := int64(7)
	repo := &identityRouteRepo{run: &service.IdentityRun{ID: 1, AccountID: 42, UserID: 5, GroupID: 7, APIKeyID: 9, Status: "running", RequestModel: "mapped-model", ExpectedModel: "openai/gpt-5.5"}, token: strings.Repeat("a", 64)}
	keys := &identityRouteKeyRepo{k: &service.APIKey{ID: 9, UserID: 5, GroupID: &g, Status: service.StatusActive, Key: "private-site-key"}}
	accounts := &identityRouteAccountRepo{a: &service.Account{ID: 42, Platform: service.PlatformOpenAI, GroupIDs: []int64{7}}}
	groups := &identityRouteGroupRepo{}
	apiKeySvc := service.NewAPIKeyService(keys, &identityRouteUserRepo{}, groups, nil, nil, nil, &config.Config{})
	svc := service.NewModelIdentityService(repo, apiKeySvc, accounts, nil)
	forwarded := 0
	auth := middleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		require.Equal(t, "Bearer private-site-key", c.GetHeader("Authorization"))
		target, group, pinned := service.IdentityTargetFromContext(c.Request.Context())
		require.True(t, pinned)
		require.EqualValues(t, 42, target)
		require.EqualValues(t, 7, group)
		require.Empty(t, c.GetHeader("X-Account-ID"))
		body, e := io.ReadAll(c.Request.Body)
		require.NoError(t, e)
		require.Contains(t, string(body), `"model":"mapped-model"`)
		forwarded++
		service.RecordIdentityEvidence(c.Request.Context(), target, "upstream-model", "upstream-request")
		c.AbortWithStatusJSON(200, gin.H{"model": "auxiliary", "choices": []gin.H{{"message": gin.H{"content": "simulated response"}}}, "usage": gin.H{"prompt_tokens": 10, "completion_tokens": 20}})
	})
	router := gin.New()
	registerIdentityProbe(router, &handler.Handlers{}, svc, auth, func(*gin.Context) {}, func(*gin.Context) {})
	request := func(path, token, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Account-ID", "99")
		router.ServeHTTP(w, r)
		return w
	}
	good := request("/internal/model-identity/1/probe", repo.token, `{"probe_id":"text","prompt":"hello"}`)
	require.Equal(t, 200, good.Code)
	require.Equal(t, 1, forwarded)
	require.Len(t, repo.probes, 1)
	require.Contains(t, string(repo.probes[0]), `"upstream_model":"upstream-model"`)
	require.NotContains(t, string(repo.probes[0]), "private-site-key")
	require.NotContains(t, string(repo.probes[0]), repo.token)
	require.Equal(t, 401, request("/internal/model-identity/1/probe", "wrong", `{}`).Code)
	require.Equal(t, 401, request("/internal/model-identity/2/probe", repo.token, `{}`).Code)
	require.Equal(t, 400, request("/internal/model-identity/1/probe", repo.token, `{"prompt":"hi","account_id":99}`).Code)
	require.Equal(t, 1, forwarded)
	keys.k.Status = service.StatusAPIKeyDisabled
	require.Equal(t, 409, request("/internal/model-identity/1/probe", repo.token, `{"prompt":"hi"}`).Code)
	keys.k.Status = service.StatusActive
	keys.k.UserID = 6
	require.Equal(t, 409, request("/internal/model-identity/1/probe", repo.token, `{"prompt":"hi"}`).Code)
	keys.k.UserID = 5
	keys.k.Quota = 1
	keys.k.QuotaUsed = 1
	require.Equal(t, 409, request("/internal/model-identity/1/probe", repo.token, `{"prompt":"hi"}`).Code)
	keys.k.Quota = 0
	expired := time.Now().Add(-time.Hour)
	keys.k.ExpiresAt = &expired
	require.Equal(t, 409, request("/internal/model-identity/1/probe", repo.token, `{"prompt":"hi"}`).Code)
	keys.k.ExpiresAt = nil
	groups.exclusive = true
	require.Equal(t, 409, request("/internal/model-identity/1/probe", repo.token, `{"prompt":"hi"}`).Code)
	groups.exclusive = false
	accounts.a.GroupIDs = nil
	require.Equal(t, 409, request("/internal/model-identity/1/probe", repo.token, `{"prompt":"hi"}`).Code)
	repo.run.Status = "cancelled"
	require.Equal(t, 401, request("/internal/model-identity/1/probe", repo.token, `{"prompt":"hi"}`).Code)
	router.POST("/ordinary", func(c *gin.Context) {
		_, _, pinned := service.IdentityTargetFromContext(c.Request.Context())
		require.False(t, pinned)
		c.Status(200)
	})
	require.Equal(t, 200, request("/ordinary", repo.token, `{"account_id":42}`).Code)
}

func TestIdentityAdminRoutesRejectUnauthenticatedAndOrdinaryUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/admin", func(c *gin.Context) {
		if role := c.GetHeader("Test-Role"); role != "" {
			c.Set(string(middleware.ContextKeyUserRole), role)
		}
	}, middleware.AdminOnly())
	svc := service.NewModelIdentityService(nil, nil, nil, nil)
	registerModelIdentityRoutes(group, &handler.Handlers{Admin: &handler.AdminHandlers{ModelIdentity: admin.NewModelIdentityHandler(svc)}})
	for _, path := range []string{"/admin/model-identity/models", "/admin/model-identity/settings", "/admin/model-identity/accounts"} {
		for _, role := range []string{"", service.RoleUser} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Test-Role", role)
			router.ServeHTTP(w, req)
			if role == "" {
				require.Equal(t, http.StatusUnauthorized, w.Code)
			} else {
				require.Equal(t, http.StatusForbidden, w.Code)
			}
		}
	}
	for _, role := range []string{"", service.RoleUser} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/admin/model-identity/settings", strings.NewReader(`{"public_base_url":"https://site.example.com"}`))
		req.Header.Set("Test-Role", role)
		router.ServeHTTP(w, req)
		require.Contains(t, []int{401, 403}, w.Code)
	}
}

func TestIdentityCallbacksPublishNormalRoutingCompletionForTestUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, repo, _, _ := remoteIdentityFixture()
	monitor := service.NewOpsRoutingMonitorService(nil, nil)
	ops := &service.OpsService{}
	ops.SetRoutingMonitor(monitor)
	router := gin.New()
	// Simulated gateway selection emits the same event as setOpsSelectedAccount.
	auth := middleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		requestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string)
		account, _, pinned := service.IdentityTargetFromContext(c.Request.Context())
		require.True(t, pinned)
		monitor.ObserveSelection(service.OpsRoutingRequestInfo{ClientRequestID: requestID, UserID: repo.run.UserID, UserLabel: "test-user", AccountID: account, AccountName: "target", RequestedModel: repo.run.RequestModel})
		c.AbortWithStatusJSON(200, gin.H{"choices": []gin.H{{"message": gin.H{"content": "ok"}}}})
	})
	registerIdentityProbe(router, &handler.Handlers{}, svc, auth, func(*gin.Context) {}, handler.OpsErrorLoggerMiddleware(ops))
	for _, tc := range []struct{ path, token, body string }{
		{"/internal/model-identity/1/remote/v1/chat/completions", "private-test-key", `{"model":"public-model"}`},
		{"/internal/model-identity/1/probe", strings.Repeat("a", 64), `{"prompt":"hello"}`},
	} {
		repo.token = strings.Repeat("a", 64)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer "+tc.token)
		router.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code)
	}
	snapshot, err := monitor.Snapshot(context.Background())
	require.NoError(t, err)
	require.Empty(t, snapshot.Active)
	require.Len(t, snapshot.Recent, 2)
	for _, event := range snapshot.Recent {
		require.EqualValues(t, 5, event.UserID)
		require.Equal(t, "test-user", event.UserLabel)
		require.EqualValues(t, 42, event.AccountID)
		require.Equal(t, service.OpsRoutingEventCompleted, event.EventType)
	}
}

func remoteIdentityFixture() (*service.ModelIdentityService, *identityRouteRepo, *identityRouteKeyRepo, *identityRouteAccountRepo) {
	group := int64(7)
	now := time.Now()
	repo := &identityRouteRepo{run: &service.IdentityRun{ID: 1, AccountID: 42, UserID: 5, GroupID: 7, APIKeyID: 9, Status: "running", StartedAt: &now, RequestModel: "public-model"}}
	keys := &identityRouteKeyRepo{k: &service.APIKey{ID: 9, UserID: 5, GroupID: &group, Status: service.StatusActive, Key: "private-test-key"}}
	accounts := &identityRouteAccountRepo{a: &service.Account{ID: 42, Platform: service.PlatformOpenAI, GroupIDs: []int64{7}}}
	api := service.NewAPIKeyService(keys, &identityRouteUserRepo{}, &identityRouteGroupRepo{}, nil, nil, nil, &config.Config{})
	return service.NewModelIdentityService(repo, api, accounts, nil), repo, keys, accounts
}

func TestIdentityRemoteCallbackPinsAndRecordsWithoutCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, repo, keys, accounts := remoteIdentityFixture()
	router := gin.New()
	forwarded := 0
	auth := middleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		account, group, pinned := service.IdentityTargetFromContext(c.Request.Context())
		require.True(t, pinned)
		require.EqualValues(t, 42, account)
		require.EqualValues(t, 7, group)
		require.Empty(t, c.GetHeader("X-Account-ID"))
		service.RecordIdentityEvidence(c.Request.Context(), account, "mapped-upstream", "upstream-id")
		forwarded++
		c.AbortWithStatusJSON(200, gin.H{"choices": []gin.H{{"message": gin.H{"content": "private-test-key echoed"}}}, "usage": gin.H{"prompt_tokens": 8, "completion_tokens": 3}})
	})
	registerIdentityProbe(router, &handler.Handlers{}, svc, auth, func(*gin.Context) {}, func(*gin.Context) {})
	request := func(id, key, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/internal/model-identity/"+id+"/remote/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("X-Account-ID", "43")
		router.ServeHTTP(w, req)
		return w
	}
	body := `{"model":"public-model","messages":[{"role":"user","content":"hi"}],"account_id":43,"group_id":8,"stream":false}`
	require.Equal(t, 200, request("1", "private-test-key", body).Code)
	require.Equal(t, 1, forwarded)
	require.Len(t, repo.probes, 1)
	require.Contains(t, string(repo.probes[0]), `"target_account_id":42`)
	require.Contains(t, string(repo.probes[0]), "mapped-upstream")
	require.Contains(t, string(repo.probes[0]), "upstream-id")
	require.Contains(t, string(repo.probes[0]), "prompt_tokens")
	require.NotContains(t, string(repo.probes[0]), "private-test-key")
	require.Equal(t, 401, request("2", "private-test-key", body).Code)
	require.Equal(t, 401, request("1", "other-account-key", body).Code)
	require.Equal(t, 400, request("1", "private-test-key", `{"model":"another-model"}`).Code)
	require.Equal(t, 400, request("1", "private-test-key", `{"model":"public-model","max_tokens":9000}`).Code)
	keys.k.Status = service.StatusAPIKeyDisabled
	require.Equal(t, 409, request("1", "private-test-key", body).Code)
	keys.k.Status = service.StatusActive
	accounts.a.GroupIDs = nil
	require.Equal(t, 409, request("1", "private-test-key", body).Code)
	accounts.a.GroupIDs = []int64{7}
	expired := time.Now().Add(-21 * time.Minute)
	repo.run.StartedAt = &expired
	require.Equal(t, 401, request("1", "private-test-key", body).Code)
	now := time.Now()
	repo.run.StartedAt = &now
	repo.run.Status = "cancelling"
	require.Equal(t, 401, request("1", "private-test-key", body).Code)
	require.Equal(t, 1, forwarded)
}

func TestIdentityRemoteCallbackQueuesAboveTenAndReleasesSlots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, repo, _, _ := remoteIdentityFixture()
	router := gin.New()
	entered := make(chan struct{}, service.IdentityProbeConcurrency+1)
	release := make(chan struct{})
	defer close(release)
	router.POST("/internal/model-identity/:run/remote/v1/chat/completions", remoteIdentityProbe(svc), func(c *gin.Context) { entered <- struct{}{}; <-release; c.JSON(200, gin.H{"ok": true}) })
	request := func() int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/internal/model-identity/1/remote/v1/chat/completions", strings.NewReader(`{"model":"public-model","messages":[]}`))
		req.Header.Set("Authorization", "Bearer private-test-key")
		router.ServeHTTP(w, req)
		return w.Code
	}
	var wg sync.WaitGroup
	for i := 0; i < service.IdentityProbeConcurrency; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); request() }()
	}
	for i := 0; i < service.IdentityProbeConcurrency; i++ {
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("probe did not acquire a slot")
		}
	}
	queued := make(chan int, 1)
	go func() { queued <- request() }()
	select {
	case code := <-queued:
		t.Fatalf("full pool should queue, got %d", code)
	case <-time.After(150 * time.Millisecond):
	}
	release <- struct{}{}
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("queued probe did not resume")
	}
	for i := 0; i < service.IdentityProbeConcurrency; i++ {
		release <- struct{}{}
	}
	wg.Wait()
	require.Equal(t, 200, <-queued)
	require.Equal(t, 0, repo.inflight)
	require.Len(t, repo.probes, service.IdentityProbeConcurrency+1)
}
