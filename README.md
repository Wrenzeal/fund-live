# FundLive - 实时基金估值系统

[![Go](https://img.shields.io/badge/Go-1.25.8-00ADD8?logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-16+-000000?logo=next.js)](https://nextjs.org)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind-4+-06B6D4?logo=tailwindcss)](https://tailwindcss.com)

> 通过基金前十大重仓股和联接基金目标 ETF 的实时行情，计算基金盘中预估涨跌幅，并补全分时走势图。

当前文档对应版本：`2026.4.4`

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

当前项目已经移除了 AI 运行时依赖。联接基金解析通过东方财富搜索完成，解析结果持久化到 `fund_mappings` 表。
当前版本支持邮箱密码登录、Google 登录、自选基金、分组管理、用户持仓修正，以及带交易时间的持仓录入、夜间真实净值同步与确认净值日推导。

## 当前能力

### 后端
- Gin API 服务
- PostgreSQL 持久化存储
- 以 `fundlive.yaml` 为主的启动配置，支持数据库日志级别与自动迁移开关
- 用户体系：邮箱密码登录、Google 登录、服务端 Session、登出、当前用户信息
- 用户偏好：自选基金、用户持仓修正、带交易时间的用户持仓记录
- `fund_history` 存储基金官方日净值、累计净值与日涨跌幅，夜间定时同步用户持仓涉及基金的真实值
- 基金目录、基金详情、持仓、分时点管理
- 支持 `sina` / `tencent` 双行情源，并可按用户绑定默认行情源
- 联接基金目标 ETF 自动解析与映射缓存
- 联接基金失败映射带冷却重试窗口，避免高频重复外呼
- 商品/期货基金通过 `fund_valuation_profiles` 走自定义标的估值
- 分时数据缺失时自动回填
- 使用新浪 5 分钟 K 线回补盘中走势
- 午休缺失 `13:00` 点时自动补点
- 基金详情 / 持仓 / 估值 / 联接基金解析共用只读瞬时补全缓存
- 冷基金数据改为后台预热，预热状态会通过 `meta.cache_status` / `FUND_DATA_WARMING` 返回前端
- 用户持仓按交易日和北京时间 `15:00` 截止自动计算确认净值日
- 受控 SQL migration：启动时会检查 `schema_migrations`、搜索索引与关键唯一约束

### 前端
- Next.js 16 App Router
- SWR 数据获取
- Recharts 分时图表
- 登录页、注册页、Google 登录按钮
- 自选页与持仓页的账户工作区布局重构
- 自定义分组下拉菜单与持仓交易时间录入面板
- 自选卡片迷你走势图支持 hover 查看对应点位涨跌幅
- 持仓页在夜间官方净值同步完成后会优先展示真实日涨跌额
- 首页登录态入口：未登录显示登录/注册，已登录显示头像 + 用户名账户菜单
- 登录用户可在账户菜单中切换自己的行情数据源
- 搜索框支持：
  - 最近搜索 3 条
  - 本地搜索次数 Top 3 快速选择
  - 本地清空历史搜索
- 通过同源 `/api/v1/*` 调用后端，避免浏览器直接跨域请求后端
- Dark / Cyber 主题下的搜索框、搜索弹层与模式切换面板针对可读性做了定向增强
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
├── fundlive.yaml                # 项目实际配置文件
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
```

说明：

- 运行时以项目根目录的 `fundlive.yaml` 为主；`docker-compose.yml` 仅提供本地示例数据库，不代表当前环境一定使用它
- 推荐通过 `fundlive.yaml` 启动项目，不要在命令行临时拼接环境变量
- 环境变量仍然可以覆盖配置文件
- `quote.default_source` 用于指定未登录用户和未设置偏好的用户默认使用的行情源，当前支持 `sina` / `tencent`
- 登录用户可通过前端账户菜单切换自己的行情源偏好，后端会将偏好持久化到用户账号并在后续请求中优先生效
- 如果前端需要以独立域名或端口直接访问后端，请在 `server.allowed_origins` 或环境变量 `CORS_ALLOWED_ORIGINS` 中显式配置允许的来源
- `database.log_level` 支持 `silent/error/warn/info`，建议日常运行使用 `warn`
- `database.auto_migrate` 默认建议保持 `false`；启动时会执行受控 SQL migration，并将结果记录到 `schema_migrations`
- 如果是一个全新的空 PostgreSQL 库，首次建表可临时打开 `database.auto_migrate=true` 启动一次；完成后应恢复为 `false`
- 请不要把真实数据库密码提交到仓库
- 如果要启用 Google 登录，后端需要配置 `auth.google_client_id` 或环境变量 `GOOGLE_CLIENT_ID`
- 前端启用 Google 登录时，需要在 `web/.env.local` 中配置 `NEXT_PUBLIC_GOOGLE_CLIENT_ID`

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

- 前端默认通过 Next.js rewrite 把 `/api/v1/*` 转发到后端
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
- 解析结果写入 `fund_mappings`
- 若目标 ETF 有持仓，则按其持仓估值
- 若目标 ETF 无持仓，则直接按 ETF 本身行情估值

### 商品 / 期货基金
- 对不适合使用股票持仓估值的基金，后端会查询 `fund_valuation_profiles`
- 当前已支持 `futures_underlying` 定价方式
- 已内置国投瑞银白银期货(LOF)A / C，底层标的为白银期货主力合约 `AG0`
- 未配置估值档案的商品/期货基金会返回明确的 `UNSUPPORTED_PRICING_MODEL`，而不是模糊的 500

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
