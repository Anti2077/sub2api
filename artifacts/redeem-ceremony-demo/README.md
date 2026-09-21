> 历史光点方案，最终实现已改为手撕票据。此目录仅保留设计迭代参考。礼花脚本引用相邻 redeem-ticket-demo/confetti.js。

# 兑换流光交互原型

独立 HTML/CSS/Canvas demo。打开 index.html，或在本目录启动静态 HTTP 服务。

- 模拟余额从 $128.00 增至 $148.00，不调用 API。
- 双侧扫光 → 中心聚光 → 曲线汇入 → 余额增长 → 双侧 School Pride 礼花。
- 可重播、重置、两倍慢速观察；遵循 prefers-reduced-motion。
- 仅用于视觉评审，尚未接入 Vue 业务页面。

礼花来自 https://www.kirilv.com/canvas-confetti/src/confetti.js ，对应页面的 School Pride 示例；仅本地保存礼花脚本，不包含示例页统计脚本。颜色调整为蓝紫、白、金，连续喷射约两秒。
