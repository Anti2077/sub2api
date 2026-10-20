# 活动卡券与 Token 农场本机验收

本分支提供三个用户入口：福利中心、卡券中心、Token 农场；管理员入口为活动卡券管理和农场管理。卡券为服务端快照资产，抽奖、兑换码、农场兑换统一走发卡账本。日卡激活后连续 24 小时，Token 卡按规范化输入、输出、缓存 Token 原子抵扣，倍率卡取更低结果；图片、工具和媒体费用保持原规则。

## 本机地址

- 前端：`http://127.0.0.1:18786`
- 后端健康检查：`http://127.0.0.1:18086/health`
- 本地模拟上游：`127.0.0.1:19086`，只返回固定协议响应，不访问真实供应商
- 测试账号：`farmer@rewards.test` / `admin@rewards.test`，密码均为 `RewardsLocalDemo123`

测试 Compose 只绑定回环地址，PostgreSQL/Redis 端口为 15486/16386，数据目录在 `/private/tmp/rewards-local`，不读取生产配置。

## 页面演示路径

普通用户登录后依次打开 `/incentives` 完成签到和抽奖，`/cards?tab=redeem` 输入管理员在 `/admin/reward-cards` 生成的兑换码并撕开票根，再到 `?tab=bag` 手动激活。`/farm` 可播种、收获、串门、摘取和用晶能兑换卡券。管理员可在 `/admin/incentives` 配置卡券奖池，在 `/admin/reward-cards` 管理模板、发卡、兑换码和规则，在 `/admin/farm` 管理作物和商店。

## 扣费证据

`/private/tmp/rewards-local/http-results.json`：无卡 `$0.00238000`；免费日卡 `$0`；倍率卡 `$0.00170000`；Token 卡全覆盖 `$0`、部分覆盖 `$0.00079333`、耗尽恢复 `$0.00238000`；流式全覆盖 `$0`；12 并发双 Key 原始 `$0.02856000`、实付 `$0.02380`、覆盖 3000 Token。每一行同时核对上游 `$0.0068`、输入 800、输出 500、缓存 200。

`/private/tmp/rewards-local/extended-results.json`：OpenAI、Anthropic、Gemini 普通及流式各两次；在途撤销和到期快照；上游错误无用量；无事件流中断不消耗卡额度或余额。HTTP 幂等重试与进程重启回放已通过：相同请求只转发一次、不同请求体返回 409、20 个并发请求只调用一次模拟上游，重启后回放不会再次调用上游。

新增 PostgreSQL 集成测试覆盖随机 Token 独立预期、卡额度并发原子性、余额/卡账本回滚、农场保护期、40% 偷取上限、收获竞争、商店库存和限兑幂等；`go test -race -tags=integration ./internal/repository -run 'TestRewards|TestFarm'` 通过。另增加 `TestFarmExpansionPreservesPlantedSnapshots`，验证 12→16 格保留作物、快照和成熟时间，第 16 格可种、第 17 格拒绝，缩回 12 格仍保留已种的第 16 格；单独 race 通过。

## 检查结果与边界

- 后端新增功能聚焦集成：8 项通过（含 durable gateway claim）；新增并发 race：通过。
- 前端 Node 24.22.0 / pnpm 9.15.9：全量 401 文件、3,065 测试通过；类型检查、国际化、lint、生产构建通过。新增卡券中心筛选、激活预览、重复提交、兑换刷新、农场成长查看与兑换重试测试均通过。
- 默认后端测试：13,680 通过、14 跳过；当前 unit 全量：23,471 通过、18 跳过、1 失败。既有 `TestInflightEstimate_AccountMappingNoDBAndBoundedMemory` 全量运行超过 8 MiB 断言（8,554,896 bytes vs 8,388,608），串行重跑仍失败（8,484,672 bytes），单独重跑通过；该问题尚未解决，整体完成门槛未通过。本轮 integration 全量：14,457 通过、16 跳过；奖励／农场 race 聚焦：8 项通过，扩格 race 另 1 项通过。
- 已检查桌面、375px 手机和浅色/深色农场页面：当前为 12 格露天菜地，手机 `innerWidth=375`、`body.scrollWidth=367`（视口 375，无横向溢出）；真实后端播种、成熟收获、晶能扣减、兑换确认、库存变化已在浏览器完成。
- 用户页面已重构为露天草地、农舍、风车、栅栏、池塘和连续翻耕菜地；移除盆沿与箱体，田地采用独立菱形命中区域，避免斜视矩形叠层串格。手机 12 格中心命中逐格核对，实际收获第 6 格后晶能 100→110，并在第 1 格播种显示 1 小时成长计时。原 6 格作物与计时未重置；默认通过迁移扩为 12 格，后台上限 16 格。背景与作物 WebP 由 `codex-image-gen-plus-plus` MCP 生成并保留 SVG 回退，来源见 `frontend/public/assets/rewards/IMAGE_PROVENANCE.md`。
- 票券卡包与签到→抽奖→入卡包流程、共享导航、状态标签、对账金额、进度条和 reduced-motion 样式已接入。农场可点击成长中作物，在底部查看名称、剩余时间和预计收益。
- 本轮浏览器再次收获第 7 格，晶能 110→120；兑换前核对模型、分组、Token、有效期和扣减预览，取消未扣减，确认后 120→100、库存 98→97，并产生卡券 #90，回执可进入卡包。倍率卡预览明确显示分组 #2、目标 0.25×、连续 24 小时，取消未激活。串门与返回路径通过，拜访时不显示自己的种子选择；减少动态效果模拟下农场无持续动画，验证后已清除模拟设置并恢复浅色桌面展示。
- 前端不确定网络失败会保留同一次操作的幂等键，成功后下一次操作使用新键；此保留仅在当前页面生命周期内，不宣称刷新后仍保留前端操作键。服务端网关重放使用独立持久化账本。
- 本地模拟未证明真实供应商的生产可用性，也未覆盖媒体供应商真实成本；这些仍需独立生产前验收。


## 本轮页面证据

最新页面截图保存在 `/private/tmp/rewards-local/screenshots/`：`farm-polished-desktop.png`、`farm-polished-mobile-375.png`、`farm-polished-dark-desktop.png`、`farm-polished-dark-375.png`、`cards-polished-mobile-375.png`、`benefits-polished-mobile-375.png`。前轮其他功能证据：`benefits-polished-desktop.png`、`cards-polished-desktop.png`、`admin-farm.jpg`。

最后一次全量 `pnpm test:run`：401 文件 / 3,065 项通过，0 失败；`pnpm typecheck`、`pnpm lint:check`、`pnpm build` 均通过，构建内国际化检查 3 项通过。扩格和农场并发检查日志：`backend-expansion-race.log`、`backend-race-farmland.log`，位于 `/private/tmp/rewards-local/`。上述前端结果以本次最后执行为准，早期 `ui-*-field12-final.log` 属于之前的 399 文件版本。

## 土地点击与对齐修正（2026-10-19）

- 移除土地点击上浮、发光和作物持续摇摆；选中改用独立细描边，收益反馈单独浮动。成功／错误提示不再占据页面流，操作后不会把农场向下推。
- 田格坐标、土地图像和菱形命中区域统一按 466×311 比例；移动端纵向间距也随场景宽度缩放。作物缩小并固定脚点，点击作物、成熟标记和土地均定位到所属田格。忙碌时保留节点与焦点，事件处理继续阻止重复提交。
- 浏览器在真实本机后端验证：第 8 格播种后进入一小时成长；点击其成长状态不发起资产操作。点击第 1 格作物收获，晶能 130→140，刷新前后场景及全部 12 块土地矩形完全相同。桌面全部 12 个土地中心及现有作物中心命中所属格；375px 下 12 个土地中心同样逐格匹配，无横向溢出（scrollWidth 367 ≤ 375）。
- 深色模式验证 `html` filter 为 `none`，仅背景图应用亮度／饱和度滤镜。已恢复浅色与正常桌面视口，服务继续运行。
- 本轮聚焦 FarmView 测试 5 项通过、0 失败、0 跳过，包含延迟请求与刷新期间保持土地节点、重复点击仅提交一次的回归；类型检查、全量 lint、生产构建均通过，构建中的国际化测试 3 项通过。全量前后端结果和既有后端 unit 失败沿用上文边界，本轮未改动扣费逻辑。
- 新截图：`farm-aligned-desktop.png`、`farm-aligned-mobile375.png`、`farm-aligned-dark.png`，位于 `/private/tmp/rewards-local/screenshots/`。

## 统一矢量透视与稳定田格（2026-10-19，最新）

本节取代前述农场贴图与 466×311 田格的视觉实现描述；历史扣费、发卡与服务端验收边界不变。

- 农场背景、土地、作物均改为 SVG，场景不再加载之前的 WebP 素材；原素材与来源记录保留。使用 960×740 的统一世界坐标，地面两个方向均为 2:1 投影，房屋、栅栏、道路与 12 格连片田地使用相同轴向；12–16 格按地块数自动布局。
- 每格四株作物按地面深度排列，幼苗、晶能芽、晶簇、花与月光藤共用画风。土垄、田埂和落地阴影固定；操作仍走现有服务端接口，本轮未修改计费或农场资产规则。
- 桌面及 375×812 手机逐格检查，12 个土地中心全部命中所属格；375px 下 document.scrollWidth=367，无横向溢出。真实坐标点击切换生长详情前后，全部 12 格矩形完全相同；没有土地位移动画或变形。自动化定位器可能自动滚动页面，故稳定性复核使用真实坐标点击，排除自动滚动影响。
- 深色模式仅对完整农场世界统一应用 brightness(.78) / saturate(.82)，html filter=none。检查完成后恢复浅色、默认桌面视口，保留用户演示页与本机服务。
- FarmView 聚焦回归 5 通过、0 失败、0 跳过，覆盖延迟操作／刷新保持节点及双击防重；全量前端 lint、类型检查、生产构建通过；国际化 3 通过。构建保留既有 chunk 大小与混合导入提示。没有重跑全量后端计费验收，不据此宣称整个功能计划已完成。
- 当前普通用户田地均处于正式小时级生长中，本轮未改成熟时间、未重置测试资产，也未再次执行真实收获；收获刷新节点保持由回归测试及上轮真实收获证据覆盖。
- 新截图：`/private/tmp/rewards-local/screenshots/farm-vector-desktop.png`、`farm-vector-mobile375.png`、`farm-vector-dark.png`。


## 晨光材质与道路边界对齐（2026-10-19，最新）

- 修正菜地中心和原道路中心约 24 world px 的纵向偏移。新增共享 `farmGeometry`，田格、道路、入口与围栏统一推导，随 12–16 格布局变化；不再分别手调位置。
- 静态左上晨光、树冠与屋顶明暗、土壤沟垄、田埂高光、叶脉露珠、水纹与落地阴影；土壤／作物使用唯一渐变 ID。未添加持续动画、图片依赖或后端资产变更。
- 真实页面桌面、375×812 手机及深色截图已核对。手机 12 格中心命中全部正确，scrollWidth=367，无横向溢出；坐标点击切换详情前后 12 格矩形相同。按实际 SVG 土壤顶面及道路 40px 描边独立核算，12 格四边净间距为 3.197/6.135/6.135/3.197 world px，16 格为 2.981/5.552/5.552/2.981，均未压到道路。16 格仅验证几何，未变更数据库或宣称完成浏览器 16 格资产操作。
- FarmView 聚焦回归 5 通过、0 失败、0 跳过；本轮变更文件 eslint 通过，国际化 3 通过，vue-tsc 类型检查及最终生产构建通过（1195 modules）。保留既有混合导入、包体积与 Browserslist 提示；全功能及后端 unit 未通过项继续沿用上文边界。
- 截图：`/private/tmp/rewards-local/screenshots/farm-light-aligned-desktop.png`、`farm-light-aligned-mobile375.png`、`farm-light-aligned-dark.png`。检查后已恢复浅色与默认桌面尺寸，演示页和服务继续运行。

## 卡模板批量管理（2026-10-19）

- 类型／发布状态组合筛选，名称、ID、模型与分组搜索；每页 12 条、当前页全选、跨页保留选择，切换筛选清除选择。复制保留权益配置但清除 ID／版本并设为未发布草稿，保存前可取消。
- 批量发布／下架确认后逐条调用原版本校验保存接口，展示每项成功／失败；失败项保持选中并在刷新后重试。保存到刷新结束持续锁定操作，已处于目标状态的项不重复保存。已发卡、兑换码和商店的权益快照不改变。
- 真实本机管理员页面搜索“日卡”，两张模板批量下架再发布均成功 2、失败 0；最终 #2 v5、#1 v6 均恢复已发布。浏览器确认复制默认未勾选发布并取消，61 条模板分页及跨页保留 12 项选择通过，实际类型筛选只显示对应类型。未新增卡券或改变余额。
- Node 24 / pnpm 9.15.9 聚焦 3 文件、14 项通过，0 失败、0 跳过；其中 TemplateManager 新增 7 项覆盖筛选、多页选择、取消、版本冲突后的部分重试、刷新期间防重复与复制草稿。改动文件 eslint、国际化、类型检查、最终生产构建及 git diff --check 通过。构建日志：`/private/tmp/rewards-local/templates-build.log`。本轮未修改计费后端，未重跑全量扣费及全量前后端测试，整体计划既有 unit 失败仍保留上文边界。
- 桌面浅／深色与 375×812 手机实际截图检查通过；手机 innerWidth=375，documentElement/body scrollWidth=367，无横向溢出。截图：`templates-bulk-desktop.png`、`templates-bulk-dark.png`、`templates-bulk-mobile375.png`，位于 `/private/tmp/rewards-local/screenshots/`。恢复浅色默认桌面尺寸并保留 `http://localhost:18786/admin/reward-cards` 已登录管理员页面，服务继续运行。未合并、推送或生产部署。

## 批量管理补全与并发验收（2026-10-20，最新）

### 可操作入口与行为

- `http://localhost:18786/admin/reward-cards`：模板按类型／发布状态筛选，名称／模型／分组搜索，12 条分页、跨页多选、批量发布／下架、复制未发布草稿；逐项回执，失败项保留重试。
- 同页“发放与撤销”：去重 1–100 个用户 ID，每人一张卡，发卡／撤销填写原因；按类型、状态、来源筛选已发卡。批量发卡与生成兑换码的未确定请求在本标签页 sessionStorage 按管理员账户保留原请求编号，刷新后重试沿用编号；不复制认证令牌。存储不可用时禁止发送新批次。
- 同页“生成兑换码”：选择已发布模板，生成 1–100 个、有效期 1–365 天；最近 500 条支持筛选／搜索／分页、多选复制／停用／恢复。已使用或已过期码不能恢复，旧兑换管理的批量更新也不能重新打开已用码。
- `http://localhost:18786/admin/farm`：商店类型／发布状态／库存筛选及批量上下架；上下架只修改发布状态，版本冲突单项失败，不回写库存／价格／限兑／卡券快照。作物支持搜索、状态筛选与批量启停，保存规则后生效；已种作物保留原快照。晶能与审计记录支持分页搜索。
- `http://localhost:18786/admin/incentives`：奖品类型／状态筛选、批量启停／移除，相对权重概率预览；保留 2–8 个奖项约束。各活动独立保存，不覆盖另一活动未保存表单。奖励／机会／周期记录为只读查询。配置／编辑器切换、刷新、离开页面均提醒未保存改动，处理中锁定重复提交。

### 本轮最终检查

- Node 24.22.0 / pnpm 9.15.9：前端全量 **405 文件、3,088 项通过、0 失败、0 跳过**；全量 lint、vue-tsc 类型检查、国际化 3 项及生产构建通过。构建仅保留既有 Browserslist／包体积提示，无锁文件更新。日志：`management-frontend-final.log`、`management-lint-final.log`、`management-build-final.log`。
- Go 1.27.2：`go test -race -json -tags=integration ./internal/repository -run 'TestRewards|TestFarm|TestRedeemCodeRepoSuite' -count=1` 最终 **38 个测试及子测试通过、0 失败、0 跳过**，覆盖卡券／农场／旧兑换及管理并发；非全量后端 integration。专项奖励费用／旧批量兑换服务 unit **4 项通过**，后端构建通过。日志：`management-integration-final.json`、`management-backend-unit-final.json`、`management-backend-build.log`。
- 审计回滚测试通过：只针对测试夹具安装临时 PostgreSQL 触发器，令兑换码状态更新后的审计 INSERT 报错；断言状态仍为 unused 且审计数不变，去掉故障后可正常停用／恢复，测试清理触发器和函数。未改变生产表约束。最初使用无效 actor ID 的故障假设不成立（actor_id 无外键），已用真实故障注入替换并完整重跑。
- 真实 HTTP 管理验收 **5 组通过**：12 路相同批次生成仅 3 个码、12 路发卡仅 1 张、12 路撤销仅 1 条审计；模板下架后重放仍返回原回执；停用／恢复／实际兑换及新旧入口拒绝重开已用码；8 轮兑换与停用竞争均唯一胜出、重复发卡 0。证据：`management-http-results.json`、`management-http-concurrency.log`。
- 当前服务真实网关 HTTP 扣费回归 **8 组通过**：无卡、零余额免费日卡、倍率卡、Token 全抵／部分／耗尽、流式、双 Key 12 路共享额度。使用独立 Decimal 预期值；原始应付 = 抵扣 + 实付。部分抵扣样例：`0.00238000 = 0.00158667 + 0.00079333`；12 路共抵 3,000 Token：`0.02856000 = 0.00476000 + 0.02380000`。上游真实记录每次仍为 `0.0068`。证据：`http-results.json`、`management-http.log`。
- 浏览器实际发卡两人后仅撤销本次 #92/#93；生成分组倍率码 #17/#18，停用再恢复；作物和奖品启停保存后恢复原配置。商店 #1 下架再上架 v4→v6，数据库验证库存 97、价格 20、每日／累计限兑 3／10、权益快照完全相同。证据：`management-shop-before.json`、`management-shop-after.json`。
- 桌面浅／深色、英文、375×812 手机已检查；模板／兑换码／发卡／商店／作物／奖池／记录无横向溢出（375px 视口，document/body 367px）。最终恢复中文、浅色、默认桌面，已登录管理员停留在模板管理。截图目录 `/private/tmp/rewards-local/screenshots/`：`management-final-desktop.png`、`management-templates-mobile-375.png`、`management-codes-dark-en-375.png`、`management-farm-mobile375-dark.png`、`management-prizes-mobile375-dark.png`。

### 验收边界

本轮聚焦检查已通过，服务继续运行；所有 HTTP 供应商响应来自独立本地协议模拟，无真实供应商调用，不代表生产验收。前述全量后端 unit 的 `TestInflightEstimate_AccountMappingNoDBAndBoundedMemory` 内存阈值失败尚未解决，本轮未重跑全量后端默认／unit／integration，不将整个活动卡券／农场计划标记为全面完成。仅修改任务工作区 `Anti2077/rewards-cards-farm`，未合并、推送或生产部署。

## 管理界面深度优化（2026-10-20，最终补充）

本节补充本次后台管理范围的最终结果；前述计费验收边界继续有效。

- 模板、已发卡、兑换码、商店、作物及奖品统一支持当前页全选、全部筛选结果选择、反选本页、跨页保留选择、12／24／48 条分页及重置筛选。修改搜索或筛选会清除选择；仅切页或调整每页数量不会清除。界面明确显示选择数量和操作覆盖的页范围。全部筛选结果仅指当前已加载的数据，不扩展到服务器未返回的历史记录。只读审计、晶能和福利记录仍不提供资产批量操作。
- 批量发卡增加可折叠的真实用户选择器：按邮箱／用户名服务端搜索，数字精确查询用户 ID，状态筛选、12 条服务器分页及跨次查询保留勾选；与手工 ID 输入双向同步，每批最多 100 人。开放规则的试点用户也复用选择器。查询失败可重试，未展开时不加载用户。
- 模板分组和福利规则使用搜索多选，支持选中全部当前搜索匹配项、移除单项与清空；保留未知既有选项。移除嵌套 button，补充禁用与 Escape 收起行为。所有同状态批量操作禁用，避免无意义版本更新。
- 作物和奖品校验覆盖全部配置，包括其他页及当前筛选之外的项目，校验名称、重复 ID、整数范围、金额、卡模板发布状态及奖池整体约束；无效项显示数量和定位按钮，可清除筛选并跳至对应页。服务端原校验仍保留。
- 浏览器实测模板 67 项全选后将每页改为 24，再反选本页，剩余 43 项；兑换码 28 项全选、反选 12 项后剩 16，改为每页 24 保持选择。用户 ID 2 与 3 跨次搜索勾选同步为 `2, 3`，未提交发卡。
- 浏览器将作物小时数临时改为 0 并筛选隐藏，保存被阻止且可一键定位；奖品权重 5 临时改为 0 后筛选至另一个奖品，仍阻止保存并可定位。两项均在表单中恢复原值并刷新，没有保存无效配置。规则分组搜索“本机”后全选 1 项，再清空恢复原状态；模板分组搜索后取消编辑，均未保存配置。
- 375×812 手机、英文、深色检查通过。视觉检查发现用户查询栏在窄屏被挤成逐字换行，已改为手机纵向排列；实测搜索框宽 267px，document/body scrollWidth=367 ≤ 375，没有横向溢出。桌面沿用横向筛选布局，DOM 嵌套 button 数为 0。最后恢复中文、浅色和默认桌面，保留已登录管理页。
- Node 24.22.0／pnpm 9.15.9 全量前端 **408 文件、3,100 项通过，0 失败、0 跳过**。最终提示文案及手机布局修正后，管理组件和国际化聚焦 **8 文件、37 项通过，0 失败、0 跳过**；最终 `pnpm lint:check`、含 vue-tsc 和国际化 3 项的 `pnpm build`、`git diff --check` 均通过。日志：`management-depth-frontend.log`、`management-depth-final-focused.log`、`management-depth-final-lint-check.log`、`management-depth-final-build.log`。构建仅保留既有包体积／Browserslist 提示。
- 此轮深度优化未修改后端计费规则；管理后端 38 项 race 集成、5 组管理 HTTP、8 组扣费 HTTP 证据沿用上一节，不宣称重新运行全量后端。健康检查确认后端 `/health` 返回 ok、前端页面 HTTP 200，本机隔离环境身份检查通过。未合并、推送或生产部署。
- 最终截图：`/private/tmp/rewards-local/screenshots/management-depth-final-desktop.png`、`management-picker-dark-en-375.png`，另有 `management-recipient-picker.png`、`management-recipient-mobile375.png`、`management-shop-selection-mobile.png`。


## 兑换票券显示已领取权益（2026-10-20，最新）

- 撕下票根并经服务端确认兑换后，原票面改为“卡券兑换回执”，显示实际发放的卡券名称、类型、Token 额度／目标倍率／模型免费权益、精确模型 ID、分组 ID、激活后有效时长、激活截止时间及卡包入口。兑换仍只领卡，手动激活才开始计时；日卡、倍率和特殊费用规则均有提示。旧余额／并发／订阅兑换继续沿用原成功展示。
- `/redeem` 响应新增可选 `card`，在发卡原事务内按卡 ID 和用户 ID 读取已发卡快照。模板后续变更不会改变本次回执；读取失败时回滚，不引入新的计费修改。前端保留独立回执快照，重置后清除；旧服务未返回 card 时显示已到账并提供卡包入口。文本采用 Vue 转义。
- 实际浏览器兑换三类卡、Token 卡 Enter 和倍率卡拖拽、重复使用旧码拒绝且不残留旧回执、回执跳转卡包均通过；已领取卡仍为待激活，演示账户余额保持 100、并发保持 50。桌面浅／深色及 375×812 视口检查通过。手机最近活动旧调整记录曾撑宽到 394px，已改为窄屏纵向布局；最终 document/body scrollWidth=367 ≤ 375。深色标签选择器已修正，实际文字颜色 rgb(110,231,183)。手机验收为浏览器视口模拟，未宣称真实手机触摸／软键盘验收。
- Node 24.22.0／pnpm 9.15.9 最后一次全量前端 **408 文件、3,108 项通过、0 失败、0 跳过**（含手机记录布局修正）。随后仅修正深色标签 CSS，聚焦票券／兑换／卡包／国际化 **4 文件、36 项通过、0 失败、0 跳过**；最终全量 lint、vue-tsc、国际化 3 项和生产构建通过。构建保留既有 Browserslist／大包体积提示。日志：`receipt-frontend-accepted.log`、`receipt-focused-accepted.log`、`receipt-lint-accepted.log`、`receipt-build-accepted.log`。
- Go `go test ./internal/handler/dto ./internal/rewards` 通过；后端构建通过。PostgreSQL `-race -tags=integration` 回执快照／回滚与管理兑换状态专项 **2 项通过、0 失败、0 跳过**，覆盖原始快照、所有者过滤、事务回滚、重复源发卡及余额不变。日志：`receipt-go-tests.log`、`receipt-backend-build.log`、`receipt-integration.json`。首次 Docker socket 自动发现失败，显式使用 OrbStack socket 后重跑通过。
- 新可复跑脚本 `python3 tools/rewards-local/receipt_check.py` 经本机环境身份检查后，真实 HTTP 三类卡回执 **3 组通过**：模板编辑后回执仍匹配原始发卡快照、与卡包一致、余额／并发不变、不自动激活、重复码拒绝及只发一张。脚本修复重复运行的用户名冲突后再次通过；不修改原计费测试状态文件。证据：`receipt-http-accepted.log`、`receipt-http-results.json`。
- 本轮再次运行真实网关 HTTP 扣费回归 **8 组通过**：无卡、零余额免费日卡、倍率卡、Token 全抵／部分／耗尽、流式、双 Key 12 路共享额度。独立 Decimal 预期核对：`0.00238000 = 0.00158667 + 0.00079333`；12 路 `0.02856000 = 0.00476000 + 0.02380000`；上游每请求成本记录仍为 `0.0068`。日志 `receipt-billing-http.log`、`http-results.json`。
- 日志和截图位于 `/private/tmp/rewards-local/`。最终截图：`screenshots/redeem-receipt-desktop.jpg`、`redeem-receipt-mobile375.jpg`、`redeem-receipt-dark-desktop.jpg`、`redeem-receipt-dark375-final.jpg`。已恢复浅色、默认视口，保留本机实际后端的兑换回执页与运行服务。
- 本轮回执专项通过；仅使用本地协议模拟上游，不代表真实供应商生产验收。前述全量后端 unit 内存阈值失败仍未解决，未重跑全量后端默认／unit／integration，整个活动卡券／农场计划不据此标记全面完成。未合并、推送或生产部署。
