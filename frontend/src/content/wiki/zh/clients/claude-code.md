# 使用 CC Switch 接入 Claude Code

这篇教程带你创建一把绑定 Claude/Anthropic 分组的 Sub2API API Key，通过 CC Switch 导入 Claude Code provider，并完成一次真实请求验证。

Claude Code 使用 Anthropic Messages API。Sub2API 导入到 CC Switch 时会使用站点根地址，Claude Code 再请求该地址下的 `/v1/messages`。因此不要在导入地址后手动重复添加 `/v1`。

> API Key 等同于密码。不要把真实 Key 放进聊天记录、截图、Issue、终端录屏或公开仓库。

## 你需要准备什么

- 一个可以登录的 Sub2API 账号；
- 一个可用的 Claude/Anthropic 分组；
- 已安装 [CC Switch](https://www.ccswitch.io/zh/)；
- 已安装 [Claude Code](https://code.claude.com/docs/en/setup)。

本文介绍的是 Claude Code，不是 Claude Desktop。创建密钥时必须选择 Claude/Anthropic 分组；OpenAI、Gemini 和 Grok 分组会被导入到其他 CC Switch 应用。

## 第一步：安装 Claude Code

macOS、Linux 或 WSL 可以使用 Anthropic 官方原生安装器：

```bash
curl -fsSL https://claude.ai/install.sh | bash
claude --version
```

这条安装命令中：

- `curl` 用来下载文件；
- `-f` 表示服务器返回错误状态时让命令失败；
- `-sS` 表示平时保持安静，但发生错误时仍显示信息；
- `-L` 表示跟随网页重定向；
- `| bash` 把下载到的官方安装脚本交给 Bash 执行；
- `claude --version` 用来确认 Claude Code 已经可以运行。

Windows PowerShell 使用：

```powershell
irm https://claude.ai/install.ps1 | iex
claude --version
```

其中 `irm` 是 `Invoke-RestMethod` 的别名，用于下载安装脚本文本；`iex` 是 `Invoke-Expression` 的别名，用于执行该文本。如果你不希望直接执行远程脚本，可以先打开上面的 Claude Code 官方安装文档，检查当前安装方式后再继续。

## 第二步：创建 Claude Code API Key

1. 登录 Sub2API，进入左侧的“API 密钥”页面。
2. 点击“创建密钥”。
3. 名称可以填写 `Claude Code - MacBook` 等便于识别的内容。
4. 在“分组”中选择一个 Claude/Anthropic 分组。
5. 第一次接入时，可以先不设置 IP 限制、过期时间和额外限流。
6. 点击“创建”，回到列表确认密钥状态为“启用”。

分组决定这把密钥可以访问哪些模型和路由。如果选择的不是 Claude/Anthropic 分组，“导入到 CCS”会按该分组的平台导入到其他应用，无法得到本文需要的 Claude Code provider。

## 第三步：导入到 CC Switch

1. 在 Sub2API 的“API 密钥”列表中找到刚创建的密钥。
2. 点击该行操作区的“导入到 CCS”。
3. 浏览器询问是否打开 CC Switch 时，选择允许。
4. 在 CC Switch 的导入预览中检查：
   - 应用类型是 `Claude Code`；
   - 供应商名称是当前 Sub2API 站点名称；
   - API 地址是站点根地址，例如 `https://your-sub2api.example`；
   - API Key 是刚创建的密钥。
5. 确认导入，然后在 Claude Code 的供应商列表中启用该 provider。

当前 Sub2API 的 Claude/Anthropic 分组会被导入为 CC Switch 的 `claude` 应用，endpoint 使用站点根地址。CC Switch 会把地址和认证信息写入 Claude Code 所需的 `ANTHROPIC_BASE_URL` 与认证配置中。

### 如果没有“导入到 CCS”按钮

可能是站点管理员关闭了 CCS 导入功能，或者浏览器没有成功打开 `ccswitch://` 协议。这时可以在 CC Switch 中手动添加 Claude Code 供应商：

| 字段 | 填写方式 |
| --- | --- |
| 应用 | `Claude Code` |
| 供应商名称 | 任意便于识别的名称，例如 `Sub2API` |
| API 地址 | Sub2API 站点根地址，例如 `https://your-sub2api.example` |
| API Key | 刚创建的 Sub2API API Key |

API 地址不要写成 `https://your-sub2api.example/v1`。Claude Code 会自己请求 `/v1/messages`，重复添加 `/v1` 可能导致请求路径变成 `/v1/v1/messages`。

## 第四步：重新打开 Claude Code

为了排除旧进程仍在使用原供应商，建议先完全退出正在运行的 Claude Code，再打开一个新终端并执行：

```bash
claude
```

如果首次打开仍显示官方登录流程，先退出 Claude Code，确认 CC Switch 中刚导入的 Claude Code provider 处于启用状态，然后重新启动。不要为了排错把真实 API Key 粘贴到聊天中。

## 第五步：完成一次最小验证

在一个不重要的测试目录中启动 Claude Code，输入：

```text
请只回复：Claude Code 已连接 Sub2API。不要修改任何文件。
```

收到正常回复后，回到 Sub2API 的使用记录，确认出现了这次请求。

接入成功应同时满足：

- CC Switch 中的 Claude Code provider 处于启用状态；
- Claude Code 能收到真实模型响应；
- 请求模型属于该密钥绑定分组允许的范围；
- Sub2API 使用记录中出现对应请求。

只看到 CC Switch 显示“启用”还不够，最终以 Claude Code 的真实响应和 Sub2API 使用记录为准。

## 常见问题

### 返回 `401`

通常是 API Key 不完整、已停用、已过期，或 CC Switch 没有把认证信息写入当前 Claude Code provider。回到 Sub2API 检查密钥状态，再重新导入并启用。

### 返回 `404`

优先检查 API 地址。Claude Code provider 应使用站点根地址，不要在后面重复添加 `/v1`。

### 提示模型不存在或不可用

当前请求使用的 Claude 模型不在密钥绑定分组的允许范围内。回到 Sub2API 检查分组模型与映射，不要只根据模型名称猜测。

### 仍然连接 Anthropic 官方服务

完全退出 Claude Code，确认 CC Switch 当前启用的是刚导入的 provider，然后重新打开终端运行 `claude`。如果你同时在 shell profile 中手动配置过 `ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY` 或 `ANTHROPIC_AUTH_TOKEN`，这些旧值也可能与 CC Switch 配置冲突。

### CC Switch 打开后导入到了其他应用

检查这把 API Key 绑定的分组平台。Claude/Anthropic 分组才会直接导入到 Claude Code；OpenAI、Gemini 和 Grok 分组分别有自己的客户端配置。

## 安全提醒

- 不要把真实 API Key 写进项目文件、`CLAUDE.md` 或公开的 shell 脚本；
- 不要分享包含认证字段的完整 CC Switch 或 Claude Code 配置；
- 怀疑 Key 泄露时，立即停用旧 Key 并创建新 Key；
- 测试 Agent 时先使用不重要的目录，并认真查看文件修改和命令执行授权。

## 本文核对范围

- Claude Code 官方原生安装方式与 `claude` 命令；
- Sub2API 的 CC Switch 导入实现：Claude/Anthropic 分组映射到 `claude`，endpoint 使用站点根地址；
- CC Switch 的 Claude Code provider 与 `ccswitch://` 导入格式；
- 核对日期：2026-09-05。
