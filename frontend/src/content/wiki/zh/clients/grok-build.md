# 使用 CC Switch 接入 Grok Build

Grok Build 是 SpaceXAI 官方推出的终端 AI 编程 Agent，命令名为 `grok`。它可以理解代码仓库、编辑文件、执行命令并管理较长的任务；项目源码以 Apache-2.0 许可证发布在 [xai-org/grok-build](https://github.com/xai-org/grok-build)。

这篇教程带你安装 Grok Build，创建一把绑定 Grok 分组的 Sub2API API Key，通过 CC Switch 导入 provider，并完成一次真实请求验证。

> Grok Build 和其他编程 Agent 一样可以修改文件、执行命令。第一次验证请使用不重要的测试目录，并逐项检查授权。API Key 不要放进聊天、截图、Issue 或公开仓库。

## 你需要准备什么

- 一个可以登录的 Sub2API 账号；
- 一个可用的 Grok 分组；
- 已安装 Grok Build；
- 已安装 [CC Switch](https://www.ccswitch.io/zh/) v3.18.0 或更新版本。Grok Build 从 CC Switch v3.18.0 开始成为可直接管理的应用。

可以在 [Grok Build 官方页面](https://x.ai/cli) 查看产品介绍，在 [官方文档](https://docs.x.ai/build/overview) 查看完整使用说明。

## 第一步：安装 Grok Build

macOS、Linux 或 Git Bash 使用官方安装脚本：

```bash
curl -fsSL https://x.ai/cli/install.sh | bash
grok --version
```

这条安装命令中：

- `curl` 用来下载文件；
- `-f` 表示服务器返回错误状态时让命令失败；
- `-sS` 表示平时保持安静，但发生错误时仍显示信息；
- `-L` 表示跟随网页重定向；
- `| bash` 把下载到的官方安装脚本交给 Bash 执行；
- `grok --version` 用来确认 `grok` 命令已经安装成功。

Windows PowerShell 使用：

```powershell
irm https://x.ai/cli/install.ps1 | iex
grok --version
```

其中 `irm` 是 `Invoke-RestMethod` 的别名，用于下载安装脚本文本；`iex` 是 `Invoke-Expression` 的别名，用于执行该文本。如果你对执行远程脚本有顾虑，可以先从上面的官方开源仓库检查安装脚本，再决定是否执行。

## 第二步：创建 Grok Build API Key

1. 登录 Sub2API，进入左侧的“API 密钥”页面。
2. 点击“创建密钥”。
3. 名称可以填写 `Grok Build - MacBook` 等便于识别的内容。
4. 在“分组”中选择一个 Grok 分组。
5. 第一次接入时，可以先不设置 IP 限制、过期时间和额外限流。
6. 点击“创建”，回到列表确认密钥状态为“启用”。

必须选择 Grok 分组。Sub2API 会根据密钥所属分组决定“导入到 CCS”应该创建哪一种客户端 provider。

## 第三步：导入到 CC Switch

1. 在 Sub2API 的“API 密钥”列表中找到刚创建的密钥。
2. 点击该行操作区的“导入到 CCS”。
3. 浏览器询问是否打开 CC Switch 时，选择允许。
4. 在 CC Switch 的导入预览中检查：
   - 应用类型是 `Grok Build`；
   - 供应商名称是当前 Sub2API 站点名称；
   - API 地址是站点根地址加 `/v1`，例如 `https://your-sub2api.example/v1`；
   - API Key 是刚创建的密钥；
   - 默认模型是 `grok-4.5`。
5. 确认导入，然后在 Grok Build 的供应商列表中启用该 provider。

当前 Sub2API 会把 Grok 分组导入为 CC Switch 的 `grokbuild` 应用，并保证 endpoint 只有一个 `/v1` 后缀。CC Switch 会把 provider 应用到 Grok Build 的配置；用户级配置通常位于 `~/.grok/config.toml`，Windows 通常位于 `%USERPROFILE%\.grok\config.toml`。

### 如果没有“导入到 CCS”按钮

先检查 CC Switch 是否至少为 v3.18.0，以及浏览器是否允许打开 `ccswitch://`。仍然无法导入时，可以在 CC Switch 中手动添加 Grok Build provider：

| 字段 | 填写方式 |
| --- | --- |
| 应用 | `Grok Build` |
| 供应商名称 | 任意便于识别的名称，例如 `Sub2API` |
| Base URL | Sub2API 站点根地址加 `/v1`，例如 `https://your-sub2api.example/v1` |
| API Key | 刚创建的 Sub2API API Key |
| 模型 | 先使用 `grok-4.5` |

如果站点地址本身已经以 `/v1` 结尾，不要再添加一次。

## 第四步：重新打开 Grok Build

切换 provider 后，先退出正在运行的 Grok Build，再打开一个新终端。首先检查当前生效配置：

```bash
grok inspect
```

`inspect` 会显示 Grok Build 最终采用了哪些配置来源和值，可以帮助判断 CC Switch 写入的 Base URL 是否生效。分享输出时先检查并遮住任何可能的认证信息。

然后读取当前 endpoint 返回的模型列表：

```bash
grok models
```

如果能够看到 `grok-4.5`，就可以启动 Grok Build：

```bash
grok
```

## 第五步：选择模型并验证

当前 Sub2API 自动导入默认使用 `grok-4.5`。进入 Grok Build 后，可以执行：

```text
/model grok-4.5
```

然后输入一个不修改文件的最小任务：

```text
请只回复：Grok Build 已连接 Sub2API。不要修改任何文件，也不要执行命令。
```

收到正常回复后，回到 Sub2API 的使用记录，确认出现了这次请求。

接入成功应同时满足：

- CC Switch 中的 Grok Build provider 处于启用状态；
- `grok models` 能读取当前分组允许的模型；
- Grok Build 能收到真实模型响应；
- Sub2API 使用记录中出现对应请求。

只看到 CC Switch 显示“启用”还不够，最终以 Grok Build 的真实响应和 Sub2API 使用记录为准。

## 模型建议

- `grok-4.5`：当前自动导入的默认模型，适合先完成连接验证和日常通用任务；
- `grok-build-0.1`：可以用于编程 Agent 场景，但必须先确认 `grok models` 的结果和当前分组确实提供它。

如果模型列表中没有目标模型，不要只手动输入名称。先回到 Sub2API 检查密钥绑定分组与模型范围。

## 常见问题

### CC Switch 中没有 Grok Build

检查 CC Switch 版本。Grok Build 的一等应用支持从 v3.18.0 开始提供，旧版本需要先升级。

### 返回 `401`

通常是 API Key 不完整、已停用、已过期，或 Grok Build 仍在使用旧认证。回到 Sub2API 检查密钥状态，再重新导入、启用 provider 并重启 `grok`。

### 返回 `404`

检查 Base URL 是否正好以一个 `/v1` 结尾。正确示例是 `https://your-sub2api.example/v1`；不要写成站点根地址，也不要重复成 `/v1/v1`。

### `grok models` 获取不到模型

依次检查网络、API Key、Base URL、密钥状态和 Grok 分组。还可以运行 `grok inspect`，确认最终生效的配置没有被其他用户级、项目级或环境变量配置覆盖。

### 模型返回不支持或不存在

当前密钥绑定的 Grok 分组没有开放该模型，或者模型配置没有使用适合的接口。Sub2API 的 Grok Build 文本请求使用 Responses API；通过本站一键导入时应优先保留自动生成的配置。

### 仍然打开官方登录或连接官方服务

先退出 Grok Build，确认 CC Switch 当前启用的是刚导入的 Grok Build provider，再打开新终端运行 `grok inspect`。如果你以前手动设置过 `GROK_MODELS_BASE_URL`、`XAI_API_KEY` 或 `~/.grok/config.toml`，旧配置可能覆盖或干扰 CC Switch 的结果。

## 安全提醒

- 不要把真实 API Key 写进项目内的 `.grok/config.toml` 或提交到 Git；
- 不要分享包含认证字段的完整 `grok inspect`、CC Switch 或 Grok Build 配置；
- 怀疑 Key 泄露时，立即停用旧 Key 并创建新 Key；
- Agent 获得文件和终端权限后可能产生真实修改，首次使用请逐项检查授权。

## 本文核对范围

- Grok Build 官方安装脚本、`grok` 命令、Apache-2.0 开源仓库与用户文档；
- Grok Build 的 `grok inspect`、`grok models`、`/model` 与自定义 Responses endpoint 配置；
- CC Switch v3.18.0 的 Grok Build 一等应用支持；
- Sub2API 的 CC Switch 导入实现：Grok 分组映射到 `grokbuild`，endpoint 使用站点根地址加 `/v1`，默认模型为 `grok-4.5`；
- 核对日期：2026-09-05。
