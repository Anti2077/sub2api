# Model identity worker

This worker vendors BazaarLink/LLMprobe-engine at commit
`5c41136741ca52b5637879cca7bd0cae07404646` (version 0.11.0).
Source: https://github.com/BazaarLink/LLMprobe-engine

The engine is licensed under AGPL-3.0-only. Its complete source, tests,
documentation and license are in `vendor/bazaarlink`. The adapter in this
directory is also distributed under AGPL-3.0-only. Publish the corresponding
source of the deployed build to users of the service.

The adapter only runs text identity and submodel probes. Self-declarations
and response model names are auxiliary evidence. IKP abstains because the
public engine does not include its raw baseline corpus. This integration
adds a conservative product gate: stale baselines cannot produce a positive
identity verdict, even though the pinned engine retains their scoring authority.
