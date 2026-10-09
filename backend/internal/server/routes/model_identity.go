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

func registerIdentityProbe(r *gin.Engine, h *handler.Handlers, svc *service.ModelIdentityService, auth middleware.APIKeyAuthMiddleware, requireGroup gin.HandlerFunc) {
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
	r.POST("/internal/model-identity/:run/probe", middleware.RequestBodyLimit(64<<10), func(c *gin.Context) {
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
		if e = svc.ProbeSlot(c.Request.Context(), id, true); e != nil {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		defer func() {
			ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			_ = svc.ProbeSlot(ctx, id, false)
		}()
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
	}, middleware.ClientRequestID(), handler.InboundEndpointMiddleware(), gin.HandlerFunc(auth), middleware.GroupModelAllowlist(), requireGroup, func(c *gin.Context) {
		switch getGroupPlatform(c) {
		case service.PlatformOpenAI, service.PlatformGrok, service.PlatformKimi, service.PlatformZhipu, service.PlatformDeepseek, service.PlatformMiniMax, service.PlatformOpenCodeGo:
			h.OpenAIGateway.ChatCompletions(c)
		default:
			h.Gateway.ChatCompletions(c)
		}
	})
}
