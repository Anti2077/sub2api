package tlsfingerprint

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Both crypto/tls and uTLS use crypto/x509 to verify upstream certificates.
// Use the original roots explicitly so an OS trust-store update cannot mask
// a regression to a Go toolchain that accepts these compromised public keys.
func TestRejectCompromisedEMudhraChains(t *testing.T) {
	dir := filepath.Join("testdata", "emudhra")
	rootPEM, err := os.ReadFile(filepath.Join(dir, "roots.pem"))
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootPEM) {
		t.Fatal("could not load original eMudhra trust anchors")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*-emudhra.we-have-a-problem.com.pem"))
	if err != nil || len(files) != 12 {
		t.Fatalf("expected all 12 reported chains, got %d: %v", len(files), err)
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var certs []*x509.Certificate
			for len(data) > 0 {
				block, rest := pem.Decode(data)
				if block == nil {
					t.Fatal("invalid certificate PEM")
				}
				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					t.Fatal(err)
				}
				certs = append(certs, cert)
				data = rest
			}
			if len(certs) < 2 {
				t.Fatal("missing intermediate")
			}
			intermediates := x509.NewCertPool()
			for _, cert := range certs[1:] {
				intermediates.AddCert(cert)
			}
			// Freeze time within certificate validity, so expiry cannot turn
			// this security regression into a false pass in the future.
			_, err = certs[0].Verify(x509.VerifyOptions{
				Roots: roots, Intermediates: intermediates,
				DNSName:     certs[0].DNSNames[0],
				CurrentTime: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
			})
			if err == nil {
				t.Fatal("accepted a compromised eMudhra chain; rebuild with Go 1.27.3 or later")
			}
			var invalid x509.CertificateInvalidError
			if !errors.As(err, &invalid) || invalid.Reason != x509.NoValidChains {
				t.Fatalf("expected compromised-key rejection, got %v", err)
			}
		})
	}
}
