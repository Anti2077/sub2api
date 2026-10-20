package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/gin-gonic/gin"
)

const replayLimit = 2 * 1024 * 1024

type gatewayReplayStore interface {
	ClaimGateway(context.Context, int64, string, string, string) (rewards.GatewayReceipt, error)
	FinishGateway(context.Context, int64, string, int, string, []byte, bool) error
}
type replayWriter struct {
	gin.ResponseWriter
	body     bytes.Buffer
	overflow bool
}

func (w *replayWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if !w.overflow {
		if w.body.Len()+n > replayLimit {
			w.overflow = true
			w.body.Reset()
		} else {
			_, _ = w.body.Write(b[:n])
		}
	}
	return n, err
}
func (w *replayWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

// GatewayReplay requires explicit opt-in. Authentication and routing checks run
// first. A durable claim precedes every provider call; an existing claim can
// only replay its response or fail closed. Correlation headers are never used
// as client-controlled billing deduplication keys.
func GatewayReplay(store gatewayReplayStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("Idempotency-Key")
		if id == "" {
			c.Next()
			return
		}
		fail := func(status int, msg string) {
			c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"type": "idempotency_error", "message": msg}})
		}
		if len(id) > 128 || strings.IndexFunc(id, func(r rune) bool { return r < 33 || r > 126 }) >= 0 {
			fail(400, "Invalid Idempotency-Key")
			return
		}
		key, ok := GetAPIKeyFromContext(c)
		if !ok || store == nil {
			fail(503, "Idempotency unavailable")
			return
		}
		path := c.Request.URL.Path
		supported := c.Request.Method == http.MethodPost && (strings.HasSuffix(path, "/responses") || strings.HasSuffix(path, "/messages") || strings.HasSuffix(path, "/chat/completions") || strings.HasSuffix(path, "/completions") || strings.Contains(path, ":generateContent") || strings.Contains(path, ":streamGenerateContent"))
		if !supported {
			fail(400, "Idempotency-Key is supported only on text generation endpoints")
			return
		}
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, replayLimit+1))
		if err != nil || len(body) > replayLimit {
			fail(413, "Idempotent request exceeds body limit")
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		group := int64(0)
		if key.GroupID != nil {
			group = *key.GroupID
		}
		// Include all request headers that can affect provider interpretation. Auth,
		// correlation and transport headers do not affect the generated response.
		headers := c.Request.Header.Clone()
		for _, h := range []string{"Authorization", "X-Api-Key", "X-Goog-Api-Key", "Idempotency-Key", "X-Client-Request-Id", "Content-Length", "Connection", "Accept-Encoding", "User-Agent"} {
			headers.Del(h)
		}
		fingerprint := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%d\n%v\n%s", c.Request.Method, c.Request.URL.RequestURI(), group, headers, body))))
		requestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string)
		r, err := store.ClaimGateway(c.Request.Context(), key.ID, id, fingerprint, requestID)
		if err != nil {
			fail(503, "Unable to claim idempotent request")
			return
		}
		if !r.Claimed {
			if r.Fingerprint != fingerprint {
				fail(409, "Idempotency-Key payload does not match")
				return
			}
			if r.State != "complete" {
				c.Header("Retry-After", "1")
				fail(409, "Original request pending or response unavailable; it will not be forwarded again")
				return
			}
			c.Header("X-Client-Request-ID", r.RequestID)
			c.Header("Idempotency-Replayed", "true")
			c.Data(r.Status, r.ContentType, r.Body)
			c.Abort()
			return
		}
		w := &replayWriter{ResponseWriter: c.Writer}
		c.Writer = w
		c.Next()
		// A disconnected client must not erase an already-used provider claim.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Second)
		defer cancel()
		// On failure the durable pending claim remains fail-closed.
		_ = store.FinishGateway(ctx, key.ID, id, w.Status(), w.Header().Get("Content-Type"), w.body.Bytes(), w.overflow)
	}
}
