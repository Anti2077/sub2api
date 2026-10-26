# LM Fingerpoint Detector notice

This worker vendors the MIT licensed `Ikaleio/lm-detector` source at commit
`4dcb4216151cbd5066d515297720130a9fd3b198`:
https://github.com/Ikaleio/lm-detector

The vendored source, detector, reference bank and calibration data are kept
under `vendor/lm-detector`. `manifest.json` records SHA-256 checksums and the
build verifies them before compiling. The local HTTP adapter and Sub2API
integration are original modifications; they run on Node 24 and only bind the
worker to loopback.

Historical BazaarLink reports and their official links remain readable in the
database/UI. New runs never call BazaarLink or require a public Base URL.
