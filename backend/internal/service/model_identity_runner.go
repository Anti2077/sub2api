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
	if svc.engineURL != "" || svc.remoteURL != "" {
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
	if r.svc.remoteURL != "" {
		r.executeRemote(ctx, run, finish)
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
	client := identityRunHTTPClient()
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
	defer func() { _ = res.Body.Close() }()
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

func (r *ModelIdentityRunner) executeRemote(ctx context.Context, run *IdentityRun, finish func(string, any)) {
	key, err := r.svc.ValidateRun(ctx, run)
	if err != nil {
		finish("configuration_error", map[string]string{"error": err.Error()})
		return
	}
	public, err := r.svc.publicBaseURL(ctx)
	if err != nil {
		finish("configuration_error", map[string]string{"error": err.Error()})
		return
	}
	baseURL := public + "/internal/model-identity/" + fmtIdentityID(run.ID) + "/remote/v1"
	body, _ := json.Marshal(map[string]any{
		"baseUrl": baseURL, "apiKey": key.Key, "modelId": run.RequestModel,
		"claimedModel": run.ExpectedModel, "runContextCheck": false, "biasFingerprint": true, "sync": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.svc.remoteURL, bytes.NewReader(body))
	if err != nil {
		finish("service_error", map[string]string{"error": "invalid BazaarLink Probe API URL"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := identityRunHTTPClient()
	res, err := client.Do(req)
	if err != nil {
		status := "service_error"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			status = "timed_out"
		} else if errors.Is(ctx.Err(), context.Canceled) {
			status = "cancelled"
		}
		finish(status, map[string]string{"error": "BazaarLink Probe API request interrupted"})
		return
	}
	defer func() { _ = res.Body.Close() }()
	var report map[string]any
	decodeErr := json.NewDecoder(io.LimitReader(res.Body, 16<<20)).Decode(&report)
	if decodeErr != nil || res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		finish("service_error", map[string]any{"error": "BazaarLink Probe API returned an invalid response", "status": res.StatusCode})
		return
	}
	redactRemoteSecrets(report)
	encoded, _ := json.Marshal(report)
	encoded = bytes.ReplaceAll(encoded, []byte(key.Key), []byte("[redacted]"))
	_ = json.Unmarshal(encoded, &report)
	report["engine_commit"] = "bazaarlink-online"
	report["engine_version"] = "Probe API"
	status := remoteRunStatus(report)
	assessment, _ := report["identityAssessment"].(map[string]any)
	verdict, _ := assessment["verdict"].(map[string]any)
	detected := remoteDetectedModel(assessment, verdict)
	report["detected_model"] = detected
	verdictStatus := verdict["status"]
	if verdictStatus == nil {
		verdictStatus = assessment["status"]
	}
	report["verdict"] = remoteIdentityVerdict(verdictStatus, detected, run.ExpectedModel)
	report["expected_model"] = run.ExpectedModel
	if items, ok := report["items"].([]any); ok {
		for _, item := range items {
			if entry, ok := item.(map[string]any); ok {
				entry["target_account_id"] = run.AccountID
				entry["request_model"] = run.RequestModel
				if entry["status"] == "error" && status == "completed" {
					status = "request_error"
				}
			}
		}
	}
	current, err := r.svc.repo.Run(ctx, run.ID)
	if err != nil || current == nil {
		finish("service_error", map[string]string{"error": "could not verify local probe evidence"})
		return
	}
	{
		var probes []struct {
			Status             int  `json:"status"`
			ConfigurationError bool `json:"configuration_error"`
		}
		if json.Unmarshal(current.Probes, &probes) == nil {
			for _, probe := range probes {
				if probe.ConfigurationError {
					status = "configuration_error"
					break
				}
				if probe.Status >= 400 && status == "completed" {
					status = "request_error"
				}
			}
		}
	}
	if status != "completed" {
		report["verdict"] = "inconclusive"
	}
	finish(status, report)
}

func identityRunHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error {
		// Run bodies contain a credential; never resend them to a redirect target.
		return http.ErrUseLastResponse
	}}
}

func remoteRunStatus(report map[string]any) string {
	switch report["status"] {
	case "completed":
		return "completed"
	case "aborted", "cancelled":
		return "cancelled"
	case "failed":
		return "request_error"
	default:
		return "service_error"
	}
}

func remoteVerdict(value any) string {
	switch value {
	case "match", "clean_match":
		return "matched"
	case "mismatch", "clean_match_submodel_mismatch", "plain_mismatch", "spoof_behavior_induced", "spoof_selfclaim_forged":
		return "mismatched"
	default:
		return "inconclusive"
	}
}

func remoteDetectedModel(assessment, verdict map[string]any) string {
	// BazaarLink's trueModel is a display label in some reports (for example,
	// "GPT 6 Astra"), while resolvedIdentity.modelId is the canonical ID used
	// by the catalog ("openai/gpt-6-astra"). Prefer the canonical field so the
	// local exact-model comparison does not reject a valid matching report.
	if resolved, ok := assessment["resolvedIdentity"].(map[string]any); ok {
		if model, ok := resolved["modelId"].(string); ok && strings.TrimSpace(model) != "" {
			return model
		}
	}
	if model, ok := verdict["trueModel"].(string); ok && strings.TrimSpace(model) != "" {
		return model
	}
	return ""
}

func normalizeRemoteModelID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if i := strings.LastIndexByte(value, '/'); i >= 0 {
		return normalizeRemoteModelToken(value[:i]) + "/" + normalizeRemoteModelToken(value[i+1:])
	}
	return normalizeRemoteModelToken(value)
}

func normalizeRemoteModelToken(value string) string {
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			_, _ = b.WriteRune(r)
		}
	}
	return b.String()
}

func remoteModelIDsMatch(expected, detected string) bool {
	expected = strings.TrimSpace(expected)
	detected = strings.TrimSpace(detected)
	if expected == "" || detected == "" {
		return false
	}
	if strings.EqualFold(expected, detected) {
		return true
	}
	// The API may return a human display name instead of a catalog ID. Compare
	// only the model portion after removing punctuation and the provider prefix;
	// this still keeps family-only values such as "openai" from matching.
	normalizedExpected, normalizedDetected := normalizeRemoteModelID(expected), normalizeRemoteModelID(detected)
	if normalizedExpected == "" || normalizedDetected == "" {
		return false
	}
	if strings.Contains(normalizedExpected, "/") && strings.Contains(normalizedDetected, "/") {
		return normalizedExpected == normalizedDetected
	}
	if strings.Contains(normalizedExpected, "/") {
		return strings.TrimPrefix(normalizedExpected, normalizedExpected[:strings.LastIndexByte(normalizedExpected, '/')+1]) == normalizedDetected
	}
	if strings.Contains(normalizedDetected, "/") {
		return strings.TrimPrefix(normalizedDetected, normalizedDetected[:strings.LastIndexByte(normalizedDetected, '/')+1]) == normalizedExpected
	}
	return normalizedExpected == normalizedDetected
}

func remoteIdentityVerdict(status any, detected, expected string) string {
	// Family evidence and off-baseline guesses cannot establish an exact model.
	if detected == "" {
		return "inconclusive"
	}
	if remoteVerdict(status) == "matched" {
		if remoteModelIDsMatch(expected, detected) {
			return "matched"
		}
		return "inconclusive"
	}
	if remoteVerdict(status) == "mismatched" && !remoteModelIDsMatch(expected, detected) {
		return "mismatched"
	}
	return "inconclusive"
}

func redactRemoteSecrets(value any) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			lower := strings.NewReplacer("-", "", "_", "").Replace(strings.ToLower(key))
			if lower == "apikey" || lower == "xapikey" || lower == "xgoogapikey" || lower == "userkey" || lower == "authorization" || lower == "proxyauthorization" || lower == "token" || lower == "accesstoken" || lower == "refreshtoken" || lower == "cookie" || lower == "setcookie" || lower == "startedbyip" {
				v[key] = "[redacted]"
				continue
			}
			redactRemoteSecrets(child)
		}
	case []any:
		for _, child := range v {
			redactRemoteSecrets(child)
		}
	}
}
