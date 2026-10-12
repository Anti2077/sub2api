package service

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// The hosted probe normally negotiates TLS 1.3. Some network paths reject that
// handshake, so allow one TLS 1.2 attempt only before any HTTP headers or body
// were sent. Never retry a response error or an uncertain paid-run outcome.
type identityTLSFallbackTransport struct {
	primary  *http.Transport
	fallback *http.Transport
}

func identityHTTPTransport() *identityTLSFallbackTransport {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		base = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: time.Second,
		}
	}
	primary := base.Clone()
	primary.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	fallback := primary.Clone()
	// This ceiling applies exclusively to a second, pre-request handshake.
	fallback.TLSClientConfig.MaxVersion = tls.VersionTLS12
	return &identityTLSFallbackTransport{primary: primary, fallback: fallback}
}

func (t *identityTLSFallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var handshakeFailed, wroteHeaders atomic.Bool
	trace := &httptrace.ClientTrace{
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			handshakeFailed.Store(identityTLSCompatibilityError(err))
		},
		WroteHeaders: func() { wroteHeaders.Store(true) },
	}
	traced := req.Clone(httptrace.WithClientTrace(req.Context(), trace))
	res, err := t.primary.RoundTrip(traced)
	if err == nil || req.URL.Scheme != "https" || req.Context().Err() != nil ||
		!handshakeFailed.Load() || wroteHeaders.Load() {
		return res, err
	}
	retry := req.Clone(req.Context())
	if req.Body != nil && req.Body != http.NoBody {
		// Transport closes even an unread body on a failed handshake. Only use
		// the caller's explicit replay factory; never reuse a closed stream.
		if req.GetBody == nil {
			return res, err
		}
		body, bodyErr := req.GetBody()
		if bodyErr != nil {
			return nil, bodyErr
		}
		retry.Body = body
	}
	return t.fallback.RoundTrip(retry)
}

func identityTLSCompatibilityError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	// Do not retry certificate, hostname, timeout, cancellation, or plaintext
	// endpoint errors. Only these remote TLS alerts indicate compatibility.
	return strings.Contains(err.Error(), "remote error: tls: protocol version not supported") ||
		strings.Contains(err.Error(), "remote error: tls: handshake failure")
}

func (t *identityTLSFallbackTransport) CloseIdleConnections() {
	t.primary.CloseIdleConnections()
	t.fallback.CloseIdleConnections()
}
