# Wiki Content Map

The navigation is task-oriented. Product internals and administrator concepts
should not appear before a new user can complete a first request.

## Launch content

| Section | Article | Outcome | Priority |
| --- | --- | --- | --- |
| Getting started | 获取 API Key 与接口地址 | Reader knows which values are needed and how to store them safely | P0 |
| Clients | 使用 CC Switch 连接 Sub2API | Reader configures a provider and completes one verified request | P0 |
| Troubleshooting | 请求失败时先看这里 | Reader distinguishes key, endpoint, model, quota, and network failures | P0 |
| Getting started | 第一次 OpenAI 兼容请求 | Reader validates the service independently of a GUI client | P1 |
| Getting started | 模型、分组与倍率 | Reader understands model access and billing without administrator details | P1 |
| Troubleshooting | CC Switch 切换后配置丢失 | Reader restores a backup and safely compares configuration changes | P1 |

## Candidate follow-ups

These are ideas, not promised compatibility. Each guide requires a currently
verified client version before publication.

| Section | Candidate article |
| --- | --- |
| Clients | Codex CLI / Codex app |
| Clients | Claude Code |
| Clients | Cherry Studio |
| Clients | OpenAI-compatible SDKs |
| Concepts | Base URL、端点与模型名的关系 |
| Concepts | 用量、余额与倍率如何计算 |
| Security | API Key 的保存、轮换与撤销 |
| Troubleshooting | `401`、`403`、`404`、`429` 与 `5xx` |

## Standard article shape

Every procedural article should use this order:

1. What the reader will accomplish.
2. Prerequisites and supported version.
3. Safety or billing warning when applicable.
4. Numbered configuration steps.
5. Minimal verification.
6. Expected result.
7. Failure diagnosis and recovery.
8. Last-verified date and evidence.
