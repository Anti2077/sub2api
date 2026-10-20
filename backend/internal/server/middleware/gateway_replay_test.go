package middleware

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"testing"
)

type replayStub struct {
	r     rewards.GatewayReceipt
	calls int
	fail  bool
}

func (s *replayStub) ClaimGateway(_ context.Context, _ int64, _, fp, rid string) (rewards.GatewayReceipt, error) {
	if s.fail {
		return rewards.GatewayReceipt{}, errors.New("db unavailable")
	}
	if s.calls == 0 {
		s.r = rewards.GatewayReceipt{Claimed: true, Fingerprint: fp, RequestID: rid, State: "pending"}
		s.calls++
		return s.r, nil
	}
	r := s.r
	r.Claimed = false
	return r, nil
}
func (s *replayStub) FinishGateway(_ context.Context, _ int64, _ string, status int, ct string, b []byte, overflow bool) error {
	s.r.Status = status
	s.r.ContentType = ct
	s.r.Body = append([]byte{}, b...)
	s.r.State = "complete"
	if overflow {
		s.r.State = "unavailable"
	}
	return nil
}
func TestGatewayReplayPreventsSecondProviderCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &replayStub{}
	calls := 0
	r := gin.New()
	r.Use(ClientRequestID(), func(c *gin.Context) { c.Set(string(ContextKeyAPIKey), &service.APIKey{ID: 7}) }, GatewayReplay(s))
	r.POST("/v1/responses", func(c *gin.Context) {
		calls++
		c.Header("Content-Type", "text/event-stream")
		_, _ = c.Writer.WriteString("data: one\n\n")
		c.Writer.Flush()
		_, _ = c.Writer.Write([]byte("data: two\n\n"))
	})
	run := func(body string) *httptest.ResponseRecorder {
		q := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body))
		q.Header.Set("Idempotency-Key", "test-key")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, q)
		return w
	}
	first := run(`{"input":"same"}`)
	second := run(`{"input":"same"}`)
	require.Equal(t, 200, first.Code)
	require.Equal(t, first.Body.String(), second.Body.String())
	require.Equal(t, "true", second.Header().Get("Idempotency-Replayed"))
	require.Equal(t, first.Header().Get("X-Client-Request-ID"), second.Header().Get("X-Client-Request-ID"))
	require.Equal(t, 1, calls)
	require.Equal(t, 409, run(`{"input":"different"}`).Code)
	require.Equal(t, 1, calls)
	s.r.State = "pending"
	require.Equal(t, 409, run(`{"input":"same"}`).Code)
	s.r.State = "unavailable"
	require.Equal(t, 409, run(`{"input":"same"}`).Code)
	s.fail = true
	require.Equal(t, 503, run(`{"input":"same"}`).Code)
	require.Equal(t, 1, calls)
}
