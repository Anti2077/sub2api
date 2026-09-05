# 使用 Open Minis 接入 Sub2API

Open Minis 相比主流 AI 客户端相对小众，但如果你主要在手机上使用 Agent，它是我目前用过体验最好的移动端 Agent 之一。它支持 iPhone、iPad 和 Android，并且可以连接 OpenAI 兼容接口。

本文使用 Sub2API 的 API Key 接入 Open Minis。你只需要填写 Base URL 和 API Key，然后刷新模型列表即可开始使用。

> API Key 等同于密码。不要把真实 Key 放进截图、聊天记录、Issue 或公开仓库。本文中的地址和 Key 均使用占位符。

## 下载 Open Minis

- iOS / iPadOS：前往 [App Store 下载 Open Minis](https://apps.apple.com/app/id6759188481)。
- Android：前往 [Open Minis Android APK Releases](https://github.com/OpenMinis/OpenMinis/releases) 下载对应版本。这里使用的是 Open Minis 项目的 GitHub Release 页面，不是 Google Play 商店；安装前请核对版本和发布信息。

## 第一步：准备 Sub2API API Key

如果你还没有 API Key：

1. 登录 Sub2API，打开“API 密钥”页面。
2. 点击“创建密钥”。
3. 选择给 Open Minis 使用的分组。分组决定这把 Key 可以访问哪些模型和路由。
4. 创建完成后确认密钥状态为“启用”。

已经有可用 Key 的话，可以直接继续。建议先在 Sub2API 的“模型”或分组配置中确认目标模型可用。

## 第二步：在 Open Minis 中添加自定义 API

在 Open Minis 的设置中，新建或编辑一个自定义 API 服务，填写以下内容：

| 字段 | 填写方式 |
| --- | --- |
| 标签 | 任意便于识别的名称，例如 `Sub2API` |
| 凭证 / API Key | 填入刚创建的 Sub2API API Key |
| 自定义 API 地址 / Base URL | 填入 Sub2API 站点根地址，例如 `https://your-sub2api.example` |
| API 格式 | 选择 `Chat Completions` |
| 状态 | 开启 |

如果界面中有“自动附加 `/v1`”开关：

- Base URL 填站点根地址时，保持开启；
- 如果你填写的地址本身已经以 `/v1` 结尾，就关闭它，避免最终请求变成 `/v1/v1/...`。

“Azure OpenAI”只适用于 Azure OpenAI 端点。接入 Sub2API 时保持关闭即可。自定义 User-Agent 没有特殊要求时保持默认。

## 第三步：获取模型并选择推荐模型

1. 保存 API 配置，并确认状态为“已启用”。
2. 在模型区域点击刷新或“获取模型”。
3. 等待 Open Minis 从 Sub2API 读取可用模型。
4. 在模型列表中选择 `gpt-5.6-luna`。

推荐 `gpt-5.6-luna` 作为移动端日常 Agent 的默认模型。如果分组没有开放这个模型，就选择模型列表中实际返回的可用模型，不要只按名称手动猜测。

## 第四步：发起一次最小验证

新建一个短任务，例如：

```text
请只回复：Open Minis 已连接 Sub2API。
```

能够正常返回文本，就说明 Base URL、API Key、接口格式和模型已经基本配置正确。需要进一步确认时，回到 Sub2API 的使用记录检查是否出现对应请求。

## 常见问题

### 获取不到模型

按以下顺序检查：

1. API Key 是否完整，且状态为“启用”；
2. Base URL 是否填写为站点根地址；
3. “自动附加 `/v1`”是否与 Base URL 的实际写法匹配；
4. API 格式是否选择 `Chat Completions`；
5. Key 所属分组是否有可用模型。

### 返回 `401`

通常是 API Key 错误、复制不完整、已停用或已过期。重新复制 Key，确认前后没有空格，并检查 Sub2API 中的状态。

### 返回 `404` 或请求路径异常

检查是否重复附加了 `/v1`。最常见的正确组合是：Base URL 填站点根地址，并开启“自动附加 `/v1`”。如果 Base URL 已经包含 `/v1`，则关闭自动附加。

### `gpt-5.6-luna` 不在列表中

这通常表示当前 Key 绑定的分组没有开放该模型。回到 Sub2API 查看分组允许的模型，或者直接使用 Open Minis 刷新后返回的其他可用模型。

## 本文核对范围

- Open Minis 官方网站提供的 iOS App Store 与 Android Release 下载入口；
- Open Minis 设置截图中的 Base URL、API Key、Chat Completions、自动附加 `/v1` 和模型刷新流程；
- Sub2API OpenAI 兼容接口的站点根地址、API Key 与分组模型关系；
- 核对日期：2026-09-05。
