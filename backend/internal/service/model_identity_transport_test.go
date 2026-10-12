package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type identityHandshakeDropListener struct {
	net.Listener
	attempts atomic.Int32
}

func (l *identityHandshakeDropListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if l.attempts.Add(1) != 1 {
			return conn, nil
		}
		// Consume only the ClientHello, then simulate a TLS 1.3 network reset.
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var hello [4096]byte
		_, _ = conn.Read(hello[:])
		_ = conn.Close()
	}
}

func identityTrustTestServer(t *testing.T, tr *identityTLSFallbackTransport, server *httptest.Server) {
	t.Helper()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	tr.primary.TLSClientConfig.RootCAs = roots
	tr.fallback.TLSClientConfig.RootCAs = roots
	t.Cleanup(tr.CloseIdleConnections)
}

func TestIdentityTransportNegotiatesTLS13(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.TLS.Version != tls.VersionTLS13 {
			t.Errorf("TLS version = %x", r.TLS.Version)
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	defer server.Close()
	tr := identityHTTPTransport()
	identityTrustTestServer(t, tr, server)
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	res, err := client.Post(server.URL, "application/json", strings.NewReader(`{"run":"once"}`))
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, int32(1), requests.Load())
}

func TestIdentityTransportHandshakeFallbackSendsBodyOnce(t *testing.T) {
	var requests atomic.Int32
	body := make(chan string, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.TLS.Version != tls.VersionTLS12 {
			t.Errorf("fallback TLS version = %x", r.TLS.Version)
		}
		content, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("body read: %v", err)
		}
		body <- string(content)
		w.WriteHeader(http.StatusNoContent)
	}))
	listener := &identityHandshakeDropListener{Listener: server.Listener}
	server.Listener = listener
	server.StartTLS()
	defer server.Close()
	tr := identityHTTPTransport()
	identityTrustTestServer(t, tr, server)
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	res, err := client.Post(server.URL, "application/json", strings.NewReader(`{"run":"once"}`))
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, int32(2), listener.attempts.Load())
	require.Equal(t, int32(1), requests.Load())
	require.Equal(t, `{"run":"once"}`, <-body)
}

func TestIdentityTransportDoesNotReplayAcceptedPOST(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("missing hijacker")
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()
	tr := identityHTTPTransport()
	identityTrustTestServer(t, tr, server)
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	_, err := client.Post(server.URL, "application/json", strings.NewReader(`{"run":"once"}`))
	require.Error(t, err)
	require.Equal(t, int32(1), requests.Load())
}

func TestIdentityTransportDoesNotRetryInvalidCertificate(t *testing.T) {
	var handshakes atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("invalid certificate accepted") }))
	server.TLS = &tls.Config{GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) { handshakes.Add(1); return nil, nil }}
	server.StartTLS()
	defer server.Close()
	tr := identityHTTPTransport()
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	_, err := client.Post(server.URL, "application/json", strings.NewReader(`{"run":"once"}`))
	require.Error(t, err)
	var verification *tls.CertificateVerificationError
	require.True(t, errors.As(err, &verification))
	require.Equal(t, int32(1), handshakes.Load())
}

func TestIdentityTransportDoesNotReplayUnrewindableBody(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("non-replayable body was replayed") }))
	listener := &identityHandshakeDropListener{Listener: server.Listener}
	server.Listener = listener
	server.StartTLS()
	defer server.Close()
	tr := identityHTTPTransport()
	identityTrustTestServer(t, tr, server)
	req, err := http.NewRequest(http.MethodPost, server.URL, io.NopCloser(strings.NewReader("once")))
	require.NoError(t, err)
	require.Nil(t, req.GetBody)
	_, err = (&http.Client{Transport: tr, Timeout: 5 * time.Second}).Do(req)
	require.Error(t, err)
	require.Equal(t, int32(1), listener.attempts.Load())
}

func TestIdentityTLSCompatibilityErrorBoundaries(t *testing.T) {
	for _, err := range []error{io.EOF, io.ErrUnexpectedEOF, syscall.ECONNRESET, errors.New("remote error: tls: handshake failure"), errors.New("remote error: tls: protocol version not supported")} {
		require.True(t, identityTLSCompatibilityError(err), "%v", err)
	}
	for _, err := range []error{nil, context.Canceled, context.DeadlineExceeded, &tls.CertificateVerificationError{Err: errors.New("certificate invalid")}, errors.New("tls: first record does not look like a TLS handshake"), errors.New("remote error: tls: bad certificate")} {
		require.False(t, identityTLSCompatibilityError(err), "%v", err)
	}
}

func TestIdentityRunClientRefusesCredentialRedirect(t *testing.T) {
	var targets atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targets.Add(1) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client := identityRunHTTPClient()
	defer client.CloseIdleConnections()
	res, err := client.Post(server.URL, "application/json", strings.NewReader(`{"run":"once"}`))
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
	require.Zero(t, targets.Load())
}
