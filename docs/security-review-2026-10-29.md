# 2026-10-29 依赖与镜像安全复核

基线：`7ffe4be3b`。检查前端及身份引擎的完整 pnpm 依赖树、Go 依赖与调用路径，以及本次发布的 `custom-7ffe4be` 镜像。该复核不代表生产部署状态。

## 本次修复

| 问题 | 与项目的关系 | 处理 |
| --- | --- | --- |
| [CVE-2026-116591 / GHSA-fpmh-frq8-7rj8](https://github.com/advisories/GHSA-fpmh-frq8-7rj8)，10 月 27 日披露，zstd 两个连续数据块可触发解码器 panic | `internal/pkg/httputil` 解压客户端 zstd 请求体；公告样本能使基线版本的测试进程崩溃 | `klauspost/compress` 从 1.18.2 升至 1.20.2；加入经项目请求体读取器执行的回归测试，要求返回错误且不返回部分请求体 |
| [CVE-2026-63209 / GHSA-259r-337f-4rfw](https://github.com/advisories/GHSA-259r-337f-4rfw)，10 月 20 日披露，S2 字典整数溢出 | 依赖模块命中；当前模块没有导入 `compress/s2` | 同一压缩库升级同时消除该告警，不添加豁免 |
| [CVE-2026-111497](https://nvd.nist.gov/vuln/detail/CVE-2026-111497)，Brotli 解码器越界写 | 已发布镜像的 Alpine `brotli-libs` 为 1.2.0-r1，扫描命中严重漏洞 | 运行时执行 `apk upgrade`；Go Brotli 移植库同时从 1.2.0 升至包含相应校验修复的 1.2.7 |
| [CVE-2026-92392](https://nvd.nist.gov/vuln/detail/CVE-2026-92392)，curl HTTP/2 upload use-after-free | 已发布镜像中的 curl/libcurl 8.22.0-r0 命中高危漏洞 | 更新运行时系统包 |
| [GHSA-5gmw-xhrv-c9v3](https://github.com/advisories/GHSA-5gmw-xhrv-c9v3)、[GHSA-85c8-ppgw-ccpr](https://github.com/advisories/GHSA-85c8-ppgw-ccpr)，10 月 5 日披露，Tinypool 原型污染可辅助代码执行 | 前端 Vitest 2 的开发依赖链；利用需要原型污染等前置条件 | Vitest 及覆盖率组件升至 4.1.12，移除旧 Tinypool 依赖链 |
| [GHSA-587h-7wmj-h5pm](https://github.com/advisories/GHSA-587h-7wmj-h5pm)，10 月 20 日披露，Vite 编辑器启动路径检查可被绕过 | 开发服务器的 Windows 特定风险 | Vite 升至 7.3.7，Vue 插件升至 6.0.10；同步清理构建工具链的其他可修复高危告警 |
| 镜像预装 npm 中的 undici、brace-expansion 告警 | 位于 `/usr/local/lib/node_modules/npm`，身份引擎运行时不依赖这些包 | 从最终镜像删除 npm/Corepack 及其启动入口；构建阶段保留包管理工具 |

Node 24.22.1 已回补 CVE-2026-111497，并包含 undici 7.30.1。不能仅凭 `process.versions.brotli` 仍显示 1.2.0 就认定 Node 自带实现未修复；本次镜像命中的系统库和 npm 内置依赖是另外的副本。参见 [Node 24 变更记录](https://github.com/nodejs/node/blob/main/doc/changelogs/CHANGELOG_V24.md#24.22.1)。

## 持续检查

- pnpm 审计覆盖生产及开发依赖，并分别检查 `frontend`、`identity-engine`。
- 审计报告缺失或返回网络/注册表错误时，检查失败，避免被当作零漏洞。
- 自定义镜像发布前运行固定版本 Trivy，拦截高危和严重的系统包及语言依赖漏洞。
- 发布流程对运行时构建阶段禁用层缓存，确保每次重新获取系统安全更新。

## 已有例外与范围限制

- 前端保留两项既有 SheetJS 读取漏洞例外（2023/2024 年披露）。当前 `admin/UsageView.vue` 只生成并写出工作簿，没有 `XLSX.read` 或读取不可信工作簿的路径；此次未延长例外或添加新例外。npm 上的旧版 SheetJS 仍会触发这两项包版本告警。
- Go 保留既有 Hysteria 例外：依赖高于上游修复版本，项目仅使用客户端；例外将于 2026-11-09 到期。
- 依赖审计与镜像扫描不能证明业务代码不存在未知漏洞；生产是否使用修复镜像需另行核对部署版本。

## 本地验证结果

- Trivy 对基线镜像报告 10 条高危/严重告警；修复后本地构建的镜像为 0 条，包括系统包、安装的 Node 包和 Go 二进制依赖。
- 前端完整测试 417 个文件、3,303 项测试全部通过；lint、类型检查及 626 项关键测试通过。
- 身份引擎依赖审计零告警；引擎测试、容器管理测试通过。
- 后端完整单元测试、集成测试、lint 和 Go 漏洞策略检查通过。首轮并行构建负载下有两个既有计时测试失败，降低负载后独立重复三次和完整单测均通过，未改动其阈值。
- 单镜像的数据库初始化、身份引擎与后端启动、管理员 API、命令行入口和优雅退出验证通过；备份客户端 `pg_dump`、`psql` 可正常运行。
- 旧版压缩库在公告样本上发生 FSE 越界 panic；新版经项目请求体读取器返回错误，回归测试通过。
