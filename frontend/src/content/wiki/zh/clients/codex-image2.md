# 使用 codex-image-gen++ 在 Codex 中生成和编辑图片

安装 [codex-image-gen++](https://github.com/Anti2077/codex-image-gen-plus-plus) MCP，在终端用 `init` 配好接口地址和 API Key，之后直接在 Codex 对话里提出图片需求。模型选择、提示词、参数和保存位置交给 Codex。

本文替换原来的 `codex-image2` Skill 教程，原页面链接继续可用。已有 Skill 不需要重新适配，按下列步骤安装 MCP 即可。

## 准备工作

- Windows、macOS 或 Linux；
- Node.js 22.13 或更新版本，推荐 Node.js 24 LTS，安装时包含 npm；
- Codex CLI，用于自动注册 MCP；Codex 桌面版也可以使用注册后的 MCP；
- 一把本站 API Key，绑定已开启生图且有可用图片模型的 GPT 分组。聊天可用不代表图片权限也可用；
- 如果准备从 CC Switch 导入，先在 CC Switch 中启用本站的 Codex 供应商。可参考[使用 CC Switch 接入 Codex](/wiki/clients/cc-switch)。

不要把 API Key 发到 Codex 聊天里；手动配置时只在终端的隐藏输入中填写。初始化只读取模型列表，不会自动生成测试图；真实生成或编辑会按本站计费规则消耗额度。

## 第一步：安装

打开终端执行：

```sh
npm install -g https://github.com/Anti2077/codex-image-gen-plus-plus/releases/latest/download/codex-image-gen-plus-plus.tgz
```

`npm install` 安装发布包，`-g` 让当前系统可以直接使用命令；链接指向 GitHub 最新正式版安装包。

然后启动配置向导：

```sh
codex-image-gen-plus-plus init
```

`init` 用来设置接口、检查模型列表并选择是否注册到 Codex。Windows PowerShell 如果提示脚本执行受限，可改用 `codex-image-gen-plus-plus.cmd init`。

## 第二步：完成初始化

1. 选择“中文”。
2. 选择“导入 CC Switch 当前 Codex 提供商”，确认 CC Switch 数据目录；不使用 CC Switch 时选择手动输入。
3. 手动输入时，API 地址填写 `https://cdn-api.anti2077.xyz/v1`，API Key 填写本站创建的用户密钥。根地址 `https://cdn-api.anti2077.xyz` 也会自动补为 `/v1`，不要填写 `/images/generations` 等完整端点。
4. 选择密钥保存位置：本地私有配置文件为默认兼容模式；也可选择系统密钥库。密钥库模式会在 MCP 进程内缓存密钥，每次生图无需重新读取，但新进程、密钥库锁定等情况仍可能触发系统授权。
5. 按需开启工作区保护，填写一个已存在目录的绝对路径。开启后图片只能保存到此目录内；这是此 MCP 的固定输出范围，不会随 Codex 聊天自动切换，使用其他项目时可重新运行 `init` 修改。
6. 等待 `/models` 检查成功并显示模型列表。这只确认连接、认证和可见模型，实际生图能力需通过真实请求确认。
7. 在“现在注册到 Codex？”中选择“是”。找不到 Codex CLI 时，向导会显示手动配置用的 TOML，按提示合并，保留已有配置。
8. 完全退出后重新打开 Codex，让 MCP 配置重新加载。

CC Switch 导入是**一次性复制**。之后在 CC Switch 切换供应商，不会自动切换 MCP 使用的图片接口；需要重新运行 `init` 导入。不支持导入 ChatGPT OAuth 登录令牌或代理占位密钥，导入失败时可选择手动配置。

## 第三步：添加一句推荐提示词

将下面这句复制到 Codex 的个性化提示词：

```text
需要生成或编辑图片时，优先使用 codex-image-gen-plus-plus MCP。
```

初始化会打印这句建议，但不会自动修改你的个性化设置。

## 第四步：检查连接与生成图片

先在终端检查连接：

```sh
codex-image-gen-plus-plus doctor
```

`doctor` 读取模型列表，不发起收费生图。接着在 Codex 对话里发送：

```text
查看当前可用的图片模型，选一个生成一张方形水彩狐狸插画，保存到当前工作区的 output/imagegen/，生成后展示给我。
```

这一步会发起真实生图并消耗额度。可用模型以当前密钥返回的列表为准，不要固定假设存在 `gpt-image-2` 或 `gpt-image-2.5`。如果列表里只有 `gpt-image-2.5-flare`、`gpt-image-2.5-sunburst` 等名称，应使用完整模型 ID，名称本身不能证明质量、速度或价格差异。

图片由 MCP 直接保存到 Codex 指定的绝对路径，不覆盖已有文件。没有指定位置时，推荐使用当前工作区的 `output/imagegen/`；若开启工作区保护，该位置还必须位于配置的根目录内。

生图较慢时会返回任务编号，由 Codex 使用 `get_job` 继续查询。不要因暂时没有图片就重复提交，以免重复计费。

## 如何编辑图片

把参考图交给 Codex，或告诉它当前工作区内的参考图路径，例如：

```text
把这张图片的背景改成傍晚海边，保留主体和构图，另存到当前工作区的 output/imagegen/，完成后展示结果。
```

Codex 会调用 `edit_image` 上传参考图。能否编辑、支持哪些尺寸及参数，取决于所选模型和接口支持情况；参考图和提示词会发送到配置的图片服务。

## 常见问题

| 问题 | 处理方式 |
| --- | --- |
| Codex 里看不到 MCP | 确认 `init` 已完成注册，完全重启 Codex；终端可运行 `codex mcp list` 检查是否存在 `codex-image-gen-plus-plus`。 |
| 显示已注册，但工具启动失败 | 检查安装目录或 Node 路径是否发生变化，重新运行 `init` 注册。 |
| CC Switch 导入失败 | 确认当前启用的是 API Key 方式的 Codex 供应商；自定义数据目录需在向导中填写。也可改为手动输入。 |
| 返回 `401` 或 `403` | 检查 Key 状态、余额和分组生图权限；地址或 Key 更换后重新运行 `init`。 |
| 模型不存在或没有权限 | 重新获取模型列表，使用准确的完整 ID，并确认 Key 绑定的分组支持该模型。 |
| 输出路径被拒绝 | 使用工作区内的绝对路径；开启保护时，目标必须在配置的根目录内，输出目录不能通过符号链接指向外部。 |
| 超时或任务中断 | 先查已有任务、输出文件和本站使用记录，再决定是否重新生成；MCP 不会自动重试或切换模型。 |

## 升级与任务记录

升级时重新执行上面的安装命令。需要更改地址、密钥、导入配置或工作区范围时，再运行 `init`，完成后重启 Codex。

MCP 启动时默认清理 7 天前已完成或失败的任务记录，保留图片和运行中的任务。也可手动预览或清理：

```sh
codex-image-gen-plus-plus jobs clean --days 30 --dry-run
codex-image-gen-plus-plus jobs clean --days 30
```

`--days 30` 只选择 30 天前的已完成或失败记录；`--dry-run` 只显示可清理数量，不删除。清理的是包含提示词和路径的任务记录，不是生成图片。

## 本文核对范围

- 依据 [codex-image-gen++ v0.2.0](https://github.com/Anti2077/codex-image-gen-plus-plus/releases/tag/v0.2.0) 的源码、安装包和中英文说明更新；
- Windows、macOS、Linux 的 Node.js 22/24 构建与命令检查已通过；这不代表每个平台的系统密钥库授权和所有上游图片模型都已实测；
- 当前密钥的模型可见性、分组权限和真实生图结果，以实际请求为准；
- 核对日期：2026-10-18。
