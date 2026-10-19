# Identity worker deployment

The worker is a Node 24 process supervised beside the Go binary. It listens on
`127.0.0.1:8081`, exposes only `/models` and the short lived `/run` capability,
and calls the Go internal capability route for each sample. Permanent API keys
stay in Go; the worker receives only a per-run 64-byte token.

The build pins `https://github.com/Ikaleio/lm-detector` at
`654c8334bb53951aaf04cbd61252d163649e1369` and verifies the packaged data
hashes. `shared_detector.bin`, `unified_bank.json`, and `tokenizer_bank.json`
are bundled; the full reference browser is retained as source provenance but
is not served to users.

Upgrading interrupts queued/running tasks from the previous engine while
preserving their reports. Configure the fixed test user once in Model identity
settings; account plans select only their billing group. Switching that user
updates dedicated Key ownership atomically and is blocked during active runs.

The UI uses a calibrated reference probability of at least 90% to confirm a
match or mismatch against the expected model. Lower probabilities remain
candidate rankings; partial or uncalibrated samples cannot confirm identity.
GPT-6 Astra and GPT-6.1 Sol share the detector's reference class. These are
site decision rules, not model-provider certification. Recent history displays
five colored slots, with hover/focus details and click-through reports.
