package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	opsRoutingRecentWindow = 30 * time.Second
	opsRoutingActiveTTL    = 2 * time.Minute
	opsRoutingRecentTTL    = 35 * time.Second
	opsRoutingMaxEvents    = 500
	opsRoutingMaxPayload   = 16 * 1024
	opsRoutingSubscriberQ  = 64
	opsRoutingRedisQueue   = 256
	opsRoutingChannel      = "sub2api:ops:routing"
	opsRoutingRecentKey    = "sub2api:ops:routing:recent"
	opsRoutingActiveKey    = "sub2api:ops:routing:active"
)

type OpsRoutingEventType string

const (
	OpsRoutingEventStarted   OpsRoutingEventType = "started"
	OpsRoutingEventSwitched  OpsRoutingEventType = "switched"
	OpsRoutingEventCompleted OpsRoutingEventType = "completed"
	OpsRoutingEventFailed    OpsRoutingEventType = "failed"
)

type OpsRoutingHop struct {
	AccountID   int64     `json:"account_id"`
	AccountName string    `json:"account_name"`
	Platform    string    `json:"platform"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type OpsRoutingEvent struct {
	EventID         string              `json:"event_id"`
	EventType       OpsRoutingEventType `json:"event_type"`
	OccurredAt      time.Time           `json:"occurred_at"`
	RequestID       string              `json:"request_id,omitempty"`
	ClientRequestID string              `json:"client_request_id,omitempty"`
	RouteKey        string              `json:"route_key"`
	Turn            int                 `json:"turn,omitempty"`
	UserID          int64               `json:"user_id,omitempty"`
	UserLabel       string              `json:"user_label,omitempty"`
	RequestedModel  string              `json:"requested_model,omitempty"`
	UpstreamModel   string              `json:"upstream_model,omitempty"`
	Platform        string              `json:"platform,omitempty"`
	AccountID       int64               `json:"account_id,omitempty"`
	AccountName     string              `json:"account_name,omitempty"`
	AccountPlatform string              `json:"account_platform,omitempty"`
	Status          string              `json:"status"`
	DurationMs      int64               `json:"duration_ms,omitempty"`
	ErrorSummary    string              `json:"error_summary,omitempty"`
	AttemptCount    int                 `json:"attempt_count"`
	Hops            []OpsRoutingHop     `json:"hops"`
}

type OpsRoutingSnapshot struct {
	GeneratedAt time.Time          `json:"generated_at"`
	Active      []*OpsRoutingEvent `json:"active"`
	Recent      []*OpsRoutingEvent `json:"recent"`
}

type OpsRoutingRequestInfo struct {
	RequestID       string
	ClientRequestID string
	UserID          int64
	UserLabel       string
	RequestedModel  string
	UpstreamModel   string
	Platform        string
	AccountID       int64
	AccountName     string
	AccountPlatform string
	Turn            int
}

type opsRoutingPersistJob struct {
	event  *OpsRoutingEvent
	active bool
}

type OpsRoutingMonitorService struct {
	redis    *redis.Client
	instance string

	mu           sync.Mutex
	active       map[string]*OpsRoutingEvent
	recent       []*OpsRoutingEvent
	subscribers  map[uint64]chan []byte
	nextSubID    uint64
	nextEventSeq atomic.Uint64
	redisQueue   chan opsRoutingPersistJob
	stop         chan struct{}
	stopOnce     sync.Once
	runMu        sync.Mutex
	runCancel    context.CancelFunc
	runStarted   bool
	pubsubMu     sync.Mutex
	pubsub       *redis.PubSub
}

func NewOpsRoutingMonitorService(_ UserRepository, redisClient *redis.Client) *OpsRoutingMonitorService {
	return &OpsRoutingMonitorService{
		redis:       redisClient,
		instance:    fmt.Sprintf("%d", time.Now().UnixNano()),
		active:      make(map[string]*OpsRoutingEvent),
		subscribers: make(map[uint64]chan []byte),
		redisQueue:  make(chan opsRoutingPersistJob, opsRoutingRedisQueue),
		stop:        make(chan struct{}),
	}
}

// OpsRoutingUserLabel prefers the configured username and falls back to a
// deliberately coarse email mask. It never exposes an API key or credential.
func OpsRoutingUserLabel(user *User) string {
	if user == nil {
		return "user"
	}
	if username := strings.TrimSpace(user.Username); username != "" {
		return trimField(username, 128)
	}
	email := strings.TrimSpace(user.Email)
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "user"
	}
	local := parts[0]
	if len(local) <= 2 {
		local = local[:1]
	} else {
		local = local[:1] + "***" + local[len(local)-1:]
	}
	return trimField(local+"@"+parts[1], 160)
}

func (s *OpsRoutingMonitorService) Start(ctx context.Context) {
	if s == nil || s.redis == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.runMu.Lock()
	if s.runStarted {
		s.runMu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.runCancel = cancel
	s.runStarted = true
	s.runMu.Unlock()
	go s.redisWriter()
	go s.redisSubscriber(runCtx)
}

func (s *OpsRoutingMonitorService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stop)
		s.runMu.Lock()
		cancel := s.runCancel
		s.runCancel = nil
		s.runMu.Unlock()
		if cancel != nil {
			cancel()
		}
		s.pubsubMu.Lock()
		pubsub := s.pubsub
		s.pubsubMu.Unlock()
		if pubsub != nil {
			_ = pubsub.Close()
		}
	})
}

func (s *OpsRoutingMonitorService) redisWriter() {
	for {
		select {
		case job := <-s.redisQueue:
			if job.event == nil || s.redis == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			payload, err := json.Marshal(job.event)
			if err == nil && len(payload) <= opsRoutingMaxPayload {
				pipe := s.redis.TxPipeline()
				if job.active {
					pipe.HSet(ctx, opsRoutingActiveKey, job.event.RouteKey, payload)
					pipe.Expire(ctx, opsRoutingActiveKey, opsRoutingActiveTTL)
				} else {
					pipe.HDel(ctx, opsRoutingActiveKey, job.event.RouteKey)
					pipe.LPush(ctx, opsRoutingRecentKey, payload)
					pipe.LTrim(ctx, opsRoutingRecentKey, 0, opsRoutingMaxEvents-1)
					pipe.Expire(ctx, opsRoutingRecentKey, opsRoutingRecentTTL)
				}
				if _, err = pipe.Exec(ctx); err == nil {
					_ = s.redis.Publish(ctx, opsRoutingChannel, payload).Err()
				}
			}
			cancel()
		case <-s.stop:
			return
		}
	}
}

func (s *OpsRoutingMonitorService) redisSubscriber(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	go func() {
		select {
		case <-s.stop:
			cancel()
		case <-ctx.Done():
		}
	}()
	pubsub := s.redis.Subscribe(ctx, opsRoutingChannel)
	s.pubsubMu.Lock()
	s.pubsub = pubsub
	s.pubsubMu.Unlock()
	defer func() {
		_ = pubsub.Close()
		s.pubsubMu.Lock()
		if s.pubsub == pubsub {
			s.pubsub = nil
		}
		s.pubsubMu.Unlock()
	}()
	if _, err := pubsub.Receive(ctx); err != nil {
		return
	}
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			select {
			case <-s.stop:
				return
			default:
			}
			continue
		}
		var event OpsRoutingEvent
		if json.Unmarshal([]byte(msg.Payload), &event) != nil || event.EventID == "" {
			continue
		}
		// The publishing instance already delivered this event locally.
		if strings.HasPrefix(event.EventID, s.instance+"-") {
			continue
		}
		s.broadcast(&event)
	}
}

func (s *OpsRoutingMonitorService) Subscribe(ctx context.Context) (<-chan []byte, func()) {
	ch := make(chan []byte, opsRoutingSubscriberQ)
	if s == nil {
		close(ch)
		return ch, func() {}
	}
	s.mu.Lock()
	id := s.nextSubID
	s.nextSubID++
	s.subscribers[id] = ch
	s.mu.Unlock()
	cancel := func() {
		s.mu.Lock()
		if current, ok := s.subscribers[id]; ok {
			delete(s.subscribers, id)
			close(current)
		}
		s.mu.Unlock()
	}
	if ctx != nil {
		go func() {
			<-ctx.Done()
			cancel()
		}()
	}
	return ch, cancel
}

func (s *OpsRoutingMonitorService) broadcast(event *OpsRoutingEvent) {
	if event == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{"type": "routing_event", "data": event})
	if err != nil || len(payload) > opsRoutingMaxPayload {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- payload:
		default:
		}
	}
}

func (s *OpsRoutingMonitorService) enqueueRedis(event *OpsRoutingEvent, active bool) {
	if s == nil || s.redis == nil || event == nil {
		return
	}
	select {
	case s.redisQueue <- opsRoutingPersistJob{event: event, active: active}:
	case <-s.stop:
		return
	default:
		log.Printf("[OpsRoutingMonitor] redis queue full; dropping persistence for %s", event.EventID)
	}
}

func (s *OpsRoutingMonitorService) nextEventID() string {
	return fmt.Sprintf("%s-%d", s.instance, s.nextEventSeq.Add(1))
}

func (s *OpsRoutingMonitorService) ObserveSelection(info OpsRoutingRequestInfo) {
	if s == nil || info.AccountID <= 0 {
		return
	}
	info.RequestID = strings.TrimSpace(info.RequestID)
	info.ClientRequestID = strings.TrimSpace(info.ClientRequestID)
	routeKey := info.RequestID
	if routeKey == "" {
		routeKey = info.ClientRequestID
	}
	if routeKey == "" {
		return
	}
	if info.Turn > 0 {
		routeKey = fmt.Sprintf("%s:turn:%d", routeKey, info.Turn)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	current := s.active[routeKey]
	if current == nil {
		current = &OpsRoutingEvent{
			EventID: s.nextEventID(), EventType: OpsRoutingEventStarted, OccurredAt: now,
			RequestID: info.RequestID, ClientRequestID: info.ClientRequestID, RouteKey: routeKey,
			Turn: info.Turn, UserID: info.UserID, UserLabel: trimField(info.UserLabel, 128), RequestedModel: trimField(info.RequestedModel, 256),
			UpstreamModel: trimField(info.UpstreamModel, 256), Platform: trimField(info.Platform, 64), Status: "active",
		}
	}
	if current.AccountID == info.AccountID {
		current.AccountName = trimField(info.AccountName, 128)
		current.AccountPlatform = trimField(routingFirstNonEmpty(info.AccountPlatform, info.Platform), 64)
		current.UpstreamModel = trimField(info.UpstreamModel, 256)
		s.active[routeKey] = current
		s.mu.Unlock()
		return
	}
	if current.AccountID > 0 {
		current.EventType = OpsRoutingEventSwitched
		current.EventID = s.nextEventID()
	} else {
		current.EventType = OpsRoutingEventStarted
	}
	current.OccurredAt = now
	current.AccountID = info.AccountID
	current.AccountName = trimField(info.AccountName, 128)
	current.AccountPlatform = trimField(routingFirstNonEmpty(info.AccountPlatform, info.Platform), 64)
	current.Platform = trimField(routingFirstNonEmpty(info.Platform, info.AccountPlatform), 64)
	current.UpstreamModel = trimField(info.UpstreamModel, 256)
	current.AttemptCount++
	current.Hops = append(current.Hops, OpsRoutingHop{AccountID: info.AccountID, AccountName: trimField(info.AccountName, 128), Platform: trimField(routingFirstNonEmpty(info.AccountPlatform, info.Platform), 64), OccurredAt: now})
	if len(current.Hops) > 32 {
		current.Hops = current.Hops[len(current.Hops)-32:]
	}
	copyEvent := cloneRoutingEvent(current)
	s.active[routeKey] = current
	s.mu.Unlock()
	s.broadcast(copyEvent)
	s.enqueueRedis(copyEvent, true)
}

func (s *OpsRoutingMonitorService) Finish(info OpsRoutingRequestInfo, status string, duration time.Duration, failed bool, errorSummary string) {
	if s == nil {
		return
	}
	routeKey := strings.TrimSpace(info.RequestID)
	if routeKey == "" {
		routeKey = strings.TrimSpace(info.ClientRequestID)
	}
	if routeKey == "" {
		return
	}
	if info.Turn > 0 {
		routeKey = fmt.Sprintf("%s:turn:%d", routeKey, info.Turn)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	event := s.active[routeKey]
	if event == nil {
		s.mu.Unlock()
		return
	}
	delete(s.active, routeKey)
	if requestedModel := strings.TrimSpace(info.RequestedModel); requestedModel != "" {
		event.RequestedModel = trimField(requestedModel, 256)
	}
	if upstreamModel := strings.TrimSpace(info.UpstreamModel); upstreamModel != "" {
		event.UpstreamModel = trimField(upstreamModel, 256)
	}
	if platform := strings.TrimSpace(info.Platform); platform != "" {
		event.Platform = trimField(platform, 64)
	}
	event.EventID = s.nextEventID()
	event.EventType = OpsRoutingEventCompleted
	if failed {
		event.EventType = OpsRoutingEventFailed
	}
	event.OccurredAt = now
	event.Status = trimField(routingFirstNonEmpty(status, map[bool]string{true: "failed", false: "completed"}[failed]), 32)
	event.DurationMs = duration.Milliseconds()
	event.ErrorSummary = trimField(errorSummary, 512)
	copyEvent := cloneRoutingEvent(event)
	s.recent = append([]*OpsRoutingEvent{copyEvent}, s.recent...)
	s.pruneLocked(now)
	s.mu.Unlock()
	s.broadcast(copyEvent)
	s.enqueueRedis(copyEvent, false)
}

func (s *OpsRoutingMonitorService) Snapshot(ctx context.Context) (*OpsRoutingSnapshot, error) {
	if s == nil {
		return &OpsRoutingSnapshot{GeneratedAt: time.Now().UTC(), Active: []*OpsRoutingEvent{}, Recent: []*OpsRoutingEvent{}}, nil
	}
	now := time.Now().UTC()
	s.mu.Lock()
	s.pruneLocked(now)
	active := cloneRoutingEvents(s.active)
	recent := cloneRoutingSlice(s.recent)
	s.mu.Unlock()
	if s.redis != nil {
		redisActive, redisRecent := s.readRedisSnapshot(ctx, now)
		active = mergeRoutingEvents(active, redisActive, true)
		recent = mergeRoutingEvents(recent, redisRecent, false)
	}
	sort.Slice(active, func(i, j int) bool { return active[i].OccurredAt.Before(active[j].OccurredAt) })
	sort.Slice(recent, func(i, j int) bool { return recent[i].OccurredAt.After(recent[j].OccurredAt) })
	return &OpsRoutingSnapshot{GeneratedAt: now, Active: active, Recent: recent}, nil
}

func (s *OpsRoutingMonitorService) readRedisSnapshot(ctx context.Context, now time.Time) ([]*OpsRoutingEvent, []*OpsRoutingEvent) {
	if ctx == nil {
		ctx = context.Background()
	}
	if values, err := s.redis.HGetAll(ctx, opsRoutingActiveKey).Result(); err == nil {
		active := make([]*OpsRoutingEvent, 0, len(values))
		for _, raw := range values {
			var event OpsRoutingEvent
			if json.Unmarshal([]byte(raw), &event) == nil && !event.OccurredAt.IsZero() && now.Sub(event.OccurredAt) <= opsRoutingActiveTTL {
				active = append(active, &event)
			}
		}
		recentRaw, recentErr := s.redis.LRange(ctx, opsRoutingRecentKey, 0, opsRoutingMaxEvents-1).Result()
		recent := make([]*OpsRoutingEvent, 0, len(recentRaw))
		if recentErr == nil {
			for _, raw := range recentRaw {
				var event OpsRoutingEvent
				if json.Unmarshal([]byte(raw), &event) == nil && !event.OccurredAt.IsZero() && now.Sub(event.OccurredAt) <= opsRoutingRecentWindow {
					recent = append(recent, &event)
				}
			}
		}
		return active, recent
	}
	return nil, nil
}

func (s *OpsRoutingMonitorService) pruneLocked(now time.Time) {
	for key, event := range s.active {
		if event == nil || now.Sub(event.OccurredAt) > opsRoutingActiveTTL {
			delete(s.active, key)
		}
	}
	filtered := s.recent[:0]
	for _, event := range s.recent {
		if event != nil && now.Sub(event.OccurredAt) <= opsRoutingRecentWindow {
			filtered = append(filtered, event)
		}
	}
	if len(filtered) > opsRoutingMaxEvents {
		filtered = filtered[:opsRoutingMaxEvents]
	}
	s.recent = filtered
}

func cloneRoutingEvent(event *OpsRoutingEvent) *OpsRoutingEvent {
	if event == nil {
		return nil
	}
	copyEvent := *event
	copyEvent.Hops = append([]OpsRoutingHop(nil), event.Hops...)
	return &copyEvent
}

func cloneRoutingSlice(events []*OpsRoutingEvent) []*OpsRoutingEvent {
	out := make([]*OpsRoutingEvent, 0, len(events))
	for _, event := range events {
		if copyEvent := cloneRoutingEvent(event); copyEvent != nil {
			out = append(out, copyEvent)
		}
	}
	return out
}

func cloneRoutingEvents(events map[string]*OpsRoutingEvent) []*OpsRoutingEvent {
	out := make([]*OpsRoutingEvent, 0, len(events))
	for _, event := range events {
		if copyEvent := cloneRoutingEvent(event); copyEvent != nil {
			out = append(out, copyEvent)
		}
	}
	return out
}

func mergeRoutingEvents(local, remote []*OpsRoutingEvent, _ bool) []*OpsRoutingEvent {
	byKey := make(map[string]*OpsRoutingEvent, len(local)+len(remote))
	for _, event := range append(local, remote...) {
		if event == nil || event.RouteKey == "" {
			continue
		}
		if current, ok := byKey[event.RouteKey]; !ok || event.OccurredAt.After(current.OccurredAt) {
			byKey[event.RouteKey] = cloneRoutingEvent(event)
		}
	}
	out := make([]*OpsRoutingEvent, 0, len(byKey))
	for _, event := range byKey {
		out = append(out, event)
	}
	return out
}

func trimField(value string, max int) string {
	value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
	if len(value) > max {
		return value[:max]
	}
	return value
}

func routingFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
