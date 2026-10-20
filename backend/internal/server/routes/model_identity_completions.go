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

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func registerIdentityCompletions(r *gin.Engine, h *handler.Handlers, svc *service.ModelIdentityService, auth middleware.APIKeyAuthMiddleware, requireGroup, opsLogger gin.HandlerFunc) {
	for _, protocol := range []string{"chat_completions", "responses", "messages"} {
		path := protocol
		if path == "chat_completions" {
			path = "chat/completions"
		}
		r.POST("/internal/model-identity/:run/v1/"+path, middleware.RequestBodyLimit(64<<10), identityProbeDeadline(), identityCompletionCapability(svc, protocol), middleware.ClientRequestID(), opsLogger, handler.InboundEndpointMiddleware(), gin.HandlerFunc(auth), middleware.GroupModelAllowlist(), requireGroup, func(c *gin.Context) {
			openai := domain.UsesOpenAIGateway(getGroupPlatform(c))
			switch protocol {
			case "responses":
				if openai {
					h.OpenAIGateway.Responses(c)
				} else {
					h.Gateway.Responses(c)
				}
			case "messages":
				if openai {
					h.OpenAIGateway.Messages(c)
				} else {
					h.Gateway.Messages(c)
				}
			default:
				if openai {
					h.OpenAIGateway.ChatCompletions(c)
				} else {
					h.Gateway.ChatCompletions(c)
				}
			}
		})
	}
}

// Only a live task capability can reach the ordinary gateway chain. Headers,
// model and billing binding are rebuilt from the immutable run snapshot.
func identityCompletionCapability(svc *service.ModelIdentityService, protocol string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("run"), 10, 64)
		credential := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if err != nil {
			c.AbortWithStatus(401)
			return
		}
		run, err := svc.Authorize(c.Request.Context(), id, credential)
		if err != nil {
			c.AbortWithStatus(401)
			return
		}
		configured := run.Options.Protocol
		if configured == "" {
			configured = "chat_completions"
		}
		if configured != protocol {
			c.AbortWithStatus(400)
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
		if _, err = svc.Authorize(c.Request.Context(), id, credential); err != nil {
			c.AbortWithStatus(401)
			return
		}
		key, err := svc.ValidateRun(c.Request.Context(), run)
		if err != nil {
			payload, _ := json.Marshal(map[string]any{"status": 409, "configuration_error": true, "error": "repair the account test configuration"})
			_ = svc.AppendProbe(c.Request.Context(), id, payload)
			c.AbortWithStatusJSON(409, gin.H{"error": gin.H{"message": "repair the account test configuration"}})
			return
		}
		raw, err := io.ReadAll(io.LimitReader(c.Request.Body, (64<<10)+1))
		var input map[string]json.RawMessage
		if err != nil || len(raw) > 64<<10 || json.Unmarshal(raw, &input) != nil {
			c.AbortWithStatus(400)
			return
		}
		// The engine's generated request is text-only and cannot install tools or
		// override account/group routing through arbitrary extension fields.
		allowed := map[string]bool{"model": true, "messages": true, "input": true, "stream": true, "stream_options": true, "max_tokens": true, "max_completion_tokens": true, "max_output_tokens": true, "reasoning_effort": true, "reasoning": true, "service_tier": true, "speed": true, "thinking": true, "output_config": true, "store": true}
		for field := range input {
			if !allowed[field] {
				c.AbortWithStatus(400)
				return
			}
		}
		for _, field := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
			if value, ok := input[field]; ok {
				var limit int
				if json.Unmarshal(value, &limit) != nil || limit < 1 || limit > 8192 {
					c.AbortWithStatus(400)
					return
				}
			}
		}
		input["model"], _ = json.Marshal(run.RequestModel)
		limit := "max_tokens"
		if protocol == "responses" {
			limit = "max_output_tokens"
			input["store"] = json.RawMessage("false")
		}
		if _, ok := input[limit]; !ok {
			input[limit] = json.RawMessage("8192")
		}
		body, _ := json.Marshal(input)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Request.ContentLength = int64(len(body))
		c.Request.Header = make(http.Header)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Header.Set("Authorization", "Bearer "+key.Key)
		c.Request.Header.Set("User-Agent", "Sub2API-Fingerpoint/1")
		if protocol == "messages" {
			c.Request.Header.Set("Anthropic-Version", "2023-06-01")
			if _, ok := input["speed"]; ok {
				c.Request.Header.Set("Anthropic-Beta", "fast-mode-2026-02-01")
			}
		}
		evidence := &service.IdentityProbeEvidence{AccountID: run.AccountID}
		ctx, cancel := context.WithCancel(service.WithIdentityEvidence(service.WithIdentityTarget(c.Request.Context(), run.AccountID, run.GroupID), evidence))
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
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
					if _, err := svc.Authorize(ctx, id, credential); err != nil {
						cancel()
						return
					}
				}
			}
		}()
		writer := &identityResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		started := time.Now()
		c.Next()
		probe := map[string]any{"protocol": protocol, "request_model": run.RequestModel, "target_account_id": run.AccountID, "status": writer.Status(), "duration_ms": time.Since(started).Milliseconds(), "client_request_id": writer.Header().Get("X-Client-Request-ID"), "evidence": evidence}
		if writer.Status() >= 400 {
			probe["error"] = "probe request failed"
			if writer.Status() == 401 || writer.Status() == 402 || writer.Status() == 403 || writer.Status() == 409 {
				probe["configuration_error"] = true
			}
		}
		// Responses may contain sensitive provider diagnostics. Keep only the safe
		// routing evidence here; full visible samples/usage arrive via the worker.
		encoded, _ := json.Marshal(probe)
		encoded = bytes.ReplaceAll(encoded, []byte(key.Key), []byte("[REDACTED]"))
		encoded = bytes.ReplaceAll(encoded, []byte(credential), []byte("[REDACTED]"))
		saveCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = svc.AppendProbe(saveCtx, id, encoded)
	}
}
