package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type ModelIdentityRunner struct {
	svc    *ModelIdentityService
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func ProvideModelIdentityRunner(svc *ModelIdentityService) *ModelIdentityRunner {
	ctx, cancel := context.WithCancel(context.Background())
	r := &ModelIdentityRunner{svc: svc, ctx: ctx, cancel: cancel}
	if svc.engineURL != "" {
		r.wg.Add(1)
		go r.loop()
	}
	return r
}
func (r *ModelIdentityRunner) Stop() { r.cancel(); r.wg.Wait() }
func (r *ModelIdentityRunner) loop() {
	defer r.wg.Done()
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	r.tick()
	dispatch := time.NewTicker(2 * time.Second)
	defer dispatch.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-tick.C:
			r.tick()
		case <-dispatch.C:
			r.dispatch()
		}
	}
}
func (r *ModelIdentityRunner) tick() {
	_ = r.svc.repo.ScanDue(r.ctx, time.Now())
	r.dispatch()
}
func (r *ModelIdentityRunner) dispatch() {
	for i := 0; i < 2; i++ {
		token, e := newIdentityToken()
		if e != nil {
			return
		}
		run, e := r.svc.repo.Claim(r.ctx, HashIdentityToken(token), time.Now())
		if e != nil {
			return
		}
		r.wg.Add(1)
		go func() { defer r.wg.Done(); r.execute(run, token) }()
	}
}
func (r *ModelIdentityRunner) execute(run *IdentityRun, token string) {
	deadline := time.Now().Add(20 * time.Minute)
	if run.StartedAt != nil {
		deadline = run.StartedAt.Add(20 * time.Minute)
	}
	ctx, cancel := context.WithDeadline(r.ctx, deadline)
	defer cancel()
	done := make(chan struct{})
	defer close(done)
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-tick.C:
				if _, e := r.svc.Authorize(ctx, run.ID, token); e != nil {
					cancel()
					return
				}
			}
		}
	}()
	finish := func(status string, report any) {
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer saveCancel()
		_ = r.svc.Finish(saveCtx, run.ID, status, report)
	}
	if _, e := r.svc.ValidateRun(ctx, run); e != nil {
		finish("configuration_error", map[string]string{"error": e.Error()})
		return
	}
	callback := strings.TrimRight(os.Getenv("MODEL_IDENTITY_CALLBACK_URL"), "/")
	if callback == "" {
		finish("service_error", map[string]string{"error": "MODEL_IDENTITY_CALLBACK_URL is not configured"})
		return
	}
	body, _ := json.Marshal(map[string]any{"run_id": run.ID, "model": run.RequestModel, "expected_model": run.ExpectedModel, "token": token, "probe_url": callback + "/internal/model-identity/" + fmtIdentityID(run.ID) + "/probe"})
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, r.svc.engineURL+"/run", bytes.NewReader(body))
	if e != nil {
		finish("service_error", map[string]string{"error": "invalid engine URL"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 20 * time.Minute}
	res, e := client.Do(req)
	if e != nil {
		status := "service_error"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			status = "timed_out"
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			status = "cancelled"
		}
		finish(status, map[string]string{"error": "identity engine request interrupted"})
		return
	}
	defer res.Body.Close()
	var report map[string]any
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(&report) != nil || report["engine_commit"] != IdentityEngineCommit {
		finish("service_error", map[string]string{"error": "invalid response or version from identity engine"})
		return
	}
	status := "completed"
	if evidence, ok := report["evidence"].(map[string]any); ok {
		if coverage, ok := evidence["coverage"].(map[string]any); ok {
			if errors, ok := coverage["errors"].(float64); ok && errors > 0 {
				status = "request_error"
			}
		}
	}
	if current, e := r.svc.repo.Run(ctx, run.ID); e == nil {
		var probes []struct {
			Status             int  `json:"status"`
			ConfigurationError bool `json:"configuration_error"`
		}
		if json.Unmarshal(current.Probes, &probes) == nil {
			for _, probe := range probes {
				if probe.Status == http.StatusConflict || probe.ConfigurationError {
					status = "configuration_error"
					break
				}
			}
		}
	}
	if status != "completed" {
		report["verdict"] = "inconclusive"
	}
	finish(status, report)
}
