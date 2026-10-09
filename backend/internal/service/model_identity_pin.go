package service

import (
	"context"
	"fmt"
)

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
		return nil, true, fmt.Errorf("identity target account concurrency limit reached")
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
		return nil, true, fmt.Errorf("identity target account concurrency limit reached")
	}
	return attachSelectionProfitGate(ctx, &AccountSelectionResult{Account: a, Acquired: true, ReleaseFunc: slot.ReleaseFunc}), true, nil
}
