# 使用本站服务器配置 RustDesk

这篇教程会把 RustDesk 的 ID/中继服务器切换到本站提供的服务器。完成后，把 RustDesk 首页显示的 **本机 ID（远程码）** 发给可信的远程维护人员即可；对方发起连接时，仍需你按 RustDesk 的提示确认或提供本次连接所需的一次性密码。

## 开始前

- 安装并打开 [RustDesk](https://rustdesk.com/)；下方截图来自 macOS 版，其他系统的入口名称或位置可能略有不同。
- 准备好本站提供的服务器配置串。它仅用于导入服务器设置，不是 Sub2API 的 API Key。
- 如果 RustDesk 已配置其他自建服务器，导入会替换当前的 ID/中继服务器设置；先确认是否需要保留原配置。

## 第一步：进入网络设置

打开 RustDesk 右上角菜单，进入“设置”→“网络”，点击“ID/中继服务器”。

<figure>
  <img src="/wiki/images/rustdesk/menu.png" alt="RustDesk 主窗口右上角的菜单按钮" width="720" height="200">
  <figcaption>图 1：从主窗口右上角打开菜单。</figcaption>
</figure>

<figure>
  <img src="/wiki/images/rustdesk/network.png" alt="RustDesk 设置中的网络页面，ID/中继服务器入口位于页面顶部" width="1600" height="780">
  <figcaption>图 2：依次进入设置、网络、ID/中继服务器。</figcaption>
</figure>

## 第二步：导入本站配置

点击下面代码块的“复制”，将**整段配置串**复制到剪贴板，不要加入空格或换行：

```text
9JSP3dnRVdVVnRlRwJkdnJTRGR1VDpHOzhVVzNEVLNmMrJUOWZHNzo0MGFVQIJiOikXZrJCLiIiOikGchJCLioXe45yN3AjMpRnbh5yazVGZrNXdyJiOikXYsVmciwiI6lHeuczNwITa05WYus2clR2azVnciojI0N3boJye
```

回到“ID/中继服务器”弹窗，点击右上角的**剪贴板图标（导入服务器配置）**。导入后核对：ID 服务器和中继服务器均显示为 `ruskdesk.anti2077.xyz`，API 服务器留空，Key 字段会自动填入服务器公钥。这里的域名拼写就是 `ruskdesk`，不要自行改成 `rustdesk`。

<figure>
  <img src="/wiki/images/rustdesk/config.png" alt="RustDesk ID/中继服务器弹窗，右上角有导入图标，ID 与中继服务器均为 ruskdesk.anti2077.xyz" width="1180" height="760">
  <figcaption>图 3：导入后核对服务器地址，再点击“确认”。</figcaption>
</figure>

## 第三步：保存并发送本机 ID

点击“确认”保存设置，返回 RustDesk 首页。等待本机 ID 正常显示，再把这个 **ID** 发给可信的远程维护人员。连接请求到来时，确认对方身份后再批准；不要把一次性密码发到公开群聊或工单中。

## 没有成功导入？

- **点击导入后没有变化**：确认整段配置串已经复制进剪贴板，再重新点击导入图标。不要把配置串填进单个服务器字段。
- **服务器地址不符**：先不要保存，重新复制本站配置并导入；不要手动修改域名拼写。
- **本机 ID 不显示或显示离线**：检查网络连接，并确认已点击“确认”保存。仍无法连接时，把错误提示发给维护人员，不要公开一次性密码。

操作入口与导入方式已于 2026-09-29 依据提供的 RustDesk macOS 截图和 [RustDesk 官方客户端配置文档](https://rustdesk.com/docs/en/self-host/client-configuration/)核对。截图所用客户端的具体版本未记录，Windows、Linux 和新版界面以实际显示为准。
