# Changelog

All notable changes to the **涨了多少** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- **后端登录与鉴权安全加固**
  - `AuthService` 新增进程内失败限流：默认 15 分钟窗口内密码登录失败 5 次、注册失败 8 次、Google 登录失败 10 次后返回 `AUTH_RATE_LIMITED` / HTTP 429。
  - 注册密码策略升级为至少 10 位、包含字母和数字、不能包含空白字符，并拒绝常见弱密码；注册页同步前端提示与预校验。
  - 认证入口 JSON body 限制为 16KB，降低大请求体滥用风险；新增 handler/service 回归测试覆盖限流、弱密码、Google 邮箱 claim 校验和超大 payload。
  - 配置新增 `auth_attempt_window_minutes`、`max_password_failures`、`max_register_failures`、`max_google_login_failures` 及对应环境变量覆盖。

- **crawler 全量基金目录同步与可用状态治理**
  - `cmd/crawler` 新增 `--catalog-only`，可配合 `--list all --save-db` 只同步 Eastmoney 基金目录的 code / name / type / catalog status，不逐只抓取详情和持仓。
  - `funds` 新增 `catalog_status / catalog_synced_at`；目录同步会将正常基金标记为 `active`，将名称含“(后端)”且详情接口不可用的份额标记为 `unavailable`，全量无 `--limit` 同步时将本地历史残留标记为 `catalog_missing`。
  - 默认基金搜索现在只返回 `active`，但直接按 ID 查询仍保留历史记录，避免影响已有持仓、自选或估值档案。
  - 2026-06-01 已重新执行全量目录同步并清理本地冗余：上游返回 26,927 条，先标记 `catalog_missing=111`，确认无用户自选/持仓/交易流水/估值档案引用后已备份到 `.omx/backups/manual/catalog-missing-funds-cleanup-20260601-094512.json` 并删除；当前本地 `funds` 表 26,927 条，其中 `active=26,737`、`unavailable=190`、`catalog_missing=0`，`stock_holdings` 为 4,533 条。

- **基金人工分类标签覆盖层**
  - `fund_classification_overrides` 扩展 `primary_theme_code` 与 `manual_tags_json`，并新增独立 migration，支持管理员补齐系统自动分类不准确的基金。
  - 新增管理员接口：读取分类字典、读取单只基金人工覆盖、保存主分类 / 主板块 / 主主题 / 人工标签 / 备注。
  - Dashboard 新增 `classification_override`，前端持仓分类卡可展示“人工校正”标签，并保留自动持仓权重拆解，避免人工标签被误读为真实持仓占比。
  - 首页与量化详情页的持仓分类卡新增管理员编辑入口；保存后刷新 dashboard 与 analysis，并自动失效该基金旧量化快照。

- **QDII 海外固定行情源配置化与选型文档**
  - 新增 `quote.overseas_source` / `QUOTE_OVERSEAS_SOURCE`，QDII / 海外持仓估值可在 `tencent` 与 `sina` 两个既有 provider 间切换。
  - 默认海外固定源保持 `tencent`，避免影响现有 QDII 估值链路；用户级国内行情源偏好仍不影响海外估值。
  - 新增 `docs/overseas-data-source-selection.md`，记录 Polygon / Alpaca / Twelve Data / Intrinio 的中期选型边界，并明确本轮不接 VIP / 付费官方 API。

- **持仓页待补齐过滤与高频排序**
  - 持仓页信息架构改为工作台导航：`总览 / 记录 / 持仓 / 风险 / 流水 / 工具` 六个功能块，顶部快捷入口直达记账、持仓列表、风险检查和流水记录；默认不再一次性展开全部长模块。
  - 各功能块说明文案压缩为短标题与短说明，记录、导入、VIP、流水筛选等操作放回对应功能区，降低首屏文字密度。
  - 持仓总览补齐“总价值 / 总收益”资产指标：官方口径按已就绪本金计算，盘中口径按已确认份额计算，并展示对应收益率，避免累计收益缺口。
  - 持仓页新增“只看待补齐”过滤器，支持在按基金聚合和分笔明细两种视图下快速定位未确认份额、官方净值未同步或真实口径未就绪的记录。
  - 过滤开启后会显示当前筛选范围和命中数量；无命中时展示空态并可一键恢复全部持仓。
  - 新增排序与筛选面板：支持仓位 / 金额、盈亏、涨跌幅、分笔数、最近录入、量化信号排序，以及盈利/亏损、就绪状态、单笔/多笔筛选；盈亏/涨跌幅跟随“官方口径 / 盘中预估”切换，缺失排序值统一后置。

- **量化分析 P2a 可信度增强第一轮**
  - `FundAnalysis` 新增 `confidence_factors`、`primary_evidence`、`counter_evidence` 与 `confidence_deductions`，用于展示可信度拆解、主证据、反方证据和扣分原因。
  - 可信度现拆分为持仓覆盖、持仓新鲜度、行业/主题映射、事件来源强度、实时行情可用性和历史对比完整度。
  - `cmd/audit-fund-analysis` 现在会输出核心样本的主证据、反方证据、数据缺口、可信度扣分和直觉复核结论。

- **量化分析 P2b AI 解释层边界第一轮**
  - 新增 `AIExplanationService` / `AIExplanationProvider` 边界；provider 输入限定为规则型分析、证据包、持仓、行业/主题快照与可引用来源。
  - `FundAnalysis` 新增 `ai_explanation`，用于返回解释层状态、规则结论、边界声明、归因段落、风险段落、引用与降级限制。
  - 未配置真实 AI provider、provider 超时或失败时，会返回非阻塞降级摘要；无证据包时解释层返回“证据不足 / 无法确认”。
  - `cmd/audit-fund-analysis` 与前端量化卡片已展示 AI 解释层状态、边界、引用和限制。

- **量化分析 P2c 缓存收口第一轮**
  - `ai_explanation` 新增缓存元信息：`cache_key`、`cache_status`、`expires_at`、`invalidation_basis`。
  - `fund_analysis_snapshots` 读取现会校验当前分析版本、同一上海自然日、解释层缓存有效期和 cache key，避免旧 JSON 快照绕过新规则。
  - `/fund/:id/analysis` 改为 fresh snapshot-first；`/analysis/batch` 与 `/analysis/rankings` 会过滤过期快照。
  - 前端 dashboard 请求增加 `include_analysis=false`，避免页面已有独立 analysis 请求时重复构建量化分析。

### Changed
- **Classic 主题暖色重配色**
  - `classic` 主题从冷灰白切换为暖白纸面底色，卡片、弹层、输入框改为暖色边框与柔和投影，降低浅色模式刺眼感。
  - 浅色主题下的 cyan / amber / rose / emerald 状态色统一降饱和，保留 A 股红涨绿跌语义，但减少高对比颜色跳动。
  - 主题切换器同步更新 Classic 描述与预览色块，新增全局偏好同步组件，确保量化详情等独立页面也继承用户选择的主题。

- **量化观察摘要与详情页实时事件重构**
  - 首页量化观察摘要卡新增“事件雷达”分区，优先展示实时宏观、持仓事件和主线暴露信号，减少主证据 / 风险限制 / 事件线索混在同一长文本里的问题。
  - `/analysis/[fundId]` 详情页新增实时事件雷达首屏模块，并将事件信号链改为按“实时宏观 / 持仓事件 / 主线暴露 / 口径与限制”分组展示。
  - `/analysis/[fundId]` 不再把缺失的预估净值、昨日净值、涨跌幅或持仓数值兜底成 `0` / `¥0.0000`，统一展示为 `--` 或加载态，避免金融数据缺失被误读为真实归零。
  - `/analysis/[fundId]` 顶部基金名会合并 dashboard / analysis / holdings 返回的基础信息；估值卡移除失效的“选择基金”占位。
  - `/analysis/[fundId]` 区分“正在生成/同步”和“确实为空”的系统态，占位文案改为短系统码；六维模块无分数时保留暗色雷达底座和扫描态。
  - 数据口径面板会跟随持仓层加载/空态显示“拉取中 / 待确认”，避免“股票持仓层”和持仓明细空态互相矛盾。
  - 宏观事件源新增美伊协议 / 霍尔木兹重开预期相关实时事件种子，按油气资源、消费成本、防御交易三类暴露方向映射；低暴露基金不会被无差别写入热点。
  - 事件模型与证据包新增来源名称、来源链接、发布时间、来源置信度和暴露映射依据；首页摘要卡与 `/analysis/[fundId]` 的实时雷达、事件分组、主/反证据都会展示可追溯来源。
  - 前端抽出 `AnalysisEventTraceMeta` 共享溯源组件，摘要卡和量化详情页复用同一套来源 / 映射依据展示，避免事件溯源 UI 分叉。
  - `analysis_version` 提升到 `baseline_v4`，避免旧快照继续隐藏新增溯源字段。
  - 新增后端回归测试覆盖实时事件命中能源暴露、消费成本缓和映射、低暴露不触发，以及事件转证据时保留来源溯源字段。

- **前端高帧率与移动端显示优化**
  - 移动端禁用高成本 `backdrop-filter`、fixed background 与 cyber 主题无限背景动画，降低小屏滚动时的重绘压力；实时状态点、慢速旋转、VIP 闪烁类无限动画在小屏降级为静态展示。
  - `ScrollReveal` 从 `transition-all` 收口为只过渡 `opacity` / `transform`，并缩短进入时长，避免无关属性参与合成。
  - 首页估值卡移动端缩小主涨跌数字与趋势图标、减少内层 glass 嵌套、净值信息改为小屏单列，降低拥挤与模糊层叠成本。
  - 分时走势图与持仓明细在移动端改为更稳的纵向表头和紧凑 padding；持仓表增加最小宽度、touch 横向滚动与小屏滑动提示，避免列被压缩。

- **前端设计系统第一批 targeted evolution**
  - 新增 `Surface`、`StatusBanner`、`EmptyState`、`SectionHeader` 轻量 UI primitives，用于收口重复的 glass 面板、状态提示和空态表达，不引入新 UI 库或动画依赖。
  - 首页市场状态 / 涨幅贡献卡片、预热 / 集合竞价 / 切换失败提示改为复用统一 primitives；持仓页登录空态、首笔持仓空态、筛选提示和无匹配空态同步收口。
  - 静态信息页 Hero 与卡片网格改为复用 `Surface`，并移除一处高频 tracking eyebrow 样式；主要页面容器从 `min-h-screen` 调整为 `min-h-[100dvh]`，提升移动端 viewport 稳定性。
  - 第二批补充 `ActionButton` 轻量 primitive，收口首页顶部导航、静态信息页返回链接、持仓页登录 / 注册入口与筛选空态恢复按钮的 CTA 类名重复，继续保持路由、业务 hook 和表单字段不变。
  - 第三批抽出 `HoldingsWorkspaceNav`，将持仓页顶部概览、快捷动作、快速开始和工作台 tab 网格下沉为纯展示组件；`holdings/page.tsx` 继续保留 active tab、seed demo、口径/视图状态和业务数据流。
  - 第四批收口 `/watchlist` 低风险展示壳层：加载占位、未登录空态、反馈提示、分组总览/管理面板和无分组/无匹配空态改为复用 `Surface`、`EmptyState`、`StatusBanner`、`ActionButton`；分组创建、选择、搜索、拖拽、删除、编辑逻辑保持不变。
  - 第五批收口公告与反馈页展示壳层：`/announcements`、`/announcements/[id]`、`/issues`、`/issues/[id]` 的加载/错误/空态、管理员提示、提交入口和内容面板继续复用 `Surface`、`EmptyState`、`StatusBanner`、`ActionButton`；公告发布、CHANGELOG 导入、反馈提交、状态更新和官方回复流程保持不变。
  - 同步收口全局未读公告弹窗 `GlobalAnnouncementDialog` 的弹窗面板、摘要壳层和查看详情 / 标记已读 CTA，使站内公告入口继续沿用同一套 `Surface` / `ActionButton` 展示语言；未读公告拉取、关闭和标记已读逻辑保持不变。

- **前端 redesign 第二轮收口与 Vercel API 域名修复**
  - 静态信息页抽出 `StaticInfoShell` / `StaticInfoHero` / `StaticInfoCardGrid`，让品牌化 404、隐私政策和服务条款复用统一壳层，并为隐私政策 / 服务条款补充页面 metadata。
  - 社区反馈入口从“我有想法！”统一收口为“反馈与想法”，去掉感叹号和英文 eyebrow；反馈列表与详情页用户可见文案不再暴露 `Issue` / `IDEA`。
  - 前端新增 `api-base-url` helper：浏览器默认继续请求同源 `/api/v1/*`，若误把 `NEXT_PUBLIC_API_URL` 配成前端域名 `https://fund.wrenzeal.top` 会回退同源代理，避免绕回前端域名或破坏 Cookie 登录。
  - Vercel Route Handler 的后端代理在 Vercel 环境下默认 fallback 到 `https://api.fund.wrenzeal.top`；正式生产仍应显式设置 `BACKEND_URL=https://api.fund.wrenzeal.top`。
  - `docs/vercel-frontend-deploy.md` 与 `web/README.md` 已补充 Vercel 环境变量、API 子域 DNS/Nginx/TLS 要求，以及不要把 `BACKEND_URL` 填成前端域名的说明。

- **自选页浮层复杂度收口**
  - 新增 `FloatingListbox` 复用组件，统一 fixed portal 下拉的定位、滚动/resize 监听、遮罩关闭、listbox 容器与最大高度计算。
  - `/watchlist` 的目标分组下拉与基金搜索结果下拉改为复用该组件，删除页面内重复的 rect state、resize/scroll effect 与 `createPortal` 样板，同时保留原有选择、搜索、禁用和可访问性语义。

- **前端 redesign 首轮体验收口**
  - 全局字体从 Inter-only 收口为 Geist + 中文字体栈，并为金融数字启用 tabular nums、标题/正文 `text-wrap` 与统一 focus-visible，可读性和键盘可访问性更稳定。
  - 默认深色背景从蓝紫 AI 渐变弱化为金融深色径向光感 + 细网格纹理，保留业务涨跌色和更克制的主操作色。
  - 新增全局 skip link，恢复移动端页面缩放能力，并补充品牌化 404、隐私政策、服务条款和统一页脚，减少用户流程死胡同。
  - `LoadingIndicator` 新增布局骨架屏，基金数据加载从纯旋转圈升级为结构化 loading 面板。
  - VIP 介绍、开通页与样例报告文案从“低吸 / 看结论 / 操作方向”降级为“证据链 / 风险观察 / 复核条件”，继续强调不构成投资建议或交易指令。
- **认证错误与生产 Cookie 安全收口**
  - 登录邮箱格式错误统一按 `INVALID_CREDENTIALS` 处理，不再区分“格式错误 / 账号不存在 / 密码错误”。
  - 认证默认异常响应改为通用 `Authentication failed`，避免向客户端暴露底层错误原文。
  - Gin 只信任本机反代代理头，避免客户端伪造代理 IP 绕过认证限流。
  - 生产 `/etc/fund-live/fundlive.yaml` 已备份并显式写入 `auth.cookie_secure=true` 及认证限流阈值，HTTPS 会话 Cookie 继续保持 HttpOnly + SameSite=Lax。

- **前后端代码优化与重复逻辑收口**
  - 前端新增 `BrandMark` 共享组件，统一首页、账户中心、社区/公告、用户模块顶部品牌块，避免品牌图标与文案结构继续多处复制。
  - 后端 `fund_search.go` 收口基金搜索文本归一化，并在 scored match 中预计算排序用 ID / 名称，减少排序阶段重复 trim，同时保持既有搜索优先级。

- **浏览器标签页与首页品牌副标题更新**
  - root metadata title / Open Graph title 改为 `FundLive - 你的基金估值系统`。
  - 新增并优化 `BrowserTitleTicker`，当浏览器标签标题较长且用户未开启 reduced motion 时，以更高频字符步进循环滚动 title，并在每轮完整标题后短暂停顿，降低低帧率跳动感。
  - 首页左上角主标题保持“涨了多少”，副标题改为 `FundLive - 实时基金估值`。
  - `/favicon.svg` 继续使用蓝色线性心电/活跃度图标，透明背景下只保留中间线条；`app/favicon.ico` 作为兼容 fallback。

- **自选页编辑分组按钮修复**
  - `/watchlist` 分组卡片不再整卡设置 `draggable`，分组排序改为只能从“拖拽排序”手柄发起，避免父级拖拽抢占内部“编辑分组”按钮点击事件。
  - “编辑分组”按钮会阻止拖拽事件冒泡，并增加点击高亮/弹跳反馈；编辑弹窗新增背景淡入与面板进入动画，并继续尊重 `prefers-reduced-motion`。

- **后端 Go 版本基线升级到 1.26.3**
  - `go.mod` 已从 `go 1.25.8` 升级到 `go 1.26.3`，CI 的 `actions/setup-go` 会继续通过 `go-version-file: go.mod` 跟随新版本。
  - 已按 Go 1.26 release notes 扫描后端受影响点：项目未使用 `net/http/httputil.ReverseProxy.Director`、`image/jpeg` 位级断言、`go/ast.BasicLit.ValueEnd`、自定义 `crypto/rand.Reader` 或相关 `GODEBUG` 兼容开关；现有 `crypto/rand` 仅用于 token/nonce，`httptest` 仅用于常规 handler 测试，因此无需配套代码改造。
  - 本机工具链 `go1.26.3 linux/amd64` 已用于运行后端测试与构建验证。

- **前端依赖漏洞修复**
  - `next` 与 `eslint-config-next` 升级到 `16.2.7`，修复 Next 16.1.x 暴露的 request smuggling、Server Components DoS、middleware/proxy bypass、SSRF、XSS 与 cache poisoning 等 npm audit 告警。
  - 由于当前 Next 16.2.7 包声明仍固定 `postcss@8.4.31`，新增 `overrides.postcss=8.5.15`，让 Next 实际解析到安全 PostCSS 补丁版本，避免 npm 建议降级到不兼容的 Next 9。
  - 前端依赖树已验证 `npm audit` 为 0 漏洞，并通过 lint 与 production build。

- **生产访问 HTTPS 与部署端口收口**
  - 已在运行环境为 `fund.wrenzeal.top` 启用 Let's Encrypt HTTPS，HTTP 80 自动 301 跳转到 HTTPS 443。
  - Nginx 保持页面/静态资源走 Next.js `127.0.0.1:13069`，`/api/` 与 `/health` 直连 Go 后端 `127.0.0.1:13896`，避免健康检查落到前端代理后再回退 8080。
  - `scripts/deploy-backend.sh` 与 self-hosted workflow 的默认后端健康检查改为 `http://127.0.0.1:13896/health`。
  - `scripts/deploy-backend.sh` 在 `/etc/fund-live/fundlive.yaml` 存在时会写入 systemd drop-in，确保后端以 `FUNDLIVE_CONFIG=/etc/fund-live/fundlive.yaml` 读取生产端口配置。
  - `scripts/deploy-frontend.sh` 默认注入 `BACKEND_URL=http://127.0.0.1:13896` 到 PM2 运行时，避免后续前端部署后同源 API 代理回退到 `127.0.0.1:8080`。

- **自选页分组管理下拉交互修复**
  - `/watchlist` 分组管理里的“目标分组”触发器与下拉面板改为不透明主题背景，并将菜单改为 portal 固定弹层，避免选项视觉重叠时点击穿透到下方按钮。
  - 分组选择下方的基金搜索结果改为 portal/fixed 浮层下拉，不再在卡片内联展开撑高整个分组管理区域，并复用不透明主题背景与轻量进入动画。

- **前端主要页面滚动进入动效统一**
  - 新增 `web/src/components/scroll-reveal.tsx`，将滚动进入监听、渐显、轻微上移/缩放和 stagger 列表动画抽成共享组件，继续使用原生 IntersectionObserver/CSS transition，不新增依赖。
  - 首页、持仓、自选、想法、公告、登录/注册、VIP 介绍/开通/任务/报告、公告详情、想法详情与持仓流水详情已接入区块级或卡片级滚动渐显。
  - `analysis-layout.tsx` 现在复用共享滚动 reveal 能力，保留量化详情页/排行榜的 section heading 兼容导出。

- **量化详情页局部布局修正**
  - 事件信号链与风险拆解改为上下排布，不再在宽屏并排展示。
  - 涨幅情况、持仓分类、数据口径改为上下排布，降低三块并排造成的拥挤和错位。
  - 风险拆解中的 CURRENT RISK、风险/事件模块和重点风险提示改为上下层级，并修复重点风险提示列表样式。

- **量化看板与排行榜前端布局重构**
  - `/analysis/[fundId]` 改为“结论与证据优先”的分段布局：首屏结论、洞察条、建议分布、六维模块、主/反证据、事件风险、结构季报、数据口径与方法限制顺序展示。
  - `/analysis/rankings` 改为排行榜首屏概览、三类观察池统计、当前首位样本放大展示、分榜列表与压缩口径说明；上方积极池 / 观察池 / 风险池统计卡保留并列展示，下方“结构偏积极 / 最值得观察 / 高风险关注”分榜改为上下排布，避免宽屏并列造成阅读割裂。
  - 删除旧版重复概览卡和重复方法说明，把结构、数据口径、方法限制分别收敛到压缩模块，降低长页面重复阅读成本。
  - 新增共享 `analysis-layout.tsx`，统一滚动进入渐显、section heading 与建议分布滚动绘制动效，使用现有 CSS/IntersectionObserver，不新增前端依赖。

- **量化规则结论阈值与审计收口**
  - `analysis_version` 提升到 `baseline_v3`，使旧版量化快照自动失效。
  - 规则主结论与前端标签统一使用阈值口径：`increase >= 55` 才显示“结构偏积极”，`decrease >= 60` 才显示“风险偏高”，否则统一为“适合观察”。
  - 观察型结论下的反方证据不再把正向事件混入限制列表；审计命令会对低可信度因子给出更明确的人工复核提示。

- **量化前端信息架构重构**
  - 基金详情页量化模块改为摘要卡，只展示核心结论、总分、建议分布、少量主因/风险和完整看板入口。
  - `/analysis/[fundId]` 量化详情页重排为结论总览、建议分布、六维模块可视化、主/反证据、事件信号链、风险/结构辅助信息。
  - 新增环形总分、堆叠分布条、模块雷达/进度条、时间线节点等视觉元素，减少纯文字规则堆叠。

- **量化事件权重口径收口**
  - 当前持仓股票、持仓权重、行业/主题与近期有来源事件优先于基金自身普通公告。
  - 基金自身事件默认作为辅助低权重信号，仅在基金经理变更、清盘/限购、费率、规模异常等直接影响产品的事件上提升权重。
  - 行业/主题热点必须先匹配到基金当前行业/主题暴露，并达到最低暴露权重后才进入宏观/政策事件层。
  - 前端量化标签和结论文案弱化投资建议口吻，从“偏加仓 / 偏减仓 / 偏持有”调整为“结构偏积极 / 风险偏高 / 适合观察”。

## [2026.4.27] - 2026-04-27

### Added
- **基金量化分析基础版量化看板落地**
  - `dashboard` 主链路现已直接返回 `analysis`
  - 基金主页新增量化分析摘要卡，并新增独立详情页 `/analysis/[fundId]`
  - 第一版已覆盖：综合评分、加 / 平 / 减分布、六维模块评分、事件分析、主要理由、风险提示

- **量化看板事件层支持当前持仓股票公告事件**
  - 新增 `holding_news_source`，通过 CNINFO 官方公告查询页按需抓取前几大重仓股近 45 天公告
  - 第一版纳入：业绩预告、定期报告、经营类公告、公司治理类公告
  - 相关事件会进入 `analysis.event_impacts`，并同步影响 `summary / reasons / warnings`

- **量化看板事件层支持基金自身公告事件**
  - 新增 `fund_notice_source`，通过 Eastmoney `api.fund.eastmoney.com/f10/JJGG` 按需抓取基金公告
  - 第一版纳入：定期报告、基金销售、人事调整与其他公告
  - 基金自身事件以 `target_scope=fund` 进入 `analysis.event_impacts`，并同步影响 `summary / reasons / warnings`

- **量化看板事件层支持板块 / 主题时事轻量聚合**
  - 基于当前重仓股近期公告事件，聚合生成主线级事件，例如 `current_exposure_event_cluster`
  - 第一版可输出类似“半导体芯片主线近期事件密集”的主题/行业事件提示
  - 当前实现仍基于重仓股公告聚合，不是独立主题资讯源

- **量化看板事件层支持目标 ETF 公告事件**
  - 对联接基金 / 目标 ETF 口径的量化分析，当前会同步拉取目标 ETF 自身公告
  - 这样事件层可同时覆盖：联接基金本体、目标 ETF、当前重仓股与主线聚合事件

- **量化看板补全行业 / 主题权重变化与风格漂移提示**
  - 新增行业 / 主题权重提升与回落事件
  - 当行业 / 主题权重出现双向显著变化时，会提示组合存在风格漂移迹象
  - 第一版仍基于当前季 / 上一季 snapshot 对比，不依赖新的外部时事源

- **量化看板上一季持仓支持本地持久化快照**
  - 新增 `stock_holding_history` 表与对应 migration
  - 读取上一季持仓时，现会优先使用本地历史快照；缺失时再按需抓取 Eastmoney 年度持仓页并落库
  - 对联接基金 / 目标 ETF 口径，上一季持仓也会优先按目标 ETF 代码对齐读取

- **量化看板支持第一版宏观 / 政策事件**
  - 新增轻量宏观/政策事件源，并按行业 / 主题编码映射到量化事件层
  - 当前宏观事件会以 `target_scope=macro` 进入 `analysis.event_impacts`
  - 第一版仍是种子源方案，主要用于提供可验证的宏观/政策解释能力

- **量化看板支持第一版纯指数层事件**
  - 新增指数调样 / 样本维护窗口提示
  - 当前纯指数层事件会以 `target_scope=index` 进入 `analysis.event_impacts`
  - 第一版主要覆盖指数调样窗口临近提示，后续再扩展到更细的指数规则变化与调样结果

- **量化看板完成第一轮权重 / 阈值校准**
  - 针对强趋势 ETF / ETF联接样本，降低风险 / 性价比模块的过度压制
  - “偏加仓”结论阈值从 `increase>=60` 调整为 `increase>=55`
  - 运行态样本复核中，`159813` 总分由 `62.2` 提升到 `68.1`，`012970` 由 `64.6` 提升到 `70.4`

- **六个核心样本完成第一轮逐只验收与第二轮校准收口**
  - 六个核心样本：`159813 / 012970 / 023408 / 005827 / 000362 / 000370`
  - 当前代码版后端审计结果已全部通过首轮验收，验收记录保存于 `.omx/backups/manual/fund-analysis-sample-review-2026-04-27.txt`
  - 第二轮校准中移除了“中性主线事件对事件分的隐性抬升”，使规则更稳、更不容易被中性事件轻微抬高

- **持仓页接入第一版量化分析标签**
  - 持仓页按基金聚合卡片与分笔明细行现可显示“偏加仓 / 偏持有 / 偏减仓”标签
  - 同时补充风险标签与可选总分展示，帮助用户在持仓页直接做轻量判断
  - 当前实现基于 holdings page 统一拉取各基金 `dashboard.analysis` 结果

- **量化看板事件层支持季度结构变化辅助事件**
  - 新增上一季前十大持仓按需抓取能力，基于 Eastmoney `FundArchivesDatas ... year=<YYYY>` 获取多期季度持仓
  - 第一版已补：前十大换手、单只重仓股季度内权重提升/回落、前三大集中度变化
  - 主行业 / 主主题切换或权重显著变化也会作为辅助事件写入看板

### Changed
- **量化看板信息层级重排**
  - 看板已调整为：六维模块评分 → 事件分析 → 综合结论 / 总分 → 详情按钮
  - 总分区已放大并强化视觉权重，基础结论贴近总分展示
  - CTA 在小屏下改为自适应布局，避免文字与箭头挤压

- **量化看板事件模型结构化增强**
  - `event_impacts` 新增：
    - `target_scope`
    - `strength`
    - `horizon`
    - `related_symbols`
    - `weight_hint`
  - 事件层当前产品口径已收口为“**当前持仓 + 持仓股票当前表现/相关时事**”优先；上一季持仓变化仅作为辅助观察

## [2026.4.25] - 2026-04-25

### Changed
- **基金持仓分类模块准确性增强**
  - 扩充行业板块词典，补齐通信设备、银行、券商与非银金融、保险、食品饮料、机械设备、地产REITs、资源周期、公用事业、农业等常见行业
  - 扩充主题词典，补齐半导体芯片、平台互联网、医疗健康、消费升级、金融、高股息红利、资源周期、地产REITs 等泛化主题
  - 增强行业与主题关键词 fallback，并为主题分类增加“按行业回退到主题”的后备逻辑

- **分类主类与置信度逻辑调整**
  - 若存在已识别的非 `other_*` 类别，主板块 / 主主题优先从已识别类别中选，不再轻易被“未归类”抢主类
  - 行业 / 主题置信度改为按“已识别覆盖率”判断，降低“未归类占比很高但仍显示 high”的误导情况
  - `other_equity / other_theme` 的展示文案已调整为更诚实的“未归类权益 / 未归类主题”

- **基金主页分类展示增强**
  - 基金主页分类模块会对低置信度结果显示“覆盖有限”提示
  - 行业与主题模块在运行时按当前 holdings 重建快照，减少旧快照长期滞后的问题

## [2026.4.24] - 2026-04-24

### Fixed
- **ETF 联接基金主展示涨跌幅改为目标 ETF quote 优先**
  - 对于已解析出目标 ETF 的联接基金，主展示的实时涨跌幅不再优先使用目标 ETF 持仓穿透估值
  - 系统现优先使用目标 ETF 的实时行情涨跌幅，只有目标 ETF quote 不可用时才回退到持仓估值链路
  - 当当前默认行情源拿不到目标 ETF quote 时，系统现会自动尝试其它已注册国内行情源，再不行才退回持仓估值

- **联接基金“追踪目标”权重显示异常**
  - 修复“追踪目标”模块中主追踪目标 `weight_percent` 显示为 `0.00%` 的问题
  - 对单一主追踪目标，现默认展示 `100.00%`

### Changed
- **基金主页新增目标 ETF 持仓模块**
  - 当联接基金已解析出目标 ETF 且可获取该 ETF 的持仓时，会在“追踪目标”模块下方新增“目标 ETF 持仓”模块
  - 该模块用于补充展示目标 ETF 的底层持仓明细，帮助用户理解联接基金跟踪的具体暴露

- **自选页分组管理补全**
  - 支持通过弹窗编辑分组名称、分组说明与分组颜色
  - 新增分组排序持久化与拖拽排序能力；在“全部分组”且未搜索时可直接拖拽排序
  - 刷新页面后分组顺序保持一致

- **联接基金残留直持仓治理工具上线**
  - 新增 `cmd/feeder-holdings-cleanup`，用于扫描已解析目标 ETF 的联接基金 residual holdings
  - 工具支持生成审计清单、推荐白名单、自动备份 CSV，以及默认 dry-run 的 cleanup 模式
  - 已完成一轮实际扫描与备份产物生成，后续可基于白名单分批清理数据库残留数据

## [2026.4.23] - 2026-04-23

### Fixed
- **自选页分组导航跳转会被 sticky 头部遮挡**
  - 修复 `/watchlist` 页面点击“分组导航”后，目标分组顶部会被 sticky 的“你的自选”头部区域挡住的问题
  - `AccountAreaShell` 现会提供动态锚点偏移量：桌面端按完整 sticky 头部高度计算，移动端按折叠后的顶部栏高度计算
  - 自选页分组 section 已切到动态 `scroll-margin-top`，跳转后无需再手动补滚

- **自选页分组定位与折叠交互增强**
  - “分组导航”胶囊按钮已补充 hover / click 动效，当前高亮分组的反馈更明显
  - 每个分组卡片头部的“展开 / 收起分组”按钮新增流光反馈、轻微位移动效和箭头旋转动画
  - 自选页新增“当前浏览分组”提示卡片，并在浏览模式说明文案中直接显示当前分组名称，减少“当前分组”语义不明确的问题

- **自选页分组管理补全：支持编辑、颜色和排序**
  - `PUT /api/v1/user/watchlist/groups/:groupId` 现已支持更新分组名称、说明和 accent 颜色字段
  - 新增 `PUT /api/v1/user/watchlist/groups/reorder`，用于持久化分组顺序
  - 自选页分组卡片已新增“编辑分组”入口，改为弹窗编辑名称、说明与颜色
  - 在“全部分组”且未搜索时，支持直接拖拽分组卡片调整顺序；刷新页面后顺序保持一致

## [2026.4.22] - 2026-04-22

### Changed
- **持仓页支持实时盈亏预估**
  - 白天在官方真实涨跌尚未就绪时，持仓页会根据基金预估涨跌幅显示实时盈亏预估
  - 夜间官方净值与真实涨跌同步后，会自动切换并覆盖预估盈亏展示
  - 单条持仓、按基金聚合视图和顶部总览卡片都已统一到这套预估 -> 官方覆盖逻辑

- **基金主页持仓分类拆分为板块 / 主题两个子模块**
  - 原本混合展示的“持仓分类”卡片现已拆成横向并列的：
    - `行业板块`
    - `主题分类`
  - 每个子模块内部继续使用列表形式展示主项与 Top 明细占比
  - 板块与主题语义分离，避免用户把行业暴露和主题暴露混为一层

- **自选页支持分组精准定位**
  - 自选页新增分组搜索，支持按分组名称与说明实时过滤
  - 顶部新增分组导航胶囊，可快速定位到目标分组
  - 分组卡片支持独立折叠 / 展开，减少分组较多时的页面长度
  - 自选页新增 `全部分组 / 当前分组` 浏览模式，方便在分组很多时聚焦单个分组查看

## [2026.4.21] - 2026-04-21

### Changed
- **集合竞价阶段首页展示收口**
  - 首页基金详情在集合竞价阶段不再整块隐藏持仓模块
  - 持仓明细、重仓股覆盖、持仓占比与 TOP 卡片会继续展示静态持仓信息
  - 现价、涨跌幅、贡献等动态行情字段改为占位，等待 09:30 开盘后恢复

- **联接 / 跟踪基金详情默认展示下一层目标**
  - `GET /api/v1/fund/:id/holdings` 新增 `display_level`、`display_items` 与 `lookthrough_available`
  - 对联接基金、ETF 联接 QDII 等基金，详情页默认展示下一层目标（如目标 ETF / 跟踪指数），不再直接展开到底层股票
  - 前端基金主页“持仓明细”与右侧统计 / TOP 卡片已适配 `target_layer`，估值链路仍可继续沿用穿透逻辑

- **实时估值覆盖范围扩容到全量可估值基金**
  - 新增 `fund_estimate_capabilities`，按 direct holdings / feeder target / valuation profile / QDII holdings 等链路记录基金当前估值能力
  - 后台新增估值覆盖调度器：启动时扫描能力清单，并将 `supported / degraded` 基金纳入后台更新池
  - 实时 collector 改为按批次和 goroutine 并发执行，默认采用：
    - 能力扫描 batch `2000`
    - 国内更新 batch `300`
    - 海外更新 batch `100`
    - 国内 worker `8`
    - 海外 worker `3`
  - collector 现已区分 request-driven 高频基金与 capability-driven 全量基金，支持 `1 / 3 / 5` 分钟分层更新

- **基金分类升级为行业 + 主题双层体系**
  - 在原有行业板块分类之外，新增主题分类：
    - `AI应用`
    - `算力`
    - `CPO/光模块`
    - `商业航天`
    - `卫星互联网`
    - `机器人`
    - `数据基础设施`
    - `军工电子`
  - 新增 `fund_themes`、`instrument_theme_map`、`fund_theme_snapshots`、`fund_theme_breakdown`
  - 基金主页 `dashboard` 现已返回 `theme_snapshot`，基金主页会同时展示“主板块”和“主主题”
  - 主题分类继续走持仓映射 + 权重聚合主链路，不引入 AI 到线上分类判断

## [2026.4.19] - 2026-04-19

### Changed
- **基金分类模块推进到第二阶段**
  - 新增稳定主分类层：`fund_categories`、`funds.category_code`、`fund_classification_overrides`
  - 基金分类现已同时支持主分类与板块快照两层语义，普通基金、联接基金、QDII 都可生成主板块与 Top 3 板块
  - 基金主页 `dashboard` 已返回 `sector_snapshot` 与主分类信息，基金主页可展示主分类、主板块与 Top 3 板块
  - `fund/search` 现支持按 `category` / `sector` 过滤，搜索结果也会回传主分类信息
  - 板块快照更新已接入主要持仓写入路径，包括基金预热和月度持仓刷新链路

- **持仓模块官方口径与盘中预估口径进一步拆清**
  - 总仓汇总现支持展示“已就绪部分”的官方市值 / 今日盈亏 / 今日涨跌幅，不再因为少数持仓待补齐就整块隐藏
  - 持仓页汇总会明确标注官方口径覆盖范围（已就绪条数与本金覆盖），避免把未就绪持仓误解为已纳入官方总仓
  - 单条持仓在真实净值未就绪时，会优先基于已确认份额显示盘中预估市值与盈亏；若份额尚未确认，则只保留按本金口径的提示性预估
  - 前端文案已区分“最新官方市值 / 今日官方盈亏”和“盘中预估市值 / 盘中预估盈亏”，减少真实口径与预估口径混淆

- **持仓页视图升级为聚合 / 明细 / 口径切换**
  - `GET /api/v1/user/holdings` 现已返回 `aggregates`，支持按基金聚合展示同一基金的多笔持仓
  - 持仓页新增两组切换：`按基金 / 分笔明细` 与 `官方口径 / 盘中预估`
  - 默认视图调整为 `按基金 + 官方口径`，聚合卡片支持展开查看分笔
  - 盘中预估总览与聚合行优先按确认份额汇总；份额未确认的记录只保留提示，不再混入预估总值

## [2026.4.18] - 2026-04-18

### Changed
- **移动端账户/社区页顶部区域收缩**
  - 自选、持仓、想法、公告等页面在移动端向下滚动时，会自动隐藏顶部标题、说明和导航区块，减少小屏幕被持续占用的高度
  - 页面滚动离开顶部后，右下角会显示“回到顶部”按钮，点击后可平滑返回顶部并重新显示顶部模块
  - 该行为统一落在共用壳组件中，桌面端布局保持不变

## [2026.4.17] - 2026-04-17

### Fixed
- **自选页删除分组 500 错误**
  - 修复 PostgreSQL 仓储在删除自选分组时使用 `DELETE ... JOIN` 导致的 500 Internal Server Error
  - 删除分组链路现改为先按用户拥有的分组子查询删除 `tb_user_watchlist_fund` 记录，再删除 `tb_user_watchlist_group`
  - 删除单只自选基金的仓储过滤方式也同步调整为子查询，避免同类 PostgreSQL 兼容性问题

- **联接基金被无效自有持仓阻断 fallback**
  - 当基金自身持仓记录存在、但 `holding_ratio` 全为 `0` 时，系统现在会将其判定为“无有效持仓”
  - 估值、持仓详情、分时回填与只读预热链路已统一切到“有效持仓”判断，不再因为脏数据阻断目标 ETF fallback
  - 像 `020465 招商中证半导体产业ETF发起式联接C` 这类基金，现在会正确回退到目标 ETF `561980` 的持仓与估值链路

- **QDII 海外基金详情缺失**
  - 持仓解析器现已支持从东财 `unify/r/105.NVDA` 这类链接或代码列文本中提取海外 ticker，如 `NVDA`、`AAPL`、`GOOG`
  - 新增海外交易所标识 `US`，并优先识别 `QDII` 基金类型，避免将 QDII 基金误判成普通股票基金
  - 零占比持仓现在会在解析阶段直接过滤，减少脏数据污染详情和估值链路
  - 已补上 QDII 海外实时行情估值链路，`017437` 这类基金现可返回真实持仓涨跌幅与贡献值；仅在海外 quote 全缺失时才回退到降级结果
  - 海外股票现已改为固定独立数据源 `overseas_fixed`，不再受用户 `sina / tencent` 行情源切换影响

- **首页卡片与走势图预估不一致**
  - 首页新增统一 `dashboard` 聚合接口，卡片 estimate 与走势图末点改为共用同一份后端快照
  - 首页 estimate 与走势图刷新频率统一为 30 秒，减少两条链路各自轮询导致的偏差
  - 收盘后 / 周末展示最近交易日曲线时，15:00 末点现已和卡片 estimate 保持一致

## [2026.4.16] - 2026-04-16

### Changed
- **持仓页切换到“本金 + 真实口径”双语义展示**
  - 用户持仓继续保留 `amount` 作为本金 / 录入金额，不再把真实净值结果回写覆盖本金语义
  - 后端为 `tb_user_fund_holding` 补充 `shares`、`confirmed_nav`、`confirmed_nav_date`，用于承接确认净值和份额口径
  - 持仓创建时会优先按 `as_of_date` 命中官方净值并计算份额；若暂时查不到确认净值，则以待补状态落库

- **持仓真实市值 / 今日盈亏 / 总仓汇总正式落地**
  - `GET /api/v1/user/holdings` 已升级为 `items + summary` 结构，单条持仓可返回真实当前市值、今日盈亏、今日涨跌幅
  - 顶部持仓总览新增总本金、总价值、总收益、总今日盈亏、总今日涨跌幅卡片
  - 总仓涨跌幅改为按昨日市值加权计算，不再做简单平均

- **真实口径与盘中预估完全分层**
  - 持仓页单条卡片已拆分“真实口径”和“盘中预估”展示，不再复用同一组金额字段
  - 当部分持仓缺少确认净值或最新官方净值时，单条会给出降级提示，总仓则不会混入不完整真实数据
  - 夜间官方净值同步完成后会补齐可命中的历史确认字段，减少旧持仓长期停留在待补状态

- **联接基金目标 ETF 解析升级为详情页优先**
  - 联接基金解析顺序已改为：成功缓存映射 -> 基金详情页 `查看相关ETF` -> 详情页 `跟踪标的` 增强搜索 -> 搜索 fallback
  - 搜索 fallback 现会过滤非标准 6 位基金代码结果，减少私募/杂项结果污染
  - 失败冷却窗口已从 12 小时缩短为 30 分钟，避免同一只基金在短期内长时间不可用
  - 已缓存成功映射但目标 ETF 暂无持仓时，会直接回到目标 ETF 行情估值降级链路，不再重新走脆弱搜索流程

## [2026.4.9] - 2026-04-09

### Changed
- **基金分时走势图与实时估值统一为单链路**
  - 顶部实时估值与分时走势图现已共享同一套基金估值快照内核，不再各自维护独立加权逻辑
  - 分时回放改为按时间点构造持仓 quote 快照，再调用统一估值内核计算基金涨跌幅与估值
  - 含港股重仓的基金现在会在分时走势图回放中纳入港股分钟数据，避免图表只反映 A 股权重

- **分时走势图边界行为修正**
  - 交易日 `15:00` 后分时图会继续展示**当天**曲线，不再错误回退到上一交易日
  - 分时图的最后一个 `15:00` 点会与顶部 estimate 保持一致
  - 午休衔接已统一：`13:00` 固定承接 `11:30` 的上午收盘点位，`13:05` 起再恢复下午真实更新

- **数据库空库首启迁移补全**
  - 受控 SQL migration 已补齐核心基金表与用户表的基线建表能力
  - 空 PostgreSQL 库首启不再依赖临时开启 `database.auto_migrate=true`
  - 当前迁移顺序已调整为先建核心 schema，再应用社区、公告、VIP 与索引类 migration

### Fixed
- **受影响分时数据回补**
  - 已对受影响的当天分时数据和一部分可安全重建的历史分时会话完成回补
  - 对无法可靠重建的旧会话保留原始记录，避免误删或写入错误历史图

- **非 VIP 页面主题收尾**
  - 首页切基金时的全屏加载态已改为直接使用主题背景变量，不再依赖未定义的 `--bg-primary`
  - 市场状态组件的未挂载占位点已切到主题变量，减少 `classic / dark / cyber` 之间的主题泄漏

## [2026.4.8] - 2026-04-08

### Added
- **公开想法详情页支持官方回复**
  - 管理员现在可以在 `/issues/:id` 详情页为每条公开想法写入一段公开可见的官方回复
  - 官方回复会记录回复人、回复时间，并对所有访问详情页的用户公开展示
  - 后端新增管理员接口 `PUT /api/v1/admin/issues/:id/reply`
  - 数据库 `issues` 表新增官方回复相关字段，并通过受控 SQL migration 管理

### Changed
- **公开想法处理链路增强**
  - 想法详情页从“只展示状态”升级为“状态 + 官方回复”的公开处理面板
  - 管理员在详情页中除了切换 `pending / accepted / completed` 状态外，还可直接编辑官方回复内容
- **基金首页展示信息更清晰**
  - 基金主卡片新增基金代码展示，放在基金经理信息上方
  - 原“前十大重仓股”标题调整为“重仓股明细”，并补充当前参与估值展示的数量说明，降低用户对“为什么不是 10 只”的误解
- **重仓股估值与持仓解析支持补强**
  - 修复新浪行情请求参数重复拼接导致部分个股（如 `002027`）无法返回实时行情的问题
  - 基金持仓解析现已支持港股 5 位代码，并能在估值链路中识别 `hk` 行情代码
  - 修复东财持仓明细在原始内容已是 UTF-8 时仍强制按 GBK 转码造成的中文名称乱码问题
  - crawler 在持仓抓取失败时会保留库内旧持仓，不再用空结果覆盖已有持仓
  - 后端新增“每月 1 日 01:00（Asia/Shanghai）”的既有持仓基金月度刷新任务，用于重抓当前已有持仓数据的基金

## [2026.4.7] - 2026-04-07

### Added
- **集合竞价交易时段**
  - 交易日新增 `call_auction` 市场会话，用于表示 `09:00-09:30` 的集合竞价阶段
  - 已补充交易日历边界测试，覆盖 `09:00`、`09:29`、`09:30` 等关键时间点

### Changed
- **集合竞价阶段的前端展示逻辑**
  - 顶部市场状态现已在交易日 `09:00-09:30` 统一显示“集合竞价中”
  - 首页在集合竞价阶段暂停基金数据请求，等待 `09:30` 开盘后再恢复获取基金数据
  - 首页估值卡、分时走势图、重仓股贡献与持仓明细在集合竞价阶段统一切换为置空 / 禁用态展示
  - 自选页基金卡片的迷你走势图在集合竞价阶段置空，并显示“集合竞价中”
  - 持仓页明细卡片中的实时预估涨跌额在集合竞价阶段统一显示为 `-`

## [2026.4.5] - 2026-04-05

### Added
- **公开“我有想法！”反馈系统**
  - 新增公开的 `/issues` 列表页与 `/issues/:id` 详情页
  - 未登录用户可浏览和搜索公开想法；登录用户可提交新的 bug、功能诉求和改进建议
  - 管理员可将想法状态更新为 `pending` / `accepted` / `completed`
  - 后端新增 `issues` 表及公开查询、登录提交、管理员改状态接口

- **公告与更新日志系统**
  - 新增公开的 `/announcements` 历史公告页与 `/announcements/:id` 详情页
  - 新增 `announcements`、`announcement_reads` 持久化表
  - 支持管理员手动发布公告
  - 支持管理员从 `CHANGELOG.md` 导入公告记录
  - 登录用户存在未读公告时会弹出提醒，并支持标记已读

- **轻量管理员能力**
  - 用户模型新增 `is_admin` 字段
  - 新增管理员鉴权中间件，用于 Issue 状态处理和公告发布 / 导入

- **VIP 前端展示版闭环**
  - 新增 `/vip` 会员介绍页
  - 新增 `/vip/checkout` 开通展示页
  - 新增 `/vip/tasks` 分析任务中心
  - 新增 `/vip/reports/:id` 报告详情页
  - 新增 VIP 样例报告模板与前端状态访问层，用于承载会员状态、额度、任务和报告展示

- **VIP 后端真实骨架**
  - 新增 `user_memberships`、`vip_usage_daily`、`analysis_tasks`、`analysis_reports`、`analysis_report_sources`
  - 新增 `GET /api/v1/vip/membership`、`GET /api/v1/vip/quota`、`GET /api/v1/vip/tasks`、`POST /api/v1/vip/tasks`
  - 新增 `GET /api/v1/vip/reports/:id`，支持读取公开示例报告和当前用户的持久化报告
  - 新增 `POST /api/v1/vip/membership/preview-activate` 与 `POST /api/v1/vip/preview/reset`，用于在真实支付接入前保留后端预览开通链路

- **VIP 支付订单与微信支付接入**
  - 新增 `vip_orders` 持久化表，用于保存 VIP 支付订单、支付状态、微信交易单号和回调原文
  - 新增 `POST /api/v1/vip/orders` 与 `GET /api/v1/vip/orders/:orderId`
  - 新增微信支付 `Native` 下单、查单与回调处理，支付成功后会自动开通或续期 VIP 会员
  - 新增 `POST /api/v1/vip/payments/wechat/notify` 回调入口
  - 新增 `fundlive.yaml` / `fundlive.example.yaml` 中的 `payment.wechat_pay` 配置结构，支持后续补齐商户参数与证书路径

### Changed
- **公开反馈与公告页面体验统一**
  - 公开反馈页面标题统一为“我有想法！”
  - 反馈页的类型 / 状态 / 想法类型选择器改为站内统一的自定义下拉样式，不再使用原生 `select`
  - “想法发送”主按钮补充与站内其他 CTA 一致的动效反馈
  - 站点导航、账户菜单与详情页返回入口的命名统一为“我有想法！”

- **VIP 入口与页面视觉强化**
  - 自选页与持仓页中的 VIP 入口已从禁用占位按钮改成真实可点击入口
  - VIP 导航页签、VIP CTA、会员页 Hero 和开通页价格区做了更明显的高级化视觉增强
  - 用户空间导航新增 `VIP 分析` 页签

- **VIP 状态读取切到后端**
  - `useVIPPreview` 已从 `localStorage` mock 状态切到后端接口，会员状态、每日额度、任务列表与报告详情均改为读取后端数据
  - 自选页与持仓页发起的 VIP 分析任务已改为真实写入 `analysis_tasks`
  - 报告详情页已改为通过后端接口读取持久化报告；报告内容当前仍复用模板化样例结构

- **VIP Checkout 改为真实订单流**
  - `/vip/checkout` 已从“直接预览开通”改为优先创建真实订单并轮询订单状态
  - 当前默认支付方式为微信支付 `Native`；若支付配置未完成，前端会明确提示而不是静默失败
  - 为开发联调保留“预览开通”后备入口，避免在商户参数未补齐时阻塞其他 VIP 功能验证

- **主题显示专项修复**
  - `classic` 主题补充了更接近 Windows light 风格的卡片层次、页签高亮和浅色背景对比度
  - `cyber` 主题完成两轮霓虹层次与控件表现验证，统一了 VIP 区域、弹层和输入控件风格

### Fixed
- **冷基金预热后的前端自动刷新缺失**
  - 修复首页在基金基础资料或分时数据处于 `warming` 状态时，仅显示“稍后自动刷新”但实际不会自动重试的问题
  - `useFund`、`useFundHoldings`、`useFundEstimate` 与 `useTimeSeries` 现已在预热期间自动重新拉取数据，无需手动刷新

- **新基金预热完成后的页面状态回退**
  - 修复切换到新基金后，预热提示结束但页面仍停留在旧基金数据、走势图为空的问题
  - 修复手动刷新后首页回到默认基金 `005827`，导致必须再次搜索才能查看新基金详情的问题
  - 当前选中的基金现已同步到 URL 查询参数，刷新页面会保留当前基金上下文

- **持仓页布局宽度不一致**
  - 修复持仓页顶部“持仓总览”在大屏下未铺满、导致下方内容宽于上方内容的问题

- **Classic 主题下的可读性问题**
  - 修复 VIP 页面“当前开放档位”等模块在浅色主题下出现白底白字的问题
  - 修复 `classic` 主题下多个页面中卡片、页签与状态信息层次不清的问题

## [2026.4.4] - 2026-04-04

### Added
- **受控数据库 migration 与唯一约束**
  - 新增 `schema_migrations` 记录表，用于追踪已应用的 SQL migration
  - 新增基金搜索 migration，统一保障 `pg_trgm` 扩展、基金代码 pattern 索引和名称 / 经理 trigram 索引
  - 新增 `fund_history(fund_id, date)` 与 `fund_time_series(fund_id, time)` 的唯一索引 migration

### Changed
- **冷数据补全切换为后台预热**
  - 读路径不再同步触发外部基金抓取，改为优先读取 warm cache，不命中时调度后台预热
  - `/api/v1/fund/:id/estimate` 与 `/api/v1/fund/:id/timeseries` 在关键数据未就绪时返回 `FUND_DATA_WARMING`，并携带 `Retry-After`
  - 前端首页新增预热提示与自动重试，切换到冷基金时不再因为预热中的 503 回滚到上一只基金

### Fixed
- **自动迁移与约束缺失**
  - 默认数据库配置改为关闭 `AutoMigrate`，避免共享环境在启动期做隐式 schema 变更
  - `fund_history` 与 `fund_time_series` 写入改为基于唯一键 upsert，消除“先查再写”的竞争窗口
  - 当前运行库中的重复 `fund_time_series` 记录已完成去重，再补上唯一约束

- **前端遗留双数据流**
  - 删除未接入主链路的 Zustand `fund-store` 与 `refresh-timer` 旧轮询实现，避免后续误接回造成重复请求

- **行情源切换接口方法错误**
  - 修复前端切换用户级行情源时错误使用 `POST /api/v1/user/quote-source` 的问题
  - 前端现已按后端路由改为调用 `PUT /api/v1/user/quote-source`，解决切换 `Tencent` 来源时报 `404` 的问题

## [2024.4.3] - 2026-04-03

### Added
- **用户级行情数据源切换**
  - 后端新增 `sina` / `tencent` 双行情源支持，未登录用户默认使用 `fundlive.yaml` 中的 `quote.default_source`
  - 用户模型新增 `preferred_quote_source` 字段，登录用户可绑定自己的默认行情源
  - 新增受保护接口 `GET /api/v1/user/quote-source` 与 `PUT /api/v1/user/quote-source`
  - 前端账户菜单新增行情源切换入口，支持登录用户在 `Sina` / `Tencent` 之间切换并即时生效

### Changed
- **估值与分时缓存按数据源隔离**
  - 实时行情缓存 key 改为包含数据源维度，避免不同用户的行情源选择互相污染
  - 分时采集跟踪目标与内存分时 key 改为包含数据源维度，保证同一基金在不同源下独立维护时间序列
  - 后端 viewer 中间件会在每次请求中解析当前生效的数据源，并注入到估值、持仓和分时链路

### Fixed
- **盘前实时估值异常归零**
  - 修复新浪实时快照在盘前 `current=0` 时被直接当作现价使用，导致基金估值显示 `-100%` 的问题
  - 对新浪行情增加现价回退逻辑：`买一价 -> 卖一价 -> 今开 -> 昨收`

- **用户级行情源切换的持久化缺口**
  - 为现有 PostgreSQL `tb_user` 表补充 `preferred_quote_source` 列并设置默认值 `sina`
  - 修复用户登录后切换行情源无法持久化的问题，后续请求会按账号绑定的数据源返回估值结果

## [2026.4.1] - 2026-04-01

### Changed
- **启动与运行配置收敛**
  - 运行时配置语义统一为以项目根目录 `fundlive.yaml` 为主，`docker-compose.yml` 明确标记为本地示例数据库配置
  - 后端新增 `database.log_level` 与 `database.auto_migrate` 配置项，支持按环境控制 GORM 日志级别与自动迁移开关
  - 当前默认推荐使用 `warn` 日志级别，并在稳定环境中关闭自动迁移，降低启动噪音与风险

- **只读链路与共享瞬时补全**
  - 基金详情 / 持仓 / 估值 / 联接基金解析的瞬时补全改为共享同一个进程内缓存实例，减少重复外部抓取
  - 后台分时采集器改为“空启动 + 按请求动态跟踪基金”，不再在启动时扫描全库基金目录
  - 分时采集器空启动时的日志文案改为明确标记 `started idle`，避免误导为已开始采集基金数据

### Fixed
- **读请求写库副作用**
  - 修复 `/api/v1/fund/:id/estimate`、分时回填与联接基金目标 ETF 补抓在只读请求路径上写入数据库的问题
  - 基金按需补全拆分为“只读瞬时抓取”与“显式持久化”两条路径，GET 接口默认只走只读路径

- **基金详情 / 持仓接口口径不一致**
  - 修复 `/api/v1/fund/:id` 与 `/api/v1/fund/:id/holdings` 在目录库数据不完整时返回空基金经理 / 空公司 / 空持仓的问题
  - 修复联接基金 `holdings` 接口与估值链路不一致的问题；当持仓来自目标 ETF 时，响应 `meta.data_source` 会标记 `target_etf:<code>`

- **联接基金失败重试过于频繁**
  - 修复 `fund_mappings` 仅缓存成功、不缓存失败决策的问题
  - 联接基金目标 ETF 解析失败后新增 12 小时冷却窗口，冷却期内不再重复打外部搜索接口
  - 持久化映射时显式刷新 `updated_at`，确保失败冷却时间不依赖 ORM 隐式行为

- **夜间官方净值同步重复执行**
  - 修复服务在北京时间 `23:00` 后重启时会立即重复触发官方净值同步的问题
  - 启动时会先检查当前持仓基金是否已经拥有最新交易日的官方净值，已同步则跳过本次补跑

- **跨域凭据配置错误**
  - 修复 CORS 返回 `Access-Control-Allow-Origin: *` 同时启用 `credentials=true` 的错误组合
  - 改为仅对显式允许的来源返回带凭据的 CORS 头，未知来源的预检请求直接拒绝

## [2026.3.31] - 2026-03-31

### Added
- **持仓交易时间与确认净值日**
  - 用户持仓记录新增 `trade_at` 字段，保存用户录入时选择的交易日期 / 提交时段
  - 后端会基于交易日与北京时间 `15:00` 截止规则自动计算 `as_of_date`
  - 新增用户持仓相关测试，覆盖交易日 `15:00` 前、`15:00` 后与周末顺延场景
  - 新增 `/api/v1/market/pricing-date` 接口，用于返回持仓录入的确认净值日、命中规则和解释文案

- **统一交易日历与净值确认预览接口**
  - 后端新增统一 A 股交易日历引擎，集中处理交易日、节假日、盘前/午休/收盘状态、前后交易日与持仓确认净值日
  - 内置 `2024`-`2026` 上交所法定休市日数据，避免继续只按“周末”判断交易日
  - 新增交易日历测试，覆盖节假日、盘前显示日、`15:00` 截止边界与持仓确认净值日解析

- **夜间官方净值同步**
  - 正式接入 `fund_history` 作为基金官方日净值 / 日涨跌幅历史表
  - 新增夜间 `23:00` 官方净值同步服务，仅针对当前用户持仓涉及的基金抓取最新净值
  - 同步完成后，持仓列表会优先展示最新官方日涨跌幅，替换原有的预估值展示

- **自选 / 持仓页交互增强**
  - 持仓录入支持交易日期选择 + `15:00 前 / 15:00 后` 两段式提交时段
  - 持仓页新增确认净值日实时预览，录入前即可看到将按哪一天收盘净值确认
  - 自选页分组选择器改为自定义下拉菜单，不再依赖浏览器原生白底 `select`
  - 自选卡片迷你走势图新增悬停提示，可显示当前点位对应的涨跌幅

### Changed
- **用户工作区布局**
  - 持仓总览区重排为“上方三输入框、下方搜索结果 / 录入信息 / 交易时间”的信息结构，降低桌面端的割裂感
  - 自选页与持仓页的 AI/VIP 入口统一下移到页面底部，避免干扰主要操作流
  - 账户工作区右上角隐藏无效的“专业模式 / 极简模式”切换，仅保留主题切换

- **交易规则源彻底统一**
  - `market/status`、分时日期选择、持仓入库时的 `as_of_date` 计算统一改为复用 `internal/trading` 的单一规则源
  - 持仓页不再在前端本地推导确认净值日，而是改为实时请求后端预览结果
  - 首页、自选卡片、分时图和刷新时机不再依赖浏览器本地交易时段判断，统一改为消费后端市场状态
  - 前端市场状态 hook 改为共享后端状态快照与边界刷新调度，避免在多个基金卡片上重复生成本地规则与多套定时器

- **用户页数据读取与分时写入性能**
  - 自选基金、收藏基金和持仓列表改为批量加载基金详情与最新官方净值，减少列表页逐条回查形成的 `N+1` 查询
  - 自选页分组基金改为按分组 ID 批量读取，不再按分组逐个查询分组内基金
  - 估值请求不再为每个分时点异步单点落库，改为仅在内存中维护 5 分钟对齐桶位，数据库继续保留规范化回补后的分时数据

- **账户工作区视觉反馈**
  - 自选页创建分组、删除分组、基金卡片删除按钮、查看详情按钮补充悬停、按压、扫光和删除态动画
  - 自选页 / 持仓页切换标签增加激活态与悬停动效

### Fixed
- **持仓录入运行时错误**
  - 修复未选中基金时直接提交持仓导致前端抛出 `fund not found` 的问题
  - `载入演示持仓` 不再因已有自选分组而提前返回，持仓为空时仍可正常导入演示数据

- **多套交易规则导致的业务不一致**
  - 修复前端首页、本地持仓预览、后端持仓入库各自计算交易状态与确认净值日的问题
  - 修复浏览器不处于北京时间时，前端交易状态与轮询频率可能判断错误的问题
  - 修复节假日场景下仅按周末判断交易日，导致显示日、轮询与确认净值日都可能出错的问题

- **首页首屏市场状态误导**
  - 修复首页在客户端状态尚未挂载时先展示默认 `盘前` 的问题
  - 首页首屏改为显示 `加载中...`，待后端市场状态返回后再切换为真实交易状态

- **原生控件风格割裂**
  - 修复持仓页日期时间控件与站点整体视觉不一致的问题
  - 修复自选页分组选择器展开后仍出现浏览器原生白底菜单的问题

## [2026.03.30] - 2026-03-30

### Added
- **用户模块基础数据层**
  - 新增纯领域层用户模型与仓储接口，覆盖用户、会话、自选基金、用户持仓修正
  - 新增 `internal/database/user_models.go`，将 `tb_user`、`tb_user_session`、`tb_user_favorite_fund`、`tb_user_holding_override` 的 GORM 模型集中放在基础设施层
  - 新增 PostgreSQL / 内存用户仓储实现，为后续注册登录、Google 登录、自选基金与用户持仓功能提供数据落点

- **邮箱密码认证与账户页面**
  - 新增服务端会话认证流程，提供 `/api/v1/auth/register`、`/api/v1/auth/login`、`/api/v1/auth/me`、`/api/v1/auth/logout`
  - 使用 HttpOnly Cookie + `tb_user_session` 存储登录态，密码仅保存哈希值
  - 新增 `/auth/login`、`/auth/register` 页面，并抽出共享 UI 偏好 hook，让登录页、注册页与首页共享三套主题风格
  - 修复首页主题状态分散和基金切换状态管理触发的前端 lint 问题

- **Google 登录与自动注册**
  - 新增 `/api/v1/auth/google`，后端会校验 Google ID Token 的签名、`iss`、`aud`、`exp`
  - 新增 Google 公钥拉取与缓存逻辑，基于 Google JWKS 校验 `RS256` 签名
  - Google 首次登录时自动注册本地账户；如邮箱已存在，则自动绑定并升级为 `hybrid` 账号
  - 登录页新增 Google Identity Services 按钮，支持前端直接提交 `credential` 到后端

- **用户偏好接口**
  - 新增受保护接口 `/api/v1/user/favorites` 与 `/api/v1/user/funds/:fundId/holding-overrides`
  - 支持收藏基金的新增、删除、列表读取
  - 支持用户持仓修正的整组替换、读取与基础校验（代码、交易所、持仓占比）

- **首页登录态入口**
  - 首页头部的登录/注册按钮在用户登录后会替换为“头像 + 用户名”账户菜单
  - 账户菜单支持主题一致的下拉展示与退出登录操作

- **联接基金穿透查询** (`FundResolver`)
  - 联接基金（如"华宝创业板人工智能ETF联接C"）自动解析目标 ETF
  - 当联接基金无直接持仓时，优先查询 `fund_mappings` 表，若无则通过东方财富搜索解析目标 ETF
  - 映射关系保存到 `fund_mappings` 表，后续请求无需重复解析
  - **支持无持仓 ETF 估值**：针对黄金 ETF、QDII ETF 等无股票持仓的基金，直接使用目标 ETF 的实时行情进行估值
  - 新增 `internal/service/fund_resolver.go`
  - `ValuationService` 新增 `SetFundResolver()` 方法

- **股票名称乱码修复工具** (`StockNameFixer`)
  - 新增 `internal/crawler/stock_name_fixer.go`
  - `--fix-names`: 检测并修复数据库中乱码的股票名称
  - `--fix-all-names`: 刷新所有股票名称（从新浪 API 获取）
  - 使用新浪财经 API 获取正确的 UTF-8 编码股票名称

- **基金切换加载指示器** (`FundLoadingIndicator`)
  - 新增 `web/src/components/loading-indicator.tsx`
  - 用户切换基金时显示全屏加载动画，提升用户体验
  - 动画包含旋转进度环、跳动圆点等视觉效果
  - 数据加载完成或 15 秒超时后自动关闭
  - 避免用户困惑"是系统正在加载还是出错"


- **项目启动配置文件** (`fundlive.yaml`)
  - 新增 `internal/appconfig/` 统一加载启动配置
  - `cmd/server`、数据库初始化和 crawler 自动复用 `fundlive.yaml`
  - 新增 `fundlive.example.yaml` 示例模板

- **按需基金数据补抓** (`FundDataLoader`)
  - 新增 `internal/service/fund_data_loader.go`
  - 对仅导入基金目录、未导入详情/持仓的基金，在估值请求时自动补抓并落库
  - 避免用户首次选择新基金时因缺少持仓而直接报错


- **商品 / 期货基金估值配置** (`fund_valuation_profiles`)
  - 新增 `FundValuationProfile` 数据模型，用于为非股票持仓型基金配置定价方式与底层标的
  - 新增 `ValuationProfileStore` 与 `futures_underlying` 定价路径
  - 默认种入国投瑞银白银期货(LOF)A / C 的白银期货主力合约配置

### Changed
- **联接基金解析改为非 AI 路径** (`FundResolver`)
  - 优先通过东方财富搜索解析目标 ETF，不再依赖 AI Agent 才能完成联接基金估值
  - 解析结果仍保存到 `fund_mappings` 表，后续请求可直接复用
  - 若目标 ETF 本地无持仓，会继续按需补抓其详情和持仓数据

- **后端启动与配置方式**
  - Go 版本固定为 `1.26.3`
  - 后端启动默认读取 `fundlive.yaml`，无需每次手动传数据库环境变量
  - 当仓库中基金数量过大时，后台分时采集自动退回默认观察名单，避免服务启动后被全量目录拖慢

### Removed
- **AI 运行时代码与接口**
  - 删除 `internal/agent/`、`cmd/agent/`、`internal/handler/agent_handler.go`
  - 移除 `/api/v1/agent/*` 路由与 `agent.yaml` / `agent.example.yaml`
  - 后端运行不再依赖 OpenAI / Ark 配置

- **未使用的后端代码与文件**
  - 删除 `internal/crawler/fund_parser.go`
  - 删除未使用的 `QuoteProvider.GetRealTimePrices`、`CacheRepository.Delete` 等接口方法
  - 删除未使用的 `FundHistory` 模型、历史净值仓储方法与若干未引用的 trading/service 辅助函数

### Fixed
- **联接基金估值 500 错误**
  - 例如 `023408` 这类仅有基金目录、无详情/持仓的联接基金，现在会自动补抓基金信息并解析目标 ETF
  - 不再因为缺少 AI 配置或本地持仓数据而在 `/api/v1/fund/:id/estimate` 返回 500

- **商品 / 期货基金估值 500 错误**
  - 例如 `161226`、`019005` 这类白银期货基金，不再因缺少股票持仓而直接返回 500
  - 后端会改用 `fund_valuation_profiles` 中配置的底层期货标的进行估值
  - 对尚未配置估值档案的商品基金，改为返回明确的 `UNSUPPORTED_PRICING_MODEL`

- **前端调用后端接口失败**
  - 前端改为相对路径调用，并通过 Next.js rewrite 同源代理到后端
  - 避免浏览器直接请求 `http://localhost:8080` 带来的跨域、宿主机或端口差异问题

- **React Hydration 错误** (`MarketStatusIndicator`)
  - 问题：服务器渲染的时间与客户端 hydrate 时不匹配（如 "43分7秒" vs "43分8秒"）
  - 原因：`useMarketStatus` hook 初始化时使用 `new Date()` 动态计算状态
  - 解决方案：
    - 新增 `createInitialStatus()` 函数生成稳定的初始状态
    - 添加 `mounted` 状态标记，仅在客户端 `useEffect` 执行后为 `true`
    - 组件在 `mounted === false` 时显示占位符，避免 SSR/CSR 内容不匹配
  - 影响文件：
    - `web/src/hooks/use-market-status.ts`
    - `web/src/components/market-status-indicator.tsx`

- **Dark / Cyber 主题下的搜索与切换面板可读性**
  - 专业模式切换和主题切换下拉面板改为不透明背景，避免滚动时与页面内容重叠
  - Dark 主题下搜索输入框背景改为更深的黑色系底色，提升输入区域识别度

- **北交所(BJ)股票名称乱码** (`SinaFinanceProvider`)
  - 问题：以 `92xxxx`、`43xxxx`、`83xxxx` 等开头的北交所股票无法正确获取股票名称
  - 原因：`buildSinaSymbol` 函数只支持上海(sh)和深圳(sz)交易所
  - 解决方案：
    - 更新 `buildSinaSymbol` 函数，支持北交所股票代码前缀 `bj`
    - 股票代码规则：`6` 开头→上海，`4/8/9` 开头→北交所，其他→深圳
    - 新增 `parseQuoteByExchange` 和 `parseBJQuote` 方法处理北交所数据格式
    - 北交所股票返回字段数量可能少于沪深，降低最小字段数要求
  - 影响文件：
    - `internal/adapter/sina_provider.go`

### Usage

**修复股票名称乱码**
```bash
# 仅修复检测到的乱码名称
go run ./cmd/crawler --fix-names

# 刷新所有股票名称
go run ./cmd/crawler --fix-all-names
```

**项目启动配置**
复制 `fundlive.example.yaml` 为 `fundlive.yaml` 后，`go run ./cmd/server` 会自动读取数据库与服务配置，无需手工传环境变量

**联接基金穿透查询**
无需 AI 配置。系统会优先通过东方财富搜索自动解析联接基金的目标 ETF，并将结果保存到 `fund_mappings`。

---

## [3.4.0] - 2026-02-02

### Added
- **AI Agent (CloudWeGo Eino)**
  - 新增 `internal/agent/` 模块，使用 CloudWeGo Eino 框架构建 AI Agent
  - **FundSearch Tool**: 封装东财基金搜索 API 为 Eino Tool
    - 接口: `fund_search(query: string) -> JSON`
    - 返回基金代码、名称、类型等信息
  - **FundRelationAgent**: ETF 联接基金关系解析 Agent
    - 使用 LLM Function Calling 自动调用工具
    - 输入联接基金名称，输出目标 ETF 代码
    - 支持 OpenAI 协议兼容的 LLM（通过环境变量或配置文件配置）
  - **AgentJob**: 批量执行任务
    - 查询所有未解析的联接基金
    - 循环调用 Agent 并持久化结果
- **FundMapping 数据库模型**
  - 存储联接基金与目标 ETF 的映射关系
  - 字段: `feeder_code`, `target_code`, `is_resolved`, `resolved_at`
- **Agent CLI 工具** (`cmd/agent/main.go`)
  - `-fund <name>`: 解析单个基金的目标 ETF
  - `-job`: 批量执行所有未解析的联接基金
  - `-stats`: 显示映射统计信息
- **YAML 配置文件支持** (`internal/agent/config.go`)
  - 支持从 `agent.yaml` 配置文件加载 OpenAI 配置
  - 配置优先级: 环境变量 > 配置文件 > 默认值
  - 自动搜索配置文件路径: `./agent.yaml`, `~/.fundlive/agent.yaml` 等
  - 提供示例配置文件 `agent.yaml.example`
- **Agent HTTP API** (`internal/handler/agent_handler.go`)
  - `POST /api/v1/agent/resolve` - 解析联接基金的目标 ETF
  - `GET /api/v1/agent/status` - 获取 Agent 状态
  - 支持 JSON 请求/响应
  - 未配置 API Key 时返回 503 状态

### Configuration

**方式 1: 环境变量**
```bash
export OPENAI_API_KEY="sk-your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"  # 可选
export OPENAI_MODEL_NAME="gpt-4o-mini"              # 可选
```

**方式 2: 配置文件 (`agent.yaml`)**
```yaml
openai:
  api_key: "sk-your-api-key"
  base_url: "https://api.openai.com/v1"  # 可选
  model: "gpt-4o-mini"                   # 可选
```

### Usage Examples
```bash
# 解析单个基金
go run ./cmd/agent -fund "华宝创业板人工智能ETF联接C"

# 批量解析所有联接基金
go run ./cmd/agent -job

# 查看统计
go run ./cmd/agent -stats
```

### Technical Details
- **框架**: CloudWeGo Eino (`github.com/cloudwego/eino`)
- **LLM 组件**: `eino-ext/components/model/openai`
- **配置加载**: `gopkg.in/yaml.v3`
- **Tool 模式**: `utils.InferTool` 自动推断参数 schema
- **Agent 循环**: ChatModel → ToolCalls → ToolsNode → 结果回填 → 最终答案

### Files Added
- `internal/agent/tools/fund_search.go` - 东财搜索 Tool
- `internal/agent/resolver.go` - Agent 主逻辑
- `internal/agent/job.go` - 批量任务
- `internal/agent/config.go` - 配置文件加载
- `cmd/agent/main.go` - CLI 入口
- `agent.yaml.example` - 示例配置文件

---

## [3.3.0] - 2026-02-02

### Added
- **时序数据持久化**
  - 新增 `FundTimeSeries` 数据库模型，存储分时走势数据
  - 时序点同时保存到内存（快速访问）和数据库（持久化）
  - 支持服务重启后从数据库恢复历史数据
  - 自动清理 7 天前的历史数据
- **后台数据采集器**
  - 新增 `StartBackgroundCollector` 方法
  - 交易时段每分钟自动采集所有基金的估值数据
  - 支持从数据库动态获取基金列表（空参数时自动获取）
  - 确保从开盘（09:30）开始有完整的分时数据
- **基金列表抓取功能**
  - 新增 `internal/crawler/fund_list.go`
  - `FetchAllFunds`: 获取市场全部基金（约 1 万只）
  - `FetchStockFunds`: 仅获取股票型+混合型基金
  - `FetchPopularFunds`: 获取预设的 20 只热门基金
  - Crawler CLI 新增 `--list` 参数：`all`, `stock`, `popular`
  - Crawler CLI 新增 `--limit` 参数，限制抓取数量

### Changed
- `FundRepository` 接口扩展：
  - 新增 `GetAllFundIDs()` 方法
  - 新增 `SaveTimeSeriesPoint()` 方法
  - 新增 `GetTimeSeriesByDate()` 方法
- `ValuationService.GetIntradayTimeSeries()` 现在支持数据库回退
- `cmd/crawler/main.go` 超时时间增加到 120 秒

### Fixed
- **分时图表渲染问题**
  - 修复时间解析使用 `toLocaleTimeString` 可能包含秒数的问题
  - 新增 `roundToNearestFiveMinutes()` 函数，将数据点舍入到 5 分钟槽位
  - 重构 `ChartContent` 组件，使用 `morningChange`/`afternoonChange` 独立字段
  - 改用 `<Line>` 组件替代 `<Area>` with data prop
- **Crawler 数据库保存超时问题**
  - 修复抓取超时导致数据库保存也失败的问题
  - 为数据库操作创建独立 context（10 分钟超时）
  - 优化日志输出：每 100 个基金打印一次进度
  - 跳过无效基金时不再逐条打印日志

### Technical Details
- 时序数据表结构：`fund_time_series (id, fund_id, date, time, change_percent, estimate_nav)`
- 复合索引：`idx_fund_time (fund_id, date, time)` 优化查询性能
- 异步持久化：不阻塞主请求流程
- 内存缓存：首次从数据库加载后缓存到内存
- 测试结果：`-list stock` 模式成功保存 512 只基金到数据库

---

## [3.2.0] - 2026-02-02

### Added
- **PostgreSQL Docker 环境**
  - 新增 `docker-compose.yml`，仅容器化数据库服务
  - 使用 `postgres:15-alpine` 镜像
  - 端口映射 `15432:5432`，支持宿主机 Go 程序直连
  - 数据持久化挂载 `./pgdata`，防止重启丢失数据
  - 设置上海时区 `TZ=Asia/Shanghai`
  - 添加健康检查 (`pg_isready`)
- **GORM 集成与自动迁移**
  - 新增 `internal/database/` 模块
  - `db.go`: 数据库连接、连接池配置、AutoMigrate 逻辑
  - `models.go`: 数据库专用模型 (`Fund`, `StockHolding`, `FundHistory`)
  - 完整 GORM 标签：主键、索引、类型定义、表关系
  - 支持环境变量配置 (`DB_HOST`, `DB_PORT`, `DB_USER`, etc.)
- **PostgresFundRepository**
  - 新增 `internal/repository/postgres_fund_repo.go`
  - 实现 `FundRepository` 接口，支持 CRUD 操作
  - Upsert 逻辑 (ON CONFLICT)，事务支持
  - domain/database 模型转换

### Changed
- `cmd/server/main.go`: 支持 `STORAGE_MODE` 环境变量切换存储模式
- `cmd/crawler/main.go`: 新增 `--save-db` 参数，抓取数据可直接入库
- Server 添加优雅关闭 (Graceful Shutdown)

### Technical Details
- **开发模式**: Local Go + Dockerized DB
- **连接方式**: Go 后端通过 `localhost:15432` 连接容器数据库
- **架构**: 数据库模型与 domain 模型分离，保持关注点分离
- **表结构**: `funds`, `stock_holdings`, `fund_history` (AutoMigrate 自动创建)


---

## [3.1.0] - 2026-02-01

### Added
- `CHANGELOG.md` for tracking project evolution
- **Smart Data Fallback** for non-trading days:
  - Backend automatically detects non-trading days (weekends/holidays)
  - Falls back to the most recent trading day's data
  - Returns `display_date` and `is_historical` fields for frontend context
- **Fixed X-Axis Domain** (09:30-15:00) in chart component:
  - X-axis always spans full trading day regardless of available data
  - Pre-market shows empty grid with correct time range
- **Lunch Break Gap Handling** (11:30-13:00):
  - Morning and afternoon sessions rendered as separate series
  - No diagonal line connecting 11:30 to 13:00
  - Visual gap clearly indicates lunch break
- Date-indexed time series storage for historical data retention

### Changed
- `ValuationService.GetIntradayTimeSeries()` now accepts optional date parameter
- `intraday-chart.tsx` completely refactored with A-Share specific rendering
- Time series data now uses composite keys (fundID + date) for proper fallback
- Chart displays "上一交易日" indicator when showing historical data

### Technical Details
- Added `TimeSeriesStorage` with date-indexed map structure
- Chart uses custom `generateTradingDayTicks()` for X-axis domain
- Implemented `splitByLunchBreak()` for dual-series rendering

---

## [3.0.0] - 2026-02-01

### Added
- **Crawler Module** (`internal/crawler/`)
  - Eastmoney data source integration
  - Fund info parser (`pingzhongdata/*.js`)
  - Holdings HTML table parser
  - GBK to UTF-8 encoding conversion
  - Stock name mapping fallback
- CLI crawler tool (`cmd/crawler/main.go`)
- Concurrent crawling with rate limiting
- Real fund data: 易方达蓝筹、中欧医疗、诺安成长

### Changed
- Replaced mock data with real crawled data
- Fund holdings now fetched from Eastmoney API

---

## [2.0.0] - 2026-01-31

### Added
- **Smart Polling Strategy**
  - `useMarketStatus` hook for A-share trading hours detection
  - Trading hours: Mon-Fri, 09:30-11:30, 13:00-15:00
  - Non-trading hours: Fetch once, disable polling
- **SWR Integration**
  - Replaced manual `useEffect` fetch with `useSWR`
  - `keepPreviousData: true` to prevent UI flashing
  - Dynamic `refreshInterval` based on market status
- **React 18 Performance**
  - `useTransition` for non-blocking UI updates
  - `useMemo` for chart data optimization
  - `memo` wrapped chart components
- `useDebounce` hook for search input
- `MarketStatusIndicator` component

### Fixed
- **Classic Theme Visibility Bug**
  - Card background: `rgba(0,0,0,0.02)` → `#ffffff`
  - Text color: proper contrast with `--text-primary: #0f172a`
  - Added `box-shadow` and `border` for card distinction

### Changed
- Refactored all components to use props instead of Zustand store
- Theme-aware CSS classes (`text-theme-primary`, `text-up`, `text-down`)
- Removed dependency on global state for data fetching

---

## [1.0.0] - 2026-01-30

### Added
- **Backend (Go)**
  - Gin HTTP framework
  - Clean Architecture: Handler → Service → Repository
  - Domain models: `Fund`, `StockHolding`, `StockQuote`, `FundEstimate`
  - `decimal.Decimal` for precise financial calculations
  - Sina Finance real-time quote provider
  - In-memory cache with 60s TTL (`go-cache`)
  - `errgroup` for concurrent quote fetching
  - RESTful API endpoints:
    - `GET /api/v1/fund/search`
    - `GET /api/v1/fund/:id`
    - `GET /api/v1/fund/:id/estimate`
    - `GET /api/v1/fund/:id/holdings`
    - `GET /api/v1/fund/:id/timeseries`
- **Frontend (Next.js 14+)**
  - App Router architecture
  - Tailwind CSS styling
  - Recharts for data visualization
  - Multi-theme system: Classic, Dark, Cyber
  - Components: EstimateCard, IntradayChart, HoldingsTable, FundSearch
  - Zustand for state management (v1)

### Technical Stack
- **Backend**: Go 1.21+, Gin, go-resty, shopspring/decimal
- **Frontend**: Next.js 14+, React 18, Tailwind CSS, SWR, Recharts

---

## Project Links

- **Repository**: https://github.com/RomaticDOG/fund-live
- **Documentation**: See `README.md`

---

## Version Naming Convention

- **Major (X.0.0)**: Breaking changes, new architecture
- **Minor (0.X.0)**: New features, enhancements
- **Patch (0.0.X)**: Bug fixes, small improvements
