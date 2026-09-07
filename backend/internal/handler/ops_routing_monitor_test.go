package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpsRoutingMonitorWebSocketTurnsUseIndependentRouteKeys(t *testing.T) {
	monitor := service.NewOpsRoutingMonitorService(nil, nil)
	updates, cancel := monitor.Subscribe(context.Background())
	defer cancel()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("GET", "/v1/responses", nil)
	ctx := context.WithValue(req.Context(), ctxkey.RequestID, "req-ws")
	ctx = context.WithValue(ctx, ctxkey.ClientRequestID, "client-ws")
	c.Request = req.WithContext(ctx)

	ops := &service.OpsService{}
	ops.SetRoutingMonitor(monitor)
	setOpsRoutingMonitor(c, ops)
	setOpsRequestContext(c, "model-a", true)
	setOpsRoutingTurn(c, 2)
	setOpsSelectedAccount(c, 7, service.PlatformOpenAI, "primary")

	var started service.OpsRoutingEvent
	requireRoutingEvent(t, <-updates, &started)
	require.Equal(t, "req-ws:turn:2", started.RouteKey)
	require.Equal(t, 2, started.Turn)
	require.Equal(t, "model-a", started.RequestedModel)

	recordOpsRoutingTurnCompletion(c, 2, "model-a", "gpt-5.5", &service.OpenAIForwardResult{
		UpstreamModel: "gpt-5.5",
	}, nil, time.Now().Add(-40*time.Millisecond))

	var completed service.OpsRoutingEvent
	requireRoutingEvent(t, <-updates, &completed)
	require.Equal(t, service.OpsRoutingEventCompleted, completed.EventType)
	require.Equal(t, "gpt-5.5", completed.UpstreamModel)
	require.GreaterOrEqual(t, completed.DurationMs, int64(0))
}

func requireRoutingEvent(t *testing.T, payload []byte, event *service.OpsRoutingEvent) {
	t.Helper()
	var envelope struct {
		Type string                  `json:"type"`
		Data service.OpsRoutingEvent `json:"data"`
	}
	require.NoError(t, json.Unmarshal(payload, &envelope))
	require.Equal(t, "routing_event", envelope.Type)
	*event = envelope.Data
}
