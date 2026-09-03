# 第一次 OpenAI 兼容请求

这个检查可以把“站点或密钥有问题”和“图形客户端配置有问题”分开。它只请求模型列表，不会生成长文本。

## 准备变量

把下面两个占位符换成站点给你的地址和 API Key：

```bash
export SUB2API_BASE_URL="https://your-sub2api.example"
export SUB2API_API_KEY="sk-your-key"
```

`export` 会把变量提供给当前终端会话中随后运行的命令。关闭这个终端窗口后，这两个变量通常不会继续保留。

## 请求模型列表

```bash
curl --fail-with-body \
  --header "Authorization: Bearer ${SUB2API_API_KEY}" \
  "${SUB2API_BASE_URL}/v1/models"
```

参数说明：

- `--fail-with-body`：HTTP 请求失败时返回非零状态，同时保留服务端错误正文；
- `--header`：发送 Bearer 鉴权头；
- 最后的 URL：请求 OpenAI 兼容的模型列表端点。

成功时会返回 JSON，并包含当前密钥可见的模型。不要只看 HTTP 状态，还要确认列表中存在你准备使用的模型 ID。

## 清理当前终端中的密钥

```bash
unset SUB2API_API_KEY
```

`unset` 会从当前终端会话移除这个变量，降低后续命令或调试输出意外带出密钥的风险。

## 如果失败

- `401`：检查 API Key 是否完整、有效并处于启用状态；
- `404`：检查站点地址，以及路径是否重复出现 `/v1/v1`；
- `429`：检查额度、并发或限流；
- 无法连接：检查域名、网络、代理和 TLS 证书。
