# Changelog

All notable changes to the **FundLive** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

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
  - Go 版本固定为 `1.25.8`
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
