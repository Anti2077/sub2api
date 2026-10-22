Public certificate fixtures from https://emudhra.we-have-a-problem.com/:

- The 12 test-server chains come from `chains.zip` (captured 2026-10-20).
- `roots.pem` contains the original roots published in `roots.json`.

These are public certificates only. No private keys are included.
The offline test supplies the original trust anchors and freezes verification
time within the certificates' validity. Go 1.27.2 accepts these chains; Go
1.27.3 must reject their compromised intermediate public keys even when the
original roots remain trusted. uTLS also uses Go's `crypto/x509` verifier.
