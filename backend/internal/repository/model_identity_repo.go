package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type identityRepository struct{ db *sql.DB }

func NewIdentityRepository(db *sql.DB) service.IdentityRepository { return &identityRepository{db: db} }

func (r *identityRepository) Config(ctx context.Context, id int64) (*service.IdentityConfig, error) {
	c := &service.IdentityConfig{}
	err := r.db.QueryRowContext(ctx, `SELECT c.account_id,c.user_id,c.group_id,c.api_key_id,k.name FROM account_identity_configs c JOIN api_keys k ON k.id=c.api_key_id WHERE c.account_id=$1`, id).Scan(&c.AccountID, &c.UserID, &c.GroupID, &c.APIKeyID, &c.KeyName)
	return c, err
}
func (r *identityRepository) Configure(ctx context.Context, c *service.IdentityConfig, name, token string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// Serialize provisioning across replicas; Key and binding commit together.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(244, $1::integer)`, c.AccountID); err != nil {
		return err
	}
	var active bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM account_identity_runs WHERE account_id=$1 AND status IN ('queued','running','cancelling'))`, c.AccountID).Scan(&active); err != nil {
		return err
	}
	if active {
		return errors.New("cancel or finish active detections before changing account configuration")
	}
	var keyID int64
	err = tx.QueryRowContext(ctx, `SELECT api_key_id FROM account_identity_configs WHERE account_id=$1 FOR UPDATE`, c.AccountID).Scan(&keyID)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `INSERT INTO api_keys(user_id,group_id,key,name,status,created_at,updated_at) VALUES($1,$2,$3,$4,'active',NOW(),NOW()) RETURNING id`, c.UserID, c.GroupID, token, name).Scan(&keyID)
	} else if err == nil {
		var status string
		var deleted, expiry *time.Time
		var quota, used float64
		err = tx.QueryRowContext(ctx, `SELECT status,deleted_at,expires_at,quota,quota_used FROM api_keys WHERE id=$1 FOR UPDATE`, keyID).Scan(&status, &deleted, &expiry, &quota, &used)
		if err == nil && (deleted != nil || status != "active" || (expiry != nil && !expiry.After(time.Now())) || (quota > 0 && used >= quota)) {
			return fmt.Errorf("configuration error: repair the existing dedicated API Key #%d before saving", keyID)
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE api_keys SET user_id=$2,group_id=$3,name=$4,updated_at=NOW() WHERE id=$1`, keyID, c.UserID, c.GroupID, name)
		}
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO account_identity_configs(account_id,user_id,group_id,api_key_id) VALUES($1,$2,$3,$4) ON CONFLICT(account_id) DO UPDATE SET user_id=EXCLUDED.user_id,group_id=EXCLUDED.group_id,updated_at=NOW()`, c.AccountID, c.UserID, c.GroupID, keyID)
	if err != nil {
		return err
	}
	c.APIKeyID = keyID
	c.KeyName = name
	return tx.Commit()
}

const identityPlanColumns = `id,account_id,request_model,expected_model,interval_minutes,enabled,last_run_at,next_run_at`

func scanIdentityPlan(row scannable) (*service.IdentityPlan, error) {
	p := &service.IdentityPlan{}
	err := row.Scan(&p.ID, &p.AccountID, &p.RequestModel, &p.ExpectedModel, &p.IntervalMinutes, &p.Enabled, &p.LastRunAt, &p.NextRunAt)
	return p, err
}
func (r *identityRepository) Plans(ctx context.Context, id int64) ([]service.IdentityPlan, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+identityPlanColumns+` FROM account_identity_plans WHERE account_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.IdentityPlan{}
	for rows.Next() {
		p, e := scanIdentityPlan(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}
func (r *identityRepository) Plan(ctx context.Context, id int64) (*service.IdentityPlan, error) {
	return scanIdentityPlan(r.db.QueryRowContext(ctx, `SELECT `+identityPlanColumns+` FROM account_identity_plans WHERE id=$1`, id))
}
func (r *identityRepository) SavePlan(ctx context.Context, p *service.IdentityPlan, now time.Time) error {
	if p.ID == 0 {
		if p.Enabled {
			next := now.Add(time.Duration(p.IntervalMinutes) * time.Minute)
			p.NextRunAt = &next
		}
		return r.db.QueryRowContext(ctx, `INSERT INTO account_identity_plans(account_id,request_model,expected_model,interval_minutes,enabled,next_run_at) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, p.AccountID, p.RequestModel, p.ExpectedModel, p.IntervalMinutes, p.Enabled, p.NextRunAt).Scan(&p.ID)
	}
	// Keep cadence for model edits; reset only when enabling or changing interval.
	_, err := r.db.ExecContext(ctx, `UPDATE account_identity_plans SET request_model=$2,expected_model=$3,next_run_at=CASE WHEN NOT $5 THEN NULL WHEN NOT enabled OR interval_minutes<>$4 THEN $6::timestamptz+($4*INTERVAL '1 minute') ELSE next_run_at END,interval_minutes=$4,enabled=$5 WHERE id=$1`, p.ID, p.RequestModel, p.ExpectedModel, p.IntervalMinutes, p.Enabled, now)
	return err
}
func (r *identityRepository) DeletePlan(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM account_identity_plans p WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM account_identity_runs r WHERE r.plan_id=p.id AND r.status IN ('queued','running','cancelling'))`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return errors.New("plan missing or still running; cancel it first")
	}
	return nil
}

const enqueueIdentitySQL = `INSERT INTO account_identity_runs(plan_id,account_id,user_id,group_id,api_key_id,request_model,expected_model) SELECT p.id,p.account_id,c.user_id,c.group_id,c.api_key_id,p.request_model,p.expected_model FROM account_identity_plans p JOIN account_identity_configs c ON c.account_id=p.account_id WHERE p.id=$1 ON CONFLICT DO NOTHING RETURNING id`

func (r *identityRepository) Enqueue(ctx context.Context, id int64) (*service.IdentityRun, error) {
	var runID int64
	err := r.db.QueryRowContext(ctx, enqueueIdentitySQL, id).Scan(&runID)
	if err != nil {
		return nil, errors.New("account already has an active detection or plan does not exist")
	}
	return r.Run(ctx, runID)
}
func (r *identityRepository) ScanDue(ctx context.Context, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `UPDATE account_identity_runs SET status=CASE WHEN status='cancelling' THEN 'cancelled' ELSE 'timed_out' END,finished_at=$1,token_hash=NULL,worker_slot=NULL WHERE status IN ('running','cancelling') AND lease_until<=$1`, now)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, pruneIdentityReportsSQL); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,next_run_at,interval_minutes FROM account_identity_plans WHERE enabled AND next_run_at<=$1 ORDER BY next_run_at FOR UPDATE SKIP LOCKED LIMIT 100`, now)
	if err != nil {
		return err
	}
	type due struct {
		id      int64
		at      time.Time
		minutes int
	}
	var ds []due
	for rows.Next() {
		var d due
		if err = rows.Scan(&d.id, &d.at, &d.minutes); err != nil {
			_ = rows.Close()
			return err
		}
		ds = append(ds, d)
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	for _, d := range ds {
		var id int64
		err = tx.QueryRowContext(ctx, enqueueIdentitySQL, d.id).Scan(&id)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE account_identity_plans SET next_run_at=$2 WHERE id=$1`, d.id, service.AdvanceIdentitySchedule(d.at, now, time.Duration(d.minutes)*time.Minute))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (r *identityRepository) Claim(ctx context.Context, hash string, now time.Time) (*service.IdentityRun, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	// A single short DB lock makes the global two-slot claim atomic across instances.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(244,0)`); err != nil {
		return nil, err
	}
	var slot int
	err = tx.QueryRowContext(ctx, `SELECT s FROM generate_series(1,2) s WHERE NOT EXISTS(SELECT 1 FROM account_identity_runs WHERE worker_slot=s AND status IN ('running','cancelling')) LIMIT 1`).Scan(&slot)
	if err != nil {
		return nil, err
	}
	var id, planID int64
	err = tx.QueryRowContext(ctx, `UPDATE account_identity_runs SET status='running',started_at=$1::timestamptz,lease_until=$1::timestamptz+INTERVAL '20 minutes',token_hash=$2,worker_slot=$3 WHERE id=(SELECT q.id FROM account_identity_runs q WHERE q.status='queued' AND NOT EXISTS(SELECT 1 FROM account_identity_runs a WHERE a.account_id=q.account_id AND a.status IN ('running','cancelling')) ORDER BY q.id FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING id,plan_id`, now, hash, slot).Scan(&id, &planID)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE account_identity_plans SET last_run_at=$2 WHERE id=$1`, planID, now); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Run(ctx, id)
}

const identityRunColumns = `id,plan_id,account_id,user_id,group_id,api_key_id,request_model,expected_model,status,started_at,finished_at,created_at,report,probes`

// Include cancellations and expired leases, which may never reach Finish.
const pruneIdentityReportsSQL = `DELETE FROM account_identity_runs WHERE id IN (
 SELECT id FROM (SELECT id,ROW_NUMBER() OVER (PARTITION BY plan_id ORDER BY id DESC) AS position
 FROM account_identity_runs WHERE status NOT IN ('queued','running','cancelling')) history WHERE position>50
)`

func scanIdentityRun(row scannable) (*service.IdentityRun, error) {
	r := &service.IdentityRun{}
	err := row.Scan(&r.ID, &r.PlanID, &r.AccountID, &r.UserID, &r.GroupID, &r.APIKeyID, &r.RequestModel, &r.ExpectedModel, &r.Status, &r.StartedAt, &r.FinishedAt, &r.CreatedAt, &r.Report, &r.Probes)
	if err == nil {
		var probes []json.RawMessage
		if json.Unmarshal(r.Probes, &probes) == nil {
			r.ProbeCount = len(probes)
		}
	}
	return r, err
}
func (r *identityRepository) Run(ctx context.Context, id int64) (*service.IdentityRun, error) {
	return scanIdentityRun(r.db.QueryRowContext(ctx, `SELECT `+identityRunColumns+` FROM account_identity_runs WHERE id=$1`, id))
}
func (r *identityRepository) Authorize(ctx context.Context, id int64, hash string, now time.Time) (*service.IdentityRun, error) {
	return scanIdentityRun(r.db.QueryRowContext(ctx, `SELECT id,plan_id,account_id,user_id,group_id,api_key_id,request_model,expected_model,status,started_at,finished_at,created_at,'{}'::jsonb,'[]'::jsonb FROM account_identity_runs WHERE id=$1 AND token_hash=$2 AND status='running' AND lease_until>$3`, id, hash, now))
}
func (r *identityRepository) Finish(ctx context.Context, id int64, status string, report json.RawMessage) error {
	_, err := r.db.ExecContext(ctx, `UPDATE account_identity_runs SET status=CASE WHEN status='cancelling' THEN 'cancelled' ELSE $2 END,report=CASE WHEN status='cancelling' THEN jsonb_set($3::jsonb,'{verdict}','"inconclusive"'::jsonb) ELSE $3::jsonb END,finished_at=NOW(),token_hash=NULL,worker_slot=NULL WHERE id=$1 AND status IN ('running','cancelling')`, id, status, report)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM account_identity_runs WHERE id IN(SELECT id FROM account_identity_runs WHERE plan_id=(SELECT plan_id FROM account_identity_runs WHERE id=$1) AND status NOT IN ('queued','running','cancelling') ORDER BY id DESC OFFSET 50)`, id)
	return err
}
func (r *identityRepository) Cancel(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE account_identity_runs SET status=CASE WHEN status='queued' THEN 'cancelled' ELSE 'cancelling' END,finished_at=CASE WHEN status='queued' THEN NOW() ELSE NULL END,token_hash=NULL WHERE id=$1 AND status IN ('queued','running')`, id)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, pruneIdentityReportsSQL)
	return err
}
func (r *identityRepository) History(ctx context.Context, id int64) ([]service.IdentityRun, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,plan_id,account_id,user_id,group_id,api_key_id,request_model,expected_model,status,started_at,finished_at,created_at,jsonb_build_object('verdict',report->'verdict','detected_model',report->'detected_model','expected_model',report->'expected_model'),'[]'::jsonb,jsonb_array_length(probes) FROM account_identity_runs WHERE plan_id=$1 ORDER BY id DESC LIMIT 50`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.IdentityRun{}
	for rows.Next() {
		run := &service.IdentityRun{}
		e := rows.Scan(&run.ID, &run.PlanID, &run.AccountID, &run.UserID, &run.GroupID, &run.APIKeyID, &run.RequestModel, &run.ExpectedModel, &run.Status, &run.StartedAt, &run.FinishedAt, &run.CreatedAt, &run.Report, &run.Probes, &run.ProbeCount)
		if e != nil {
			return nil, e
		}
		out = append(out, *run)
	}
	return out, rows.Err()
}
func (r *identityRepository) AppendProbe(ctx context.Context, id int64, probe json.RawMessage) error {
	_, err := r.db.ExecContext(ctx, `UPDATE account_identity_runs SET probes=probes || jsonb_build_array($2::jsonb) WHERE id=$1 AND jsonb_array_length(probes)<500`, id, probe)
	return err
}
func (r *identityRepository) ProbeSlot(ctx context.Context, id int64, acquire bool) error {
	query := `UPDATE account_identity_runs SET probe_inflight=GREATEST(0,probe_inflight-1) WHERE id=$1`
	if acquire {
		query = `UPDATE account_identity_runs SET probe_inflight=probe_inflight+1 WHERE id=$1 AND status='running' AND lease_until>NOW() AND probe_inflight<4 AND jsonb_array_length(probes)+probe_inflight<500`
	}
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return errors.New("probe concurrency limit reached or run inactive")
	}
	return nil
}
