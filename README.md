# FundLive - 实时基金估值系统

[![Go](https://img.shields.io/badge/Go-1.25.8-00ADD8?logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-16+-000000?logo=next.js)](https://nextjs.org)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind-4+-06B6D4?logo=tailwindcss)](https://tailwindcss.com)

> 通过基金前十大重仓股和联接基金目标 ETF 的实时行情，计算基金盘中预估涨跌幅，并补全分时走势图。

当前文档对应版本：`2026.4.18`

## 2026.4.18 更新摘要

- 自选、持仓、想法、公告等页面在移动端滚动离开顶部后，会自动隐藏上方标题、描述与导航区块，减少小屏幕内容被长期占用
- 页面新增移动端“回到顶部”按钮，点击后可平滑回到顶部并重新显示顶部模块
- 该优化统一落在共用页面壳组件中，桌面端布局保持不变

## 2026.4.17 更新摘要

- 修复自选页删除分组时报 `500 Internal Server Error` 的问题
- PostgreSQL 下删除分组的仓储实现已改为“先删分组内基金，再删分组本身”的兼容写法
- 删除单只自选基金的仓储过滤方式也同步调整，避免出现同类数据库兼容性错误
- 联接基金在存在“0% 自有持仓”脏数据时，不再被错误判定为已有持仓；当前会继续回退到目标 ETF 持仓与估值链路
- QDII 海外基金现在已支持海外持仓实时估值；像 `017437` 这类基金可返回真实海外持仓涨跌幅与贡献值
- 海外股票现已固定走独立数据源 `overseas_fixed`，不再受用户 `sina / tencent` 行情源切换影响
- 首页卡片与走势图现已切到统一 `dashboard` 快照链路，且都按 30 秒刷新；收盘后 / 周末图末 15:00 点会与卡片 estimate 保持一致

## 2026.4.16 更新摘要

- 持仓页现已完成“本金口径 + 真实净值口径”拆分：`amount` 继续表示用户录入本金，不再和真实市值混用
- 后端已为用户持仓补充 `shares`、`confirmed_nav`、`confirmed_nav_date`，创建持仓时会优先按确认净值日补齐份额
- 持仓列表接口已升级为 `items + summary`，支持返回单只基金的真实当前市值、今日盈亏、今日涨跌幅与总仓汇总
- 持仓页顶部新增总本金 / 总当前市值 / 总今日盈亏 / 总今日涨跌幅四张总览卡
- 当部分持仓缺少确认净值或最新官方净值时，页面会显示降级文案，不再把盘中预估和夜间真实口径混算到同一位置
- 联接基金目标 ETF 解析已升级为“详情页相关 ETF 优先、搜索 fallback 增强”的通用链路；失败冷却时间从 12 小时缩短到 30 分钟

## 2026.4.9 更新摘要

- 基金分时走势图与顶部实时估值现已统一到同一套估值快照内核，图表最后一个 `15:00` 点会和上方 estimate 保持一致
- 含港股重仓的基金现在会在分时走势图回放中纳入港股分钟数据；像易方达蓝筹精选混合这类基金不再只反映 A 股权重
- 交易日 `15:00` 后分时图会继续展示**当天**曲线，不会再错误回退到上一交易日
- 分时走势图午休衔接已统一：`13:00` 固定承接 `11:30` 的上午收盘点位，`13:05` 起再继续显示下午真实波动
- 已对受影响的当天与部分可安全重建的历史分时数据完成回补；对无法可靠重建的旧会话保留原始记录，避免误删
- 空 PostgreSQL 库首启现已支持通过受控 SQL migration 自动创建核心基金表和用户表，不再依赖临时开启 `database.auto_migrate=true`
- 首页切基金时的全屏加载态与市场状态占位已统一使用主题变量，`classic / dark / cyber` 下的非 VIP 页面主题一致性进一步收口

## 2026.4.8 更新摘要

- 公开“我有想法！”详情页新增官方回复区块：管理员可公开回复每条想法的处理说明、修改点和进展
- 管理员现在可在想法详情页一并更新处理状态与官方回复，不再只有“待接收 / 处理中 / 已完成”三个状态标签
- 后端新增 `PUT /api/v1/admin/issues/:id/reply` 接口，并将官方回复写入 `issues` 表
- 基金首页主卡片新增基金代码展示，原“前十大重仓股”模块改为更准确的“重仓股明细”表达，并显示当前参与估值展示的数量
- 修复新浪行情请求中部分个股无法返回实时行情的问题；例如 `002027` 现已可正常参与估值计算
- 基金持仓解析现已支持港股 5 位代码，像易方达蓝筹精选混合这类含港股重仓的基金可以展示更完整的重仓股明细
- 修复东财持仓明细在内容已是 UTF-8 时被误按 GBK 转码导致的中文名称乱码问题
- crawler 在持仓抓取失败时会保留已有持仓，不再用空结果覆盖数据库中的旧持仓
- 后端新增每月 1 日 01:00（Asia/Shanghai）的既有持仓基金月度刷新任务，用于重抓当前已有持仓的基金

## 2026.4.7 更新摘要

- 新增交易日 `09:00-09:30` 的集合竞价时段识别，市场状态会统一显示“集合竞价中”
- 首页在集合竞价阶段暂停基金数据请求，开盘后再恢复获取和展示基金数据
- 首页估值卡、分时走势图、重仓股贡献与持仓明细在集合竞价阶段统一置空或进入禁用态
- 自选页迷你走势图在集合竞价阶段置空，并显示“集合竞价中”
- 持仓页明细中的实时预估涨跌额在集合竞价阶段统一显示为 `-`

## 2026.4.5 更新摘要

- 新增公开“我有想法！”反馈系统：游客可浏览与搜索，登录用户可提交 bug / 功能诉求 / 改进建议，管理员可更新处理状态
- 新增公告与更新日志系统：支持历史公告页、公告详情页、登录后未读公告弹窗与已读记录
- 新增轻量管理员能力：用户模型新增 `is_admin` 字段，用于 Issue 状态处理和公告发布 / 导入
- 公开反馈页已统一命名为“我有想法！”，并将筛选器重写为站内统一风格的自定义下拉，发送按钮补充了动效反馈
- 修复冷基金预热期间首页只提示“自动刷新”但不会真正重试的问题，当前预热链路会自动重新拉取基础资料、估值和分时数据
- 修复切换新基金后预热完成仍停留在旧基金数据的问题；当前选中的基金现已同步到 URL，刷新页面不会再回到默认基金
- 新增 VIP 前端页面闭环：会员介绍页、开通页、任务中心、报告详情页，以及自选/持仓页的 VIP 入口
- 自选页和持仓页中的 VIP 入口已从占位按钮改成真实可点击入口，并接入会员状态、任务与报告流转；报告内容当前仍以模板化样例驱动
- 新增 VIP 后端基础骨架：`user_memberships`、`vip_usage_daily`、`analysis_tasks`、`analysis_reports`、`analysis_report_sources`
- 新增 VIP 后端接口：会员状态、剩余额度、任务创建 / 列表、报告详情，以及预览开通 / 重置接口
- 前端 `useVIPPreview` 已切到后端真实数据读取；VIP 页面当前是“后端真实状态 + 模板化报告内容 + 真实订单流”的过渡形态
- 新增 `vip_orders` 与微信支付 `Native` 下单 / 查单 / 回调链路，支付成功后会自动开通或续期 VIP 会员
- `/vip/checkout` 已改为真实订单流：优先创建微信支付订单并轮询状态，支付配置缺失时会明确提示
- VIP 页面与按钮已做高级化视觉增强，和普通功能区形成明显区分
- 修复持仓页“持仓总览”和下方内容宽度不一致的问题
- 对 `classic` 与 `cyber` 主题做了专项可读性与层次修复，尤其增强了 `classic` 主题下的卡片、页签和 VIP 页面文字可读性

## 2026.4.4 更新摘要

- 搜索链路已改为“精确 / 前缀优先 + 模糊补充”，并补充 PostgreSQL `pg_trgm` / pattern 索引
- 冷基金数据补全已改为后台预热，前端会识别 `FUND_DATA_WARMING` 并自动重试
- 数据库默认自动迁移已关闭，启动时改为执行受控 SQL migration，并记录到 `schema_migrations`
- 已为 `fund_history(fund_id,date)` 与 `fund_time_series(fund_id,time)` 补充唯一约束；运行库中的重复分时记录已完成去重
- 前端已移除遗留的 Zustand 轮询实现，当前主数据流统一由 SWR 驱动
- 修复用户级行情源切换的请求方法错误；切换 `Tencent` 来源不再因为调用错误接口而返回 `404`

## 2026.4.3 更新摘要

- 后端新增 `sina` / `tencent` 双行情源支持，未登录用户按 `quote.default_source` 走默认源
- 登录用户现在可以在账户菜单中切换自己的行情数据源，偏好会持久化到后端账号
- 实时行情缓存、分时采集跟踪和分时内存 key 已按数据源隔离，避免不同用户互相污染
- 修复新浪盘前 `current=0` 导致基金估值显示 `-100%` 的问题，现价会按盘口 / 今开 / 昨收回退
- 用户级行情源切换联调已通过：切换后估值接口 `data_source` 会跟随返回对应源

## 项目简介

公募基金净值通常按日更新，盘中无法直接看到当日变化。FundLive 的目标是：

- 搜索基金并查看盘中实时预估涨跌幅
- 支持普通基金和 ETF 联接基金
- 展示重仓股贡献明细
- 展示盘中分时走势
- 为首次打开的基金按需补抓基础数据和持仓数据

当前项目已经移除了 AI 运行时依赖。联接基金解析会优先读取基金详情页中的“相关 ETF / 跟踪标的”线索，再回退到东方财富搜索，解析结果持久化到 `fund_mappings` 表。
当前版本支持邮箱密码登录、Google 登录、自选基金、分组管理、用户持仓修正，以及带交易时间的持仓录入、夜间真实净值同步与确认净值日推导。

## 当前能力

### 后端
- Gin API 服务
- PostgreSQL 持久化存储
- 以 `fundlive.yaml` 为主的启动配置，支持数据库日志级别与自动迁移开关
- 用户体系：邮箱密码登录、Google 登录、服务端 Session、登出、当前用户信息
- 用户偏好：自选基金、用户持仓修正、带交易时间的用户持仓记录
- 用户持仓现已支持持久化确认净值 / 份额，并可在官方净值同步后返回真实当前市值、今日盈亏、今日涨跌幅与总仓汇总
- `fund_history` 存储基金官方日净值、累计净值与日涨跌幅，夜间定时同步用户持仓涉及基金的真实值
- 基金目录、基金详情、持仓、分时点管理
- 支持 `sina` / `tencent` 双行情源，并可按用户绑定默认行情源
- 联接基金目标 ETF 自动解析与映射缓存
- 联接基金目标解析现优先使用详情页 `查看相关ETF`，并结合 `跟踪标的` 做搜索增强
- 联接基金失败映射带冷却重试窗口，默认冷却时间为 30 分钟，避免高频重复外呼同时减少长时间不可用
- 联接基金若只存在 `holding_ratio=0` 的无效自有持仓，系统会继续回退到目标 ETF 持仓与估值链路
- QDII 海外基金现已支持展示海外股票持仓详情；未接入海外实时行情时会返回明确的降级提示，而不是直接报错
- 商品/期货基金通过 `fund_valuation_profiles` 走自定义标的估值
- 分时数据缺失时自动回填
- 基金重仓股解析现已支持 A 股与港股 5 位代码混合持仓场景
- 已有持仓基金会在每月 1 日 01:00（Asia/Shanghai）自动执行一次月度持仓刷新
- 使用新浪 5 分钟 K 线回补盘中走势
- 港股重仓基金的分时走势图现已通过统一估值回放链路计入港股分钟数据，并与顶部实时估值口径保持一致
- 午休缺失 `13:00` 点时自动补点
- 基金详情 / 持仓 / 估值 / 联接基金解析共用只读瞬时补全缓存
- 冷基金数据改为后台预热，预热状态会通过 `meta.cache_status` / `FUND_DATA_WARMING` 返回前端
- 用户持仓按交易日和北京时间 `15:00` 截止自动计算确认净值日
- 受控 SQL migration：启动时会检查 `schema_migrations`、搜索索引与关键唯一约束
- VIP 基础后端：会员状态、每日额度、分析任务、报告详情与来源列表已具备持久化数据结构
- 支持 VIP 预览开通 / 重置链路，在真实支付接入前先使用后端保存会员状态
- VIP 支付后端：已支持 `vip_orders`、微信支付 `Native` 下单、订单状态查询、微信回调处理，以及支付成功后自动开通 / 续期会员
- 公开“我有想法！”系统：支持游客浏览、筛选和搜索；登录用户可提交 bug / 功能诉求 / 改进建议；管理员可更新处理状态并公开回复处理说明
- 公告系统：支持历史公告展示、管理员手动发布、从 `CHANGELOG.md` 导入，以及登录后未读公告弹窗 / 已读记录

### 前端
- Next.js 16 App Router
- SWR 数据获取
- Recharts 分时图表
- VIP 前端页面：会员介绍页、开通页、任务中心、报告详情页
- VIP 页面当前已切到后端真实会员 / 额度 / 任务 / 报告接口；订单与支付状态也由后端驱动
- VIP 开通页当前已接入真实订单流，优先调用微信支付下单接口并轮询支付状态；在配置未补齐时保留开发环境预览开通入口
- 登录页、注册页、Google 登录按钮
- 自选页与持仓页的账户工作区布局重构
- 自定义分组下拉菜单与持仓交易时间录入面板
- 自选卡片迷你走势图支持 hover 查看对应点位涨跌幅
- 持仓页现可在夜间官方净值同步完成后展示单条持仓与总仓的真实当前市值、今日盈亏、今日涨跌幅
- 首页登录态入口：未登录显示登录/注册，已登录显示头像 + 用户名账户菜单
- 登录用户可在账户菜单中切换自己的行情数据源
- 首页基金主卡片会显示基金代码，重仓股区域会显示当前实际参与估值展示的数量
- 含港股重仓的基金现在可在首页重仓股明细中展示更完整的持仓覆盖
- 搜索框支持：
  - 最近搜索 3 条
  - 本地搜索次数 Top 3 快速选择
  - 本地清空历史搜索
- 通过同源 `/api/v1/*` 调用后端，避免浏览器直接跨域请求后端
- 自选页与持仓页已接入 VIP 入口，支持真实会员状态、任务与模板化示例报告流转
- 新增公开的“我有想法！”页面，可查看全站反馈并在登录后提交新的想法
- 想法详情页新增公开“官方回复”区块，所有用户都可查看管理员写入的处理说明
- 新增公开的公告 / 更新日志页面，可查看历史更新记录
- 登录用户如有未读公告，会在进入站点后收到弹窗提醒并可标记已读
- Dark / Cyber / Classic 三套主题已做专项可读性修正，其中 Classic 针对浅色层次、页签与 VIP 页面做了增强
- 自选 / 持仓页按钮、切换标签与删除操作增加动画反馈

## 技术架构

### Backend (Go)
- Framework: Gin
- Math: shopspring/decimal
- HTTP Client: go-resty
- Auth: bcrypt password hashing + Google ID token verification + HttpOnly session cookie
- Storage: PostgreSQL + GORM
- Cache: go-cache
- Startup Config: `fundlive.yaml`

### Frontend (Next.js)
- Framework: Next.js 16 + React 19
- Styling: Tailwind CSS 4
- Data Fetching: SWR
- Charts: Recharts
- Icons/UI helpers: lucide-react, Radix Slot
- Local Preferences: browser `localStorage`

## 项目结构

```text
fund/
├── cmd/
│   ├── crawler/                 # 数据抓取 CLI
│   └── server/                  # 后端服务入口
├── internal/
│   ├── adapter/                 # 外部行情适配器（新浪）
│   ├── appconfig/               # 项目配置加载
│   ├── crawler/                 # 基金/持仓/分钟 K 线抓取
│   ├── database/                # GORM 模型、受控 migration 与 DB 初始化
│   ├── domain/                  # 领域模型与接口
│   ├── handler/                 # HTTP Handler
│   ├── middleware/              # 中间件
│   ├── repository/              # 仓储实现
│   ├── service/                 # 估值、联接基金解析、分时回填
│   └── trading/                 # A 股交易时段判断
├── web/                         # Next.js 前端
├── fundlive.example.yaml        # 项目配置模板
└── docker-compose.yml           # 本地示例 PostgreSQL 容器配置
```

## 配置文件

项目默认按以下顺序查找配置：

- `fundlive.yaml`
- `fundlive.yml`
- `config/fundlive.yaml`
- `~/.fundlive/fundlive.yaml`

配置模板见 [fundlive.example.yaml](/root/workspace/fund/fundlive.example.yaml)。

示例：

```yaml
server:
  port: 8080
  allowed_origins:
    - http://127.0.0.1:3000
    - http://localhost:3000

storage:
  mode: postgres

quote:
  default_source: sina

database:
  host: 127.0.0.1
  port: 5432
  user: postgres
  password: your-db-password
  name: my-db
  ssl_mode: disable
  timezone: Asia/Shanghai
  log_level: warn
  auto_migrate: false

auth:
  cookie_name: fundlive_session
  cookie_secure: false
  session_ttl_hours: 720
  google_client_id: your-google-client-id.apps.googleusercontent.com

payment:
  wechat_pay:
    enabled: false
    app_id: your-wechat-pay-app-id
    merchant_id: your-wechat-pay-merchant-id
    merchant_certificate_serial_no: your-merchant-cert-serial
    merchant_private_key_path: /path/to/apiclient_key.pem
    api_v3_key: your-32-byte-api-v3-key
    notify_url: https://your-domain.example.com/api/v1/vip/payments/wechat/notify
    platform_certificate_path: /path/to/wechatpay_cert.pem
    platform_public_key_path: ""
    platform_serial_no: your-wechat-platform-serial
```

说明：

- 运行时以项目根目录的本地 `fundlive.yaml` 为主；`docker-compose.yml` 仅提供本地示例数据库，不代表当前环境一定使用它
- 推荐通过 `fundlive.yaml` 启动项目，不要在命令行临时拼接环境变量
- 环境变量仍然可以覆盖配置文件
- `quote.default_source` 用于指定未登录用户和未设置偏好的用户默认使用的行情源，当前支持 `sina` / `tencent`
- 登录用户可通过前端账户菜单切换自己的行情源偏好，后端会将偏好持久化到用户账号并在后续请求中优先生效
- 如果前端需要以独立域名或端口直接访问后端，请在 `server.allowed_origins` 或环境变量 `CORS_ALLOWED_ORIGINS` 中显式配置允许的来源
- `database.log_level` 支持 `silent/error/warn/info`，建议日常运行使用 `warn`
- `database.auto_migrate` 默认建议保持 `false`；启动时会执行受控 SQL migration，并将结果记录到 `schema_migrations`
- 全新的空 PostgreSQL 库现在也可以直接依赖受控 SQL migration 完成核心表初始化，一般不再需要临时开启 `database.auto_migrate=true`
- 请不要把真实数据库密码提交到仓库；本地 `fundlive.yaml` 建议仅保留在开发机，不纳入版本控制
- 如果要启用 Google 登录，后端需要配置 `auth.google_client_id` 或环境变量 `GOOGLE_CLIENT_ID`
- 前端启用 Google 登录时，需要在 `web/.env.local` 中配置 `NEXT_PUBLIC_GOOGLE_CLIENT_ID`
- 如果要启用微信支付，需要补齐 `payment.wechat_pay` 下的商户号、应用 ID、商户私钥、商户证书序列号、API v3 Key 和回调地址
- 当前支付方式先接入微信支付 `Native` 模式，后端会返回 `code_url`，前端会轮询订单状态并在支付成功后自动刷新会员状态

## 快速开始

### 1. 准备后端环境

```bash
cd /root/workspace/fund
go version
```

要求：

- Go `1.25.8`
- 一个可连接的 PostgreSQL 实例

### 2. 配置数据库

复制模板并填写你的数据库连接信息：

```bash
cp fundlive.example.yaml fundlive.yaml
```

### 3. 初始化数据库数据

项目支持通过 crawler 抓取基金详情和持仓并落库。

先建议导入一小批，确认链路正常：

```bash
go run ./cmd/crawler --list popular --save-db --timeout 20m
```

如果要继续扩充：

```bash
# 股票型 + 混合型基金，建议先配合 --limit 分批导入
go run ./cmd/crawler --list stock --limit 200 --save-db --timeout 20m

# 全量导入，耗时长，建议在稳定网络和较长时间窗口下运行
go run ./cmd/crawler --list all --save-db --timeout 60m
```

如果持仓股票名称有乱码，可以修复：

```bash
go run ./cmd/crawler --fix-all-names
```

### 4. 启动后端

```bash
go run ./cmd/server
```

默认读取 `fundlive.yaml`。

启动后接口地址：

- `http://localhost:8080/health`
- `http://localhost:8080/api/v1/fund/search?q=005827`
- `http://localhost:8080/api/v1/auth/me`

### 5. 启动前端

```bash
cd /root/workspace/fund/web
npm install
npm run dev
```

前端地址：

- `http://localhost:3000`

说明：

- 前端默认通过同源 `/api/v1/*` 代理路由把请求转发到后端
- 如果后端不在 `127.0.0.1:8080`，请在启动前端前设置：

```bash
BACKEND_URL=http://your-backend-host:8080 npm run dev
```

- 如果要在前端显示 Google 登录按钮，请额外配置：

```bash
cd /root/workspace/fund/web
echo 'NEXT_PUBLIC_GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com' >> .env.local
```

## API 概览

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/health` | 健康检查 |
| POST | `/api/v1/auth/register` | 邮箱密码注册并建立登录态 |
| POST | `/api/v1/auth/login` | 邮箱密码登录 |
| POST | `/api/v1/auth/google` | Google 登录 / 首次自动注册 |
| GET | `/api/v1/auth/me` | 获取当前登录用户 |
| POST | `/api/v1/auth/logout` | 退出当前登录态 |
| GET | `/api/v1/user/favorites` | 获取当前用户自选基金 |
| POST | `/api/v1/user/favorites` | 新增自选基金 |
| DELETE | `/api/v1/user/favorites/:fundId` | 删除自选基金 |
| GET | `/api/v1/user/funds/:fundId/holding-overrides` | 获取当前用户对某基金的持仓修正 |
| PUT | `/api/v1/user/funds/:fundId/holding-overrides` | 替换当前用户对某基金的持仓修正 |
| GET | `/api/v1/fund/search?q=<query>` | 搜索基金 |
| GET | `/api/v1/fund/:id` | 获取基金基础信息 |
| GET | `/api/v1/fund/:id/estimate` | 获取盘中实时预估 |
| GET | `/api/v1/fund/:id/holdings` | 获取基金持仓 |
| GET | `/api/v1/fund/:id/timeseries` | 获取分时走势 |
| GET | `/api/v1/market/status` | 获取当前市场时段 |

### `timeseries` 当前行为

后端会优先返回数据库/内存中已有的分时点；如果不完整，则自动回填：

- 交易中：返回当日 `09:30` 到当前 5 分钟点
- 午休：返回当日 `09:30` 到 `11:30`
- 收盘后 / 盘前 / 周末：返回上一交易日走势
- 如果序列里有 `11:30` 但没有 `13:00`，会自动补一条 `13:00` 点，值复制 `11:30`

## 数据获取策略

### 基金详情与持仓
- 基础基金目录可以批量导入数据库
- 某只基金如果只有目录信息、没有详情/持仓，后端会在首次请求估值或分时时自动补抓

### 联接基金
- 若本基金无直接持仓，后端会尝试解析目标 ETF
- 解析顺序为：成功缓存映射 -> 详情页 `查看相关ETF` -> 详情页 `跟踪标的` 增强搜索 -> 搜索 fallback
- 解析结果写入 `fund_mappings`
- 若目标 ETF 有持仓，则按其持仓估值
- 若目标 ETF 无持仓，则直接按 ETF 本身行情估值
- 若本基金仅存在占比为 `0%` 的无效自有持仓，也会视为“无有效持仓”并继续回退到目标 ETF

### 商品 / 期货基金
- 对不适合使用股票持仓估值的基金，后端会查询 `fund_valuation_profiles`
- 当前已支持 `futures_underlying` 定价方式
- 已内置国投瑞银白银期货(LOF)A / C，底层标的为白银期货主力合约 `AG0`
- 未配置估值档案的商品/期货基金会返回明确的 `UNSUPPORTED_PRICING_MODEL`，而不是模糊的 500

### QDII / 海外基金
- 持仓解析器现已支持美股 / 海外 ticker（如 `NVDA`、`AAPL`、`GOOG`）
- 对已拿到海外持仓、但当前仍无稳定海外实时行情 provider 的基金，首页估值会返回降级结果：
  - 持仓详情可展示
  - 最新已知净值可展示
  - 盘中实时涨跌与分时图暂不伪造
- 这类基金的 `data_source` 会明确标记为 `QDII持仓详情（盘中估值暂不支持）`

### 分时走势
- 优先使用已持久化分时点
- 缺失时使用新浪 5 分钟 K 线回补
- 普通基金按重仓股权重合成
- 联接基金按目标 ETF 或目标 ETF 持仓回算

## 前端搜索框逻辑

搜索框当前有两组本地偏好数据，均保存在浏览器 `localStorage`：

- 历史搜索：最近 3 个实际选择过的基金
- 快速选择：本地累计搜索次数 Top 3

说明：

- 这是当前浏览器本地数据，不跨设备同步
- 清空历史搜索只会清空最近记录，不会清空本地搜索次数排行

## 用户系统与登录态

### 登录方式
- 邮箱密码注册 / 登录
- Google 登录（需要前后端都配置同一个 Google Web Client ID）

### 登录态行为
- 后端使用 `tb_user_session` + HttpOnly Cookie 存储会话
- 首页未登录时显示“登录 / 注册”入口
- 首页已登录时显示“头像 + 用户名”账户菜单，并支持退出登录

### Google 登录配置
后端 `fundlive.yaml`：

```yaml
auth:
  google_client_id: your-google-client-id.apps.googleusercontent.com
```

前端 `web/.env.local`：

```env
NEXT_PUBLIC_GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
```

## 当前限制

- A 股交易日历当前已内置 `2024`-`2026` 上交所休市日数据；超出覆盖年份时会回退到“工作日”规则
- 全量基金详情与持仓抓取耗时较长，不建议一次性在弱网络环境下跑完
- 商品/期货基金当前仅支持已配置 `fund_valuation_profiles` 的品种
- 自选分组与持仓记录前端页面已补齐；`holding overrides`（持仓修正）的前端管理入口仍未完整接入
- 本地搜索偏好仍然是浏览器本地数据，未并入用户账号同步
- Google 登录依赖合法的 Google Web Client ID 与浏览器可访问 Google Identity Services

## 常见问题

### 1. 前端报 Turbopack chunk 加载错误

开发环境下如果出现旧 chunk 缓存问题：

```bash
cd /root/workspace/fund/web
rm -rf .next
npm run dev
```

然后浏览器强制刷新一次。

### 2. 选中新基金后估值报错

通常是数据库里只有基金目录、没有详情和持仓。当前后端会先进入后台预热，并在接口返回 `FUND_DATA_WARMING` 时由前端自动重试；如果长时间仍失败，再检查后端日志和外部数据源可达性。

### 3. 商品 / 期货基金估值返回 `UNSUPPORTED_PRICING_MODEL`

说明该基金不适合走股票持仓估值，且当前还没有配置 `fund_valuation_profiles`。处理方式：

- 为该基金新增估值档案
- 指定 `pricing_method`、`quote_source`、`underlying_symbol` 等字段
- 再重新调用 `/estimate`

### 4. 分时图不完整

当前后端已支持自动分时回填。如果仍出现断档，优先确认：

- 当前基金是否有可用持仓或可解析的目标 ETF
- 后端是否是最新代码版本
- 浏览器是否还在使用旧前端 dev 缓存

## 免责声明

本工具仅供学习和参考使用，估值数据基于公开持仓和实时行情推算，可能与实际净值存在偏差。**不构成任何投资建议**，投资有风险，入市需谨慎。

## License

MIT License
