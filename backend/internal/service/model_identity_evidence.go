package service

import (
	"context"
	"strconv"
)

func fmtIdentityID(id int64) string { return strconv.FormatInt(id, 10) }

type IdentityProbeEvidence struct {
	AccountID     int64  `json:"account_id"`
	UpstreamModel string `json:"upstream_model"`
	RequestID     string `json:"request_id"`
}
type identityEvidenceKey struct{}

func WithIdentityEvidence(ctx context.Context, e *IdentityProbeEvidence) context.Context {
	return context.WithValue(ctx, identityEvidenceKey{}, e)
}
func RecordIdentityEvidence(ctx context.Context, accountID int64, upstreamModel, requestID string) {
	if e, ok := ctx.Value(identityEvidenceKey{}).(*IdentityProbeEvidence); ok {
		e.AccountID = accountID
		e.UpstreamModel = upstreamModel
		e.RequestID = requestID
	}
}
