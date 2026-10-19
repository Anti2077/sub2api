package routes

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIdentityFingerpointProtocolsBindCapabilityAndBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for protocol, path := range map[string]string{"chat_completions": "chat/completions", "responses": "responses", "messages": "messages"} {
		t.Run(protocol, func(t *testing.T) {
			svc, repo, _, _ := remoteIdentityFixture()
			repo.token = strings.Repeat("a", 64)
			repo.run.Options.Protocol = protocol
			repo.run.EngineCommit = service.IdentityEngineCommit
			forwarded := 0
			auth := middleware.APIKeyAuthMiddleware(func(c *gin.Context) {
				account, group, pinned := service.IdentityTargetFromContext(c.Request.Context())
				require.True(t, pinned)
				require.EqualValues(t, 42, account)
				require.EqualValues(t, 7, group)
				require.Equal(t, "Bearer private-test-key", c.GetHeader("Authorization"))
				require.Empty(t, c.GetHeader("X-Account-ID"))
				var input map[string]any
				require.NoError(t, json.NewDecoder(c.Request.Body).Decode(&input))
				require.Equal(t, "public-model", input["model"])
				if protocol == "responses" {
					require.Equal(t, false, input["store"])
				}
				forwarded++
				c.AbortWithStatusJSON(200, gin.H{"output": "private-test-key must not enter evidence"})
			})
			router := gin.New()
			registerIdentityCompletions(router, &handler.Handlers{}, svc, auth, func(*gin.Context) {}, func(*gin.Context) {})
			request := func(token, body string) int {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/internal/model-identity/1/v1/"+path, strings.NewReader(body))
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("X-Account-ID", "99")
				router.ServeHTTP(w, req)
				return w.Code
			}
			require.Equal(t, 401, request("private-test-key", `{}`))
			require.Equal(t, 400, request(repo.token, `{"model":"other","account_id":99}`))
			require.Equal(t, 400, request(repo.token, `{"max_tokens":8193}`))
			require.Equal(t, 200, request(repo.token, `{"model":"client-override","messages":[],"store":true}`))
			require.Equal(t, 1, forwarded)
			require.Len(t, repo.probes, 1)
			require.NotContains(t, string(repo.probes[0]), "private-test-key")
			require.NotContains(t, string(repo.probes[0]), repo.token)
			require.Contains(t, string(repo.probes[0]), `"target_account_id":42`)
			repo.run.Status = "cancelling"
			require.Equal(t, 401, request(repo.token, `{}`))
			require.Equal(t, 0, repo.inflight)
		})
	}
}
