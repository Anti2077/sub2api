# 正式兑换页隔离预览

从仓库根目录运行：

```sh
pnpm --dir frontend exec vite --config ../artifacts/redeem-ceremony-preview/preview.config.mjs
```

打开 http://127.0.0.1:4180/ 。加载真实 RedeemView、动效组件、站点样式和中文文案；布局及 API/store 仅在此独立 Vite 配置内替换为本地模拟，不读真实凭证、不调用后端。

普通代码增加 $20，CONCURRENCY 增加并发，SUBSCRIPTION 模拟订阅，FAIL 模拟兑换失败。支持切换深浅色。
