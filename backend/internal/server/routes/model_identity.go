package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type identityResponseWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *identityResponseWriter) Write(b []byte) (int, error) {
	if w.body.Len()+len(b) <= 2<<20 {
		_, _ = w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}
func (w *identityResponseWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func registerIdentityProbe(r *gin.Engine, h *handler.Handlers, svc *service.ModelIdentityService, auth middleware.APIKeyAuthMiddleware, requireGroup gin.HandlerFunc, opsLogger gin.HandlerFunc) {
	r.GET("/internal/model-identity/:run/capability", func(c *gin.Context) {
		id, e := strconv.ParseInt(c.Param("run"), 10, 64)
		if e != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		run, e := svc.Authorize(c.Request.Context(), id, strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if e != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.JSON(http.StatusOK, gin.H{"request_model": run.RequestModel, "expected_model": run.ExpectedModel})
	})
	r.POST("/internal/model-identity/:run/probe", middleware.RequestBodyLimit(64<<10), identityProbeDeadline(), func(c *gin.Context) {
		id, e := strconv.ParseInt(c.Param("run"), 10, 64)
		if e != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		run, e := svc.Authorize(c.Request.Context(), id, token)
		if e != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !waitIdentityProbeSlot(c, svc, id) {
			return
		}
		defer func() {
			ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			_ = svc.ProbeSlot(ctx, id, false)
		}()
		if _, e = svc.Authorize(c.Request.Context(), id, token); e != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		key, e := svc.ValidateRun(c.Request.Context(), run)
		if e != nil {
			payload, _ := json.Marshal(map[string]any{"status": 409, "error": e.Error()})
			_ = svc.AppendProbe(c.Request.Context(), id, payload)
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": gin.H{"message": e.Error()}})
			return
		}
		var input struct {
			ProbeID   string `json:"probe_id"`
			Prompt    string `json:"prompt"`
			MaxTokens int    `json:"max_tokens"`
		}
		decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil || input.Prompt == "" || len(input.Prompt) > 32000 || len(input.ProbeID) > 128 {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if input.MaxTokens < 1 {
			input.MaxTokens = 1024
		}
		if input.MaxTokens > 4096 {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		body, _ := json.Marshal(map[string]any{"model": run.RequestModel, "messages": []map[string]string{{"role": "user", "content": input.Prompt}}, "stream": false, "max_tokens": input.MaxTokens})
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Request.ContentLength = int64(len(body))
		c.Request.Header = make(http.Header)
		c.Request.Header.Set("Authorization", "Bearer "+key.Key)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Header.Set("User-Agent", "Sub2API-ModelIdentity/1")
		requestCtx, cancel := context.WithTimeout(service.WithIdentityTarget(c.Request.Context(), run.AccountID, run.GroupID), 180*time.Second)
		defer cancel()
		done := make(chan struct{})
		defer close(done)
		evidence := &service.IdentityProbeEvidence{AccountID: run.AccountID}
		requestCtx = service.WithIdentityEvidence(requestCtx, evidence)
		c.Request = c.Request.WithContext(requestCtx)
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-requestCtx.Done():
					return
				case <-ticker.C:
					if _, err := svc.Authorize(requestCtx, id, token); err != nil {
						cancel()
						return
					}
				}
			}
		}()
		writer := &identityResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()
		probe := map[string]any{"probe_id": input.ProbeID, "prompt": input.Prompt, "request_model": run.RequestModel, "target_account_id": run.AccountID, "client_request_id": writer.Header().Get("X-Client-Request-ID"), "status": writer.Status(), "evidence": evidence, "response": json.RawMessage(writer.body.Bytes())}
		if !json.Valid(writer.body.Bytes()) {
			probe["response"] = writer.body.String()
		}
		if writer.Status() >= 400 {
			probe["response"] = map[string]any{"error": "probe request failed", "status": writer.Status()}
			probe["error"] = "probe request failed"
			if writer.Status() == http.StatusUnauthorized || writer.Status() == http.StatusForbidden || writer.Status() == http.StatusPaymentRequired {
				probe["configuration_error"] = true
				probe["error"] = "configuration error: test user balance, subscription, group permission or dedicated Key rejected; repair the test configuration"
			}
		}
		encoded, _ := json.Marshal(probe)
		encoded = bytes.ReplaceAll(encoded, []byte(key.Key), []byte("[redacted]"))
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer saveCancel()
		_ = svc.AppendProbe(saveCtx, id, encoded)
	}, middleware.ClientRequestID(), opsLogger, handler.InboundEndpointMiddleware(), gin.HandlerFunc(auth), middleware.GroupModelAllowlist(), requireGroup, func(c *gin.Context) {
		switch getGroupPlatform(c) {
		case service.PlatformOpenAI, service.PlatformGrok, service.PlatformKimi, service.PlatformZhipu, service.PlatformDeepseek, service.PlatformMiniMax, service.PlatformOpenCodeGo:
			h.OpenAIGateway.ChatCompletions(c)
		default:
			h.Gateway.ChatCompletions(c)
		}
	})
	// This route is the only public identity callback. Keep capability/probe
	// routes above private. Its own guard runs before ordinary Key billing.
	r.POST("/internal/model-identity/:run/remote/v1/chat/completions", middleware.RequestBodyLimit(1<<20), identityProbeDeadline(),
		remoteIdentityProbe(svc), middleware.ClientRequestID(), opsLogger, handler.InboundEndpointMiddleware(), gin.HandlerFunc(auth),
		middleware.GroupModelAllowlist(), requireGroup, func(c *gin.Context) {
			switch getGroupPlatform(c) {
			case service.PlatformOpenAI, service.PlatformGrok, service.PlatformKimi, service.PlatformZhipu, service.PlatformDeepseek, service.PlatformMiniMax, service.PlatformOpenCodeGo:
				h.OpenAIGateway.ChatCompletions(c)
			default:
				h.Gateway.ChatCompletions(c)
			}
		})
}

func identityProbeDeadline() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 180*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func remoteIdentityProbe(svc *service.ModelIdentityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("run"), 10, 64)
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if err != nil || len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		credential := strings.TrimSpace(parts[1])
		run, key, err := svc.AuthorizeRemote(c.Request.Context(), id, credential)
		if err != nil {
			if run != nil && key != nil { // Only record errors for the authenticated run Key.
				payload, _ := json.Marshal(map[string]any{"status": 409, "configuration_error": true, "error": "repair the account test configuration"})
				_ = svc.AppendProbe(c.Request.Context(), id, payload)
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": gin.H{"message": "repair the account test configuration"}})
			} else {
				c.AbortWithStatus(http.StatusUnauthorized)
			}
			return
		}
		if !waitIdentityProbeSlot(c, svc, id) {
			return
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = svc.ProbeSlot(ctx, id, false)
		}()
		// Recheck the binding after waiting; queued requests cannot outlive cancellation.
		if _, _, err := svc.AuthorizeRemote(c.Request.Context(), id, credential); err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, (1<<20)+1))
		var input map[string]json.RawMessage
		var model string
		if err != nil || len(body) > 1<<20 || json.Unmarshal(body, &input) != nil || json.Unmarshal(input["model"], &model) != nil || model != run.RequestModel {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "probe must use the configured request model"}})
			return
		}
		for _, field := range []string{"max_tokens", "max_completion_tokens"} {
			if value, ok := input[field]; ok {
				var count int
				if json.Unmarshal(value, &count) != nil || count < 1 || count > 4096 {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "probe output limit must be 1 to 4096 tokens"}})
					return
				}
			}
		}
		if _, a := input["max_tokens"]; !a {
			if _, b := input["max_completion_tokens"]; !b {
				input["max_tokens"] = json.RawMessage("4096")
				body, _ = json.Marshal(input)
			}
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Request.ContentLength = int64(len(body))
		// Keep authentication and client fingerprint headers. Account pinning is
		// installed exclusively in the server context; client fields are ignored.
		c.Request.Header.Del("X-Account-ID")
		c.Request.Header.Del("X-Group-ID")
		evidence := &service.IdentityProbeEvidence{AccountID: run.AccountID}
		ctx, cancel := context.WithTimeout(service.WithIdentityEvidence(service.WithIdentityTarget(c.Request.Context(), run.AccountID, run.GroupID), evidence), 180*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		done := make(chan struct{})
		defer close(done)
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					if _, _, err := svc.AuthorizeRemote(ctx, id, credential); err != nil {
						cancel()
						return
					}
				}
			}
		}()
		writer := &identityResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()
		probe := map[string]any{"request_model": run.RequestModel, "target_account_id": run.AccountID, "client_request_id": writer.Header().Get("X-Client-Request-ID"), "status": writer.Status(), "evidence": evidence, "response": json.RawMessage(writer.body.Bytes())}
		if !json.Valid(writer.body.Bytes()) {
			probe["response"] = writer.body.String()
		}
		if writer.Status() >= 400 {
			probe["response"] = map[string]any{"error": "probe request failed", "status": writer.Status()}
			probe["error"] = "probe request failed"
			if writer.Status() == 401 || writer.Status() == 402 || writer.Status() == 403 || writer.Status() == 409 {
				probe["configuration_error"] = true
			}
		}
		encoded, _ := json.Marshal(probe)
		encoded = bytes.ReplaceAll(encoded, []byte(key.Key), []byte("[redacted]"))
		saveCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = svc.AppendProbe(saveCtx, id, encoded)
	}
}

func waitIdentityProbeSlot(c *gin.Context, svc *service.ModelIdentityService, id int64) bool {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if err := svc.WaitProbeSlot(ctx, id); err != nil {
		c.Header("Retry-After", "2")
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"type": "rate_limit_error", "message": "Detection probe queue unavailable; retry later"}})
		return false
	}
	return true
}
