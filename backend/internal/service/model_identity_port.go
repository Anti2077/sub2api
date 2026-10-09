package service

import (
	"context"
	"encoding/json"
	"time"
)

const IdentityEngineCommit = "5c41136741ca52b5637879cca7bd0cae07404646"

type IdentityConfig struct {
	AccountID int64  `json:"account_id"`
	UserID    int64  `json:"user_id"`
	GroupID   int64  `json:"group_id"`
	APIKeyID  int64  `json:"api_key_id"`
	KeyName   string `json:"key_name"`
}
type IdentityPlan struct {
	ID              int64      `json:"id"`
	AccountID       int64      `json:"account_id"`
	RequestModel    string     `json:"request_model"`
	ExpectedModel   string     `json:"expected_model"`
	IntervalMinutes int        `json:"interval_minutes"`
	Enabled         bool       `json:"enabled"`
	LastRunAt       *time.Time `json:"last_run_at"`
	NextRunAt       *time.Time `json:"next_run_at"`
}
type IdentityRun struct {
	ID            int64           `json:"id"`
	PlanID        int64           `json:"plan_id"`
	AccountID     int64           `json:"account_id"`
	UserID        int64           `json:"user_id"`
	GroupID       int64           `json:"group_id"`
	APIKeyID      int64           `json:"api_key_id"`
	RequestModel  string          `json:"request_model"`
	ExpectedModel string          `json:"expected_model"`
	Status        string          `json:"status"`
	StartedAt     *time.Time      `json:"started_at"`
	FinishedAt    *time.Time      `json:"finished_at"`
	CreatedAt     time.Time       `json:"created_at"`
	Report        json.RawMessage `json:"report"`
	Probes        json.RawMessage `json:"probes"`
	ProbeCount    int             `json:"probe_count"`
}
type IdentityModel struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Family string `json:"family"`
}
type IdentityRepository interface {
	Config(context.Context, int64) (*IdentityConfig, error)
	Configure(context.Context, *IdentityConfig, string, string) error
	Plans(context.Context, int64) ([]IdentityPlan, error)
	Plan(context.Context, int64) (*IdentityPlan, error)
	SavePlan(context.Context, *IdentityPlan, time.Time) error
	DeletePlan(context.Context, int64) error
	Enqueue(context.Context, int64) (*IdentityRun, error)
	ScanDue(context.Context, time.Time) error
	Claim(context.Context, string, time.Time) (*IdentityRun, error)
	Authorize(context.Context, int64, string, time.Time) (*IdentityRun, error)
	Finish(context.Context, int64, string, json.RawMessage) error
	Cancel(context.Context, int64) error
	Run(context.Context, int64) (*IdentityRun, error)
	History(context.Context, int64) ([]IdentityRun, error)
	AppendProbe(context.Context, int64, json.RawMessage) error
	ProbeSlot(context.Context, int64, bool) error
}

// AdvanceIdentitySchedule preserves the original cadence and skips missed slots.
func AdvanceIdentitySchedule(due, now time.Time, interval time.Duration) time.Time {
	if due.After(now) {
		return due
	}
	return due.Add((now.Sub(due)/interval + 1) * interval)
}
