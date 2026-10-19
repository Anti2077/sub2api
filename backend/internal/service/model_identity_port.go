package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const IdentityEngineCommit = "654c8334bb53951aaf04cbd61252d163649e1369"

const (
	DefaultIdentityConcurrentAccounts = 1
	DefaultIdentityProbeConcurrency   = 4
	MaxIdentityConcurrentAccounts     = 2
	MaxIdentityProbeConcurrency       = 10
	IdentityProbeConcurrency          = DefaultIdentityProbeConcurrency
)

var ErrIdentityProbeSlotsFull = errors.New("identity probe slots full")

type IdentityConfig struct {
	AccountID          int64  `json:"account_id"`
	UserID             int64  `json:"user_id"`
	GroupID            int64  `json:"group_id"`
	APIKeyID           int64  `json:"api_key_id"`
	KeyName            string `json:"key_name"`
	ConfigurationError string `json:"configuration_error,omitempty"`
}
type IdentityPlan struct {
	ID              int64                `json:"id"`
	AccountID       int64                `json:"account_id"`
	RequestModel    string               `json:"request_model"`
	ExpectedModel   string               `json:"expected_model"`
	IntervalMinutes int                  `json:"interval_minutes"`
	Enabled         bool                 `json:"enabled"`
	Options         *IdentityPlanOptions `json:"options,omitempty"`
	LastRunAt       *time.Time           `json:"last_run_at"`
	NextRunAt       *time.Time           `json:"next_run_at"`
}
type IdentityPlanOptions struct {
	Protocol        string `json:"protocol"`
	ReasoningEffort string `json:"reasoning_effort"`
	ServiceTier     string `json:"service_tier"`
	Stream          bool   `json:"stream"`
	Parallel        bool   `json:"parallel"`
	Relaxed         bool   `json:"relaxed"`
	Tokenizer       bool   `json:"tokenizer"`
}

func DefaultIdentityPlanOptions() IdentityPlanOptions {
	return IdentityPlanOptions{Protocol: "chat_completions", Stream: true, Parallel: true, Relaxed: true}
}

type IdentityRun struct {
	ID               int64               `json:"id"`
	PlanID           int64               `json:"plan_id"`
	AccountID        int64               `json:"account_id"`
	UserID           int64               `json:"user_id"`
	GroupID          int64               `json:"group_id"`
	APIKeyID         int64               `json:"api_key_id"`
	RequestModel     string              `json:"request_model"`
	ExpectedModel    string              `json:"expected_model"`
	Status           string              `json:"status"`
	StartedAt        *time.Time          `json:"started_at"`
	FinishedAt       *time.Time          `json:"finished_at"`
	CreatedAt        time.Time           `json:"created_at"`
	Report           json.RawMessage     `json:"report"`
	Probes           json.RawMessage     `json:"probes"`
	ProbeCount       int                 `json:"probe_count"`
	ProbeConcurrency int                 `json:"probe_concurrency"`
	Options          IdentityPlanOptions `json:"options"`
	EngineCommit     string              `json:"engine_commit,omitempty"`
}
type IdentityPlanCard struct {
	Plan        IdentityPlan   `json:"plan"`
	AccountName string         `json:"account_name"`
	GroupName   string         `json:"group_name"`
	Config      IdentityConfig `json:"config"`
	RecentRuns  []IdentityRun  `json:"recent_runs"`
}
type IdentityPlanPage struct {
	Items []IdentityPlanCard `json:"items"`
	Total int                `json:"total"`
}
type IdentityModel struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Family string `json:"family"`
}
type IdentityAccountPage struct {
	AccountIDs []int64 `json:"account_ids"`
	Total      int     `json:"total"`
}
type IdentityRepository interface {
	SaveTestUserSettings(context.Context, int64, string) ([]string, error)
	Overview(context.Context, int, int, string) (*IdentityPlanPage, error)
	Progress(context.Context, int64, json.RawMessage) error
	PlannedAccounts(context.Context, int, int, string) (*IdentityAccountPage, error)
	Config(context.Context, int64) (*IdentityConfig, error)
	Configure(context.Context, *IdentityConfig, string, string) error
	Plans(context.Context, int64) ([]IdentityPlan, error)
	Plan(context.Context, int64) (*IdentityPlan, error)
	SavePlan(context.Context, *IdentityPlan, time.Time) error
	DeletePlan(context.Context, int64) error
	Enqueue(context.Context, int64) (*IdentityRun, error)
	ScanDue(context.Context, time.Time) error
	Claim(context.Context, string, time.Time, int, int) (*IdentityRun, error)
	Authorize(context.Context, int64, string, time.Time) (*IdentityRun, error)
	SetRemoteRunID(context.Context, int64, string) error
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
