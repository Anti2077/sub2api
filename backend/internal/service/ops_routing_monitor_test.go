package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func routingInfo(requestID string, accountID int64) OpsRoutingRequestInfo {
	return OpsRoutingRequestInfo{
		RequestID: requestID, ClientRequestID: "client-" + requestID, UserLabel: "alice",
		RequestedModel: "claude-3-7-sonnet", UpstreamModel: "claude-3-7-sonnet-20250219",
		Platform: PlatformAnthropic, AccountID: accountID, AccountName: "account-" + string(rune('0'+accountID)), AccountPlatform: PlatformAnthropic,
	}
}

func TestOpsRoutingMonitorLifecycleAndFailover(t *testing.T) {
	monitor := NewOpsRoutingMonitorService(nil, nil)
	updates, cancel := monitor.Subscribe(context.Background())
	defer cancel()

	monitor.ObserveSelection(routingInfo("req-1", 1))
	var started map[string]any
	require.NoError(t, json.Unmarshal(<-updates, &started))
	require.Equal(t, "routing_event", started["type"])

	monitor.ObserveSelection(routingInfo("req-1", 2))
	var switchedEnvelope map[string]any
	require.NoError(t, json.Unmarshal(<-updates, &switchedEnvelope))
	switched, ok := switchedEnvelope["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "switched", switched["event_type"])

	monitor.Finish(routingInfo("req-1", 2), "200", 125*time.Millisecond, false, "")
	var completedEnvelope map[string]any
	require.NoError(t, json.Unmarshal(<-updates, &completedEnvelope))
	completed, ok := completedEnvelope["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "completed", completed["event_type"])
	require.Equal(t, float64(2), completed["attempt_count"])
	require.Len(t, completed["hops"], 2)

	snapshot, err := monitor.Snapshot(context.Background())
	require.NoError(t, err)
	require.Empty(t, snapshot.Active)
	require.Len(t, snapshot.Recent, 1)
}

func TestOpsRoutingUserLabelPrefersUsernameAndMasksEmail(t *testing.T) {
	require.Equal(t, "alice", OpsRoutingUserLabel(&User{Username: "alice", Email: "alice@example.com"}))
	require.Equal(t, "a***e@example.com", OpsRoutingUserLabel(&User{Email: "alice@example.com"}))
	require.Equal(t, "a@x.io", OpsRoutingUserLabel(&User{Email: "a@x.io"}))
	require.Equal(t, "user", OpsRoutingUserLabel(&User{Email: "invalid"}))
}

func TestOpsRoutingMonitorRedisSnapshotAndPubSub(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	monitorOne := NewOpsRoutingMonitorService(nil, client)
	monitorTwo := NewOpsRoutingMonitorService(nil, client)
	monitorOne.Start(context.Background())
	monitorTwo.Start(context.Background())
	t.Cleanup(func() {
		monitorOne.Stop()
		monitorTwo.Stop()
		_ = client.Close()
	})

	updates, cancel := monitorTwo.Subscribe(context.Background())
	defer cancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && client.PubSubNumSub(context.Background(), opsRoutingChannel).Val()[opsRoutingChannel] < 1 {
		time.Sleep(10 * time.Millisecond)
	}
	require.GreaterOrEqual(t, client.PubSubNumSub(context.Background(), opsRoutingChannel).Val()[opsRoutingChannel], int64(1))
	monitorOne.ObserveSelection(routingInfo("req-redis", 3))
	select {
	case payload := <-updates:
		var envelope map[string]any
		require.NoError(t, json.Unmarshal(payload, &envelope))
		require.Equal(t, "routing_event", envelope["type"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cross-instance routing event")
	}

	snapshot, err := monitorTwo.Snapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, snapshot.Active, 1)
	require.Equal(t, int64(3), snapshot.Active[0].AccountID)
}

func TestOpsRoutingMonitorRedisSnapshotFiltersExpiredEvents(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	monitor := NewOpsRoutingMonitorService(nil, client)
	t.Cleanup(func() { _ = client.Close() })

	old := &OpsRoutingEvent{
		EventID: "old-event", EventType: OpsRoutingEventStarted,
		OccurredAt: time.Now().UTC().Add(-opsRoutingActiveTTL - time.Second),
		RouteKey:   "old-route", Status: "active", AccountID: 10,
	}
	oldPayload, err := json.Marshal(old)
	require.NoError(t, err)
	require.NoError(t, client.HSet(context.Background(), opsRoutingActiveKey, old.RouteKey, oldPayload).Err())
	require.NoError(t, client.LPush(context.Background(), opsRoutingRecentKey, oldPayload).Err())

	snapshot, err := monitor.Snapshot(context.Background())
	require.NoError(t, err)
	require.Empty(t, snapshot.Active)
	require.Empty(t, snapshot.Recent)
}

func TestOpsRoutingMonitorStopCancelsRedisSubscriber(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	monitor := NewOpsRoutingMonitorService(nil, client)
	monitor.Start(context.Background())
	require.Eventually(t, func() bool {
		return client.PubSubNumSub(context.Background(), opsRoutingChannel).Val()[opsRoutingChannel] == 1
	}, time.Second, 10*time.Millisecond)

	monitor.Stop()
	require.Eventually(t, func() bool {
		return client.PubSubNumSub(context.Background(), opsRoutingChannel).Val()[opsRoutingChannel] == 0
	}, time.Second, 10*time.Millisecond)
	_ = client.Close()
}
