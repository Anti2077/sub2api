package routes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type identityRouteRepo struct {
	service.IdentityRepository
	run    *service.IdentityRun
	token  string
	probes []json.RawMessage
}

func (r *identityRouteRepo) Authorize(_ context.Context, id int64, hash string, _ time.Time) (*service.IdentityRun, error) {
	if id != r.run.ID || hash != service.HashIdentityToken(r.token) || r.run.Status != "running" {
		return nil, errors.New("unauthorized")
	}
	return r.run, nil
}
func (r *identityRouteRepo) ProbeSlot(context.Context, int64, bool) error { return nil }
func (r *identityRouteRepo) AppendProbe(_ context.Context, _ int64, p json.RawMessage) error {
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
	return r.k, nil
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
	svc := service.NewModelIdentityService(repo, apiKeySvc, accounts)
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
	registerIdentityProbe(router, &handler.Handlers{}, svc, auth, func(*gin.Context) {})
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
	svc := service.NewModelIdentityService(nil, nil, nil)
	registerModelIdentityRoutes(group, &handler.Handlers{Admin: &handler.AdminHandlers{ModelIdentity: admin.NewModelIdentityHandler(svc)}})
	for _, role := range []string{"", service.RoleUser} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/model-identity/models", nil)
		req.Header.Set("Test-Role", role)
		router.ServeHTTP(w, req)
		if role == "" {
			require.Equal(t, http.StatusUnauthorized, w.Code)
		} else {
			require.Equal(t, http.StatusForbidden, w.Code)
		}
	}
}
