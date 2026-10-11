package service

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const IdentityProbeMaxRetries = 5

func IdentityProbeRetryableStatus(status int) bool {
	switch status {
	case 0, http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// IdentityTargetRetryable permits transient upstream failures to be retried
// while the identity callback is pinned to one account. It never applies to
// ordinary client traffic or configuration/authentication failures.
func IdentityTargetRetryable(ctx context.Context, status int) bool {
	_, _, pinned := IdentityTargetFromContext(ctx)
	return pinned && IdentityProbeRetryableStatus(status)
}

func IdentityProbeRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 3 {
		attempt = 3
	}
	return time.Duration(1<<(attempt-1)) * time.Second
}

type identityTargetKey struct{}
type identityTarget struct{ AccountID, GroupID int64 }

// WithIdentityTarget is installed only after authenticating a detection capability.
// No public middleware translates headers or request parameters into this context.
func WithIdentityTarget(ctx context.Context, accountID, groupID int64) context.Context {
	return context.WithValue(ctx, identityTargetKey{}, identityTarget{accountID, groupID})
}
func IdentityTargetFromContext(ctx context.Context) (int64, int64, bool) {
	p, ok := ctx.Value(identityTargetKey{}).(identityTarget)
	return p.AccountID, p.GroupID, ok
}

// Only authenticated identity probes may ignore the target's manual scheduling
// switch. Check a copy so cached/repository accounts and ordinary traffic stay
// paused, while every other availability gate still applies.
func (a *Account) isSchedulableForIdentityContext(ctx context.Context) bool {
	if a == nil {
		return false
	}
	id, groupID, pinned := IdentityTargetFromContext(ctx)
	if pinned && id == a.ID && identityAccountInGroup(a, groupID) {
		probeAccount := *a
		probeAccount.Schedulable = true
		return probeAccount.IsSchedulable()
	}
	return a.IsSchedulable()
}

func identityTargetAllowed(ctx context.Context, g *int64, excluded map[int64]struct{}) (int64, bool, error) {
	id, group, ok := IdentityTargetFromContext(ctx)
	if !ok {
		return 0, false, nil
	}
	if g == nil || *g != group {
		return id, true, fmt.Errorf("identity detection cannot change its billing group")
	}
	if _, failed := excluded[id]; failed {
		return id, true, fmt.Errorf("identity target account failed; account failover is forbidden")
	}
	return id, true, nil
}
func (s *GatewayService) selectIdentityTarget(ctx context.Context, groupID *int64, model string, excluded map[int64]struct{}) (*AccountSelectionResult, bool, error) {
	id, pinned, e := identityTargetAllowed(ctx, groupID, excluded)
	if !pinned || e != nil {
		return nil, pinned, e
	}
	group, e := s.groupRepo.GetByID(ctx, *groupID)
	if e != nil {
		return nil, true, e
	}
	if group.ClaudeCodeOnly {
		return nil, true, fmt.Errorf("identity target group requires a Claude Code client")
	}
	ctx = s.withGroupContext(ctx, group)
	ctx = s.withGatewayProfitControlGate(ctx, groupID)
	a, e := s.accountRepo.GetByID(ctx, id)
	if e != nil {
		return nil, true, e
	}
	if !identityAccountInGroup(a, *groupID) || a.Platform != group.Platform || !s.isAccountSchedulableForModelSelection(ctx, a, model) || !s.isModelSupportedByAccountWithContext(ctx, a, model) || !s.isAccountSchedulableForQuota(a) || !s.isAccountSchedulableForWindowCost(ctx, a, false) || !s.isAccountSchedulableForRPM(ctx, a, false) || !s.isGatewayAccountProfitEligible(ctx, a) || s.checkChannelPricingRestriction(ctx, groupID, model) || s.isStickyAccountUpstreamRestricted(ctx, groupID, a, model) {
		return nil, true, fmt.Errorf("identity target account is unavailable or no longer compatible")
	}
	slot, e := s.tryAcquireAccountSlot(ctx, a.ID, a.Concurrency)
	if e != nil {
		return nil, true, e
	}
	if !slot.Acquired {
		cfg := s.schedulingConfig()
		result, err := s.newSelectionResult(ctx, a, false, nil, &AccountWaitPlan{AccountID: a.ID, MaxConcurrency: a.Concurrency, Timeout: cfg.FallbackWaitTimeout, MaxWaiting: cfg.FallbackMaxWaiting})
		return result, true, err
	}
	result, e := s.newSelectionResult(ctx, a, true, slot.ReleaseFunc, nil)
	if e != nil {
		slot.ReleaseFunc()
	}
	return result, true, e
}
func (s *OpenAIGatewayService) selectOpenAIIdentityTarget(ctx context.Context, groupID *int64, model string, excluded map[int64]struct{}, transport OpenAIUpstreamTransport, capability OpenAIEndpointCapability, imageCapability OpenAIImagesCapability, compact bool, platform string) (*AccountSelectionResult, bool, error) {
	id, pinned, e := identityTargetAllowed(ctx, groupID, excluded)
	if !pinned || e != nil {
		return nil, pinned, e
	}
	a, e := s.accountRepo.GetByID(ctx, id)
	if e != nil {
		return nil, true, e
	}
	a = s.recheckSelectedOpenAIAccountFromDB(ctx, a, groupID, platform, model, compact, capability)
	scheduler := &defaultOpenAIAccountScheduler{service: s, stats: newOpenAIAccountRuntimeStats()}
	req := OpenAIAccountScheduleRequest{GroupID: groupID, Platform: platform, RequestedModel: model, RequiredTransport: transport, RequiredCapability: capability, RequiredImageCapability: imageCapability, RequireCompact: compact}
	if a == nil || !identityAccountInGroup(a, *groupID) || !a.IsSchedulableForModelWithContext(ctx, model) || !scheduler.isAccountTransportCompatible(a, transport) || !scheduler.isAccountRequestCompatible(ctx, a, req) || s.checkChannelPricingRestriction(ctx, groupID, model) {
		return nil, true, fmt.Errorf("identity target account is unavailable or no longer compatible")
	}
	slot, e := s.tryAcquireAccountSlot(ctx, a.ID, a.Concurrency)
	if e != nil {
		return nil, true, e
	}
	if !slot.Acquired {
		cfg := s.schedulingConfig()
		return attachSelectionProfitGate(ctx, &AccountSelectionResult{Account: a, WaitPlan: &AccountWaitPlan{AccountID: a.ID, MaxConcurrency: a.Concurrency, Timeout: cfg.FallbackWaitTimeout, MaxWaiting: cfg.FallbackMaxWaiting}}), true, nil
	}
	return attachSelectionProfitGate(ctx, &AccountSelectionResult{Account: a, Acquired: true, ReleaseFunc: slot.ReleaseFunc}), true, nil
}
