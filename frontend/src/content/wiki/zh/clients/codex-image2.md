# 修复 Codex 的 gpt-image-2 调用

本站当前所有分组都已提供生图模型 `gpt-image-2`。本文中提到的 Image2，指的就是模型 ID `gpt-image-2`。

不过，Codex 能正常使用聊天模型，不代表它一定会自动通过当前的 CC Switch 供应商调用图片接口。`codex-image2` Skill 可以直接调用 OpenAI 兼容的图片生成接口，但它需要获得正确的 API 地址和密钥。最省事的解决方法，是让你自己的 Codex 完成安装和适配，并在每次生图时动态读取 CC Switch 当前启用的 Codex provider。

你不需要手动寻找配置文件，也不要把 API Key 粘贴到聊天中。把下面的完整提示词发给 Codex，让它自主完成即可。

> 这段提示词允许 Codex 在安全检查和 dry-run 通过后，生成一张测试图。真实请求只允许生成一张图片，可能产生少量模型用量。

## 复制下面的提示词给 Codex

```text
请帮我安装、配置并测试这个 Skill：

https://github.com/fengfengzhidao/codex-image2-skill

目标：让 codex-image2 自动使用 CC Switch 的“当前 Codex provider”，读取 API 地址和密钥，并生成一张测试图。

请自主完成整个流程，不要要求我把 API Key 粘贴到聊天中。

具体要求：

1. 首先识别当前环境：

   - 操作系统；
   - CPU 架构；
   - 当前 Codex 客户端及 Codex Home；
   - CC Switch 是否安装；
   - CC Switch 的实际配置或数据库位置；
   - codex-image2 是否已经安装。

   不要照搬其他用户的路径，也不要假设一定是 macOS。

2. 安装 Skill：

   https://github.com/fengfengzhidao/codex-image2-skill

   安装仓库中的 codex-image2 目录。若已安装，先检查现有内容和本地修改，只补充缺失配置，不要盲目覆盖。

3. 完整阅读 SKILL.md 和相关源码，确认其实际需要的配置。预期环境变量为：

   - CODEX_API_URL
   - CODEX_API_KEY

4. 只读检查 CC Switch 的当前 Codex provider。根据实际版本和数据结构寻找：

   - 当前 provider 标记；
   - API base URL；
   - OPENAI_API_KEY 或等价认证字段。

   某些版本可能使用 SQLite，某些版本或平台可能使用其他配置形式。请先检查实际结构，再选择可靠的读取方法。

5. 建立平台适配的安全启动方式：

   - macOS：使用适合当前架构的 Darwin 可执行文件和 Zsh/Bash 启动器；
   - Windows：使用对应的 EXE 和 PowerShell 启动器；
   - 其他系统：先检查仓库是否提供兼容二进制；没有时检查源码和现有工具链，再选择风险最低的方式。不要伪装成已支持。
   - 每次调用时重新读取 CC Switch 当前 provider，以便切换 provider 后自动生效。
   - 将 API 地址临时映射为 CODEX_API_URL。
   - 将密钥临时映射为 CODEX_API_KEY。
   - 环境变量只传给本次 codex-image2 子进程，执行结束后不保留。

6. 安全要求：

   - 不得在聊天、日志、命令参数或错误报告中显示密钥；
   - 不得把密钥写入 Skill、项目文件、shell profile、PowerShell profile 或 Codex 全局配置；
   - 不得打印包含密钥的完整数据库记录；
   - 不得启用可能输出变量的 shell tracing；
   - 不得修改 CC Switch 数据库；
   - 不得切换 provider；
   - 不得改变 Codex 的模型、路由、权限、记忆或其他无关配置；
   - 当前 provider 缺失、不唯一或配置不完整时，明确报错并停止，不能擅自选择其他 provider。

7. 更新本地 SKILL.md，使 Codex 后续调用 $codex-image2 时优先使用上述动态启动方式，同时保留原始环境变量配置作为兼容方案。

8. 验证配置：

   - 检查启动器语法；
   - 运行 Codex Skill 的 quick_validate；
   - 验证缺失配置时能够安全失败；
   - 先执行一次 --dry-run，确认模型、参数和最终图片接口正确；
   - 确认 dry-run 不联网、不生成图片、不消耗额度。

9. dry-run 通过后，我授权你直接进行一次真实测试生图请求。无需再次询问，但仅允许生成一张，不要批量生成，也不要改用其他收费服务。

   测试图要求：

   一只透明玻璃瓶放在安静的深色摄影棚中，瓶内悬浮着明亮的蓝色星云，电影感产品摄影，玻璃边缘清晰，构图居中，无人物、无文字、无 Logo、无水印。

   参数：

   - model：gpt-image-2
   - size：1024x1024
   - quality：auto
   - n：1

10. 将图片保存到当前项目合适的用户输出目录，不要覆盖已有文件。生成后：

   - 检查实际文件格式；
   - 检查实际像素尺寸，不要只相信请求参数；
   - 目视检查主体、构图、水印、文字和明显伪影；
   - 在最终回复中直接展示图片；
   - 报告模型、质量、请求尺寸、实际尺寸和输出文件；
   - 不得显示 API Key。

执行过程中请简要解释关键命令及参数的作用。遇到可以安全处理的问题时自行处理，只有涉及密钥暴露、破坏性操作或需要改动无关配置时才停下来询问。
```

## Codex 会替你完成什么

正常情况下，Codex 会依次完成：

1. 识别操作系统、CPU 架构、Codex Home 和 CC Switch 的实际安装位置；
2. 安装或检查 `codex-image2` Skill，而不是直接覆盖现有本地修改；
3. 只读获取 CC Switch 当前启用的 Codex provider；
4. 将当前 provider 的 API 地址和密钥仅传给本次生图子进程；
5. 运行语法检查、`quick_validate` 和不联网的 `--dry-run`；
6. 使用 `gpt-image-2` 生成一张 `1024x1024` 测试图；
7. 检查实际图片格式、尺寸和画面质量，并直接展示结果。

如果动态启动方式配置成功，CC Switch 切换到其他 Codex provider 后，后续调用也会重新读取当前 provider，不需要重复把 API Key 写入环境变量或配置文件。

## 后续如何生图

配置成功后，可以直接对 Codex 说：

```text
使用 $codex-image2 生成一张图片：
一只橘猫坐在月球表面，远处可以看到地球，电影感灯光，无文字、无 Logo、无水印。
```

Codex 会根据当前系统选择对应的原生可执行文件，并把图片保存到当前项目合适的输出目录中。

## 如果 Codex 停止并要求处理

以下情况应当停止，而不是让 Codex 猜测或擅自选择：

- 找不到 CC Switch；
- 当前 Codex provider 缺失或同时存在多个“当前”provider；
- 当前 provider 缺少 API 地址或认证字段；
- 当前系统没有兼容的二进制，也没有安全可用的构建工具链；
- 继续操作可能暴露 API Key、覆盖本地修改或改变无关配置。

如果只是权限、可执行文件标记或输出目录不存在等可安全处理的问题，可以让 Codex 自行修复并继续。

## 本文核对范围

- 本站当前所有分组均提供 `gpt-image-2`；
- `codex-image2` 上游仓库包含 Skill 源码以及 Windows x64、Windows ARM64、macOS Intel 和 macOS Apple Silicon 原生可执行文件；
- 上游 Skill 使用 `CODEX_API_URL` 和 `CODEX_API_KEY` 调用 OpenAI 兼容图片接口；
- CC Switch 动态 provider 读取方式由用户的 Codex 根据实际系统、版本和数据结构建立，不假设固定路径；
- 核对日期：2026-09-05。
