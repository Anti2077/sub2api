# 手撕兑换票据设计原型

独立静态演示，不调用真实 API；任意非空代码均模拟增加 $20。正式组件及真实请求状态请使用相邻 `redeem-ceremony-preview` 的隔离预览。

支持向右下方拖拽撕票、未撕断松手回弹、余额到账和彩色彩带。正式实现额外包含接口等待、无效码按压震动、盖章入场、中英文、深色模式和减少动态效果。

confetti.js 来自 https://www.kirilv.com/canvas-confetti/src/confetti.js ，ISC 授权见 LICENSE.canvas-confetti。
