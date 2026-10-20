package rewards

import (
	"context"
	"database/sql"
	"errors"
)

// GatewayReceipt retains the provider response, not permission to call it again.
// Pending receipts are fail-closed, including across a process crash.
type GatewayReceipt struct {
	Claimed                                    bool
	Fingerprint, RequestID, State, ContentType string
	Status                                     int
	Body                                       []byte
}

func (s *Store) ClaimGateway(ctx context.Context, key int64, id, fingerprint, requestID string) (GatewayReceipt, error) {
	r := GatewayReceipt{Fingerprint: fingerprint, RequestID: requestID, State: "pending"}
	var inserted int
	err := s.DB.QueryRowContext(ctx, `INSERT INTO reward_gateway_requests(api_key_id,idempotency_key,fingerprint,request_id) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING 1`, key, id, fingerprint, requestID).Scan(&inserted)
	if err == nil {
		r.Claimed = true
		return r, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return r, err
	}
	err = s.DB.QueryRowContext(ctx, `SELECT fingerprint,request_id,state,status,content_type,body FROM reward_gateway_requests WHERE api_key_id=$1 AND idempotency_key=$2`, key, id).Scan(&r.Fingerprint, &r.RequestID, &r.State, &r.Status, &r.ContentType, &r.Body)
	return r, err
}
func (s *Store) FinishGateway(ctx context.Context, key int64, id string, status int, contentType string, body []byte, overflow bool) error {
	state := "complete"
	if overflow {
		state = "unavailable"
		body = nil
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE reward_gateway_requests SET state=$3,status=$4,content_type=$5,body=$6,completed_at=NOW() WHERE api_key_id=$1 AND idempotency_key=$2 AND state='pending'`, key, id, state, status, contentType, body)
	return err
}
