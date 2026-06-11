<div align="center">
  <img src="web/public/favicon.svg" width="72" height="72" alt="FundLive logo" />

  <h1>涨了多少 · FundLive</h1>

  <p>
    <strong>实时基金估值、持仓管理与量化观察系统</strong><br />
    <strong>Real-time fund estimation, portfolio tracking, and quantitative insight dashboard</strong>
  </p>

  <p>
    <a href="https://go.dev/"><img alt="Go 1.26.3" src="https://img.shields.io/badge/Go-1.26.3-00ADD8?logo=go" /></a>
    <a href="https://nextjs.org/"><img alt="Next.js 16.2.7" src="https://img.shields.io/badge/Next.js-16.2.7-000000?logo=next.js" /></a>
    <a href="https://react.dev/"><img alt="React 19.2" src="https://img.shields.io/badge/React-19.2-61DAFB?logo=react" /></a>
    <a href="https://tailwindcss.com/"><img alt="Tailwind CSS 4" src="https://img.shields.io/badge/Tailwind-4-06B6D4?logo=tailwindcss" /></a>
  </p>
</div>

---

## 中文

### 项目简介

FundLive（涨了多少）通过基金公开持仓、目标 ETF、实时行情与本地快照，估算基金盘中涨跌，帮助用户观察自选基金、个人持仓、行业/主题暴露和量化信号。

> 本项目只做数据整理与观察辅助，不构成任何投资建议。

### 核心能力

- **实时估值**：基于前十大重仓股、联接基金目标 ETF、QDII 海外持仓等链路估算盘中涨跌。
- **基金搜索与详情**：支持基金目录同步、可用状态治理、详情/持仓预热与分时图展示。
- **自选与持仓**：支持自选分组、持仓记录、官方口径与盘中预估口径切换。
- **量化看板**：输出综合评分、建议分布、主/反证据、风险提示、事件链和数据可信度拆解。
- **分类与主题暴露**：按行业/主题聚合持仓，可叠加管理员人工分类修正。
- **账号与认证**：支持邮箱密码登录、Google 登录、HttpOnly Cookie 会话和认证失败限流。
- **VIP / 报告雏形**：保留会员、任务、报告与微信支付配置边界，便于后续生产化。

### 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go 1.26.3, Gin, GORM |
| 数据库 | PostgreSQL（开发也保留内存仓储能力） |
| 前端 | Next.js 16 App Router, React 19, TypeScript |
| UI | Tailwind CSS 4, Radix Slot, lucide-react, Recharts |
| 状态/请求 | SWR, Zustand |
| 部署 | Docker Compose / systemd + PM2 + Nginx |

### 目录结构

```text
cmd/                         后端入口与运维工具
  server/                    API 服务
  crawler/                   基金目录/详情/持仓同步
  audit-fund-analysis/       量化分析审计命令
internal/                    后端领域、服务、仓储、适配器与 handler
web/                         Next.js 前端
  src/app/                   App Router 页面与 API 代理
  src/components/            共享 UI 与业务组件
  public/favicon.svg         项目图标 / 浏览器标签图标
scripts/                     部署脚本
docs/                        设计与数据源文档
fundlive.example.yaml        后端配置模板
```

### 本地启动

#### 1. 准备配置

```bash
cp fundlive.example.yaml fundlive.yaml
```

按需修改 PostgreSQL、认证和支付配置。环境变量会覆盖 YAML 中的同名配置。

Google 登录需要后端和前端使用同一个 Web Client ID：

```yaml
# fundlive.yaml
auth:
  google_client_id: your-google-client-id.apps.googleusercontent.com
```

```env
# web/.env.local
NEXT_PUBLIC_GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
```

#### 2. 启动后端

```bash
go run ./cmd/server
```

默认端口来自 `fundlive.yaml` 的 `server.port`，本地模板为 `8080`。

#### 3. 启动前端

```bash
cd web
npm install
npm run dev
```

打开 `http://localhost:3000`。

### 常用命令

```bash
# 后端测试
go test ./...

# 前端检查与构建
cd web
npm run lint
npm run build

# 快速刷新基金目录（只更新目录元数据）
go run ./cmd/crawler --list all --save-db --catalog-only
```

### 部署提示

- 生产环境建议使用 HTTPS，并设置 `auth.cookie_secure=true`。
- 当前生产链路通常为：Nginx -> Next.js 前端 -> Go API。
- 若前端通过 Next.js API 代理访问后端，PM2 / 运行环境需要设置 `BACKEND_URL`。
- 部署脚本位于 `scripts/deploy-backend.sh` 与 `scripts/deploy-frontend.sh`。

### 当前边界

- 实时估值依赖公开持仓、行情源和缓存快照，可能与基金真实净值存在偏差。
- A 股交易日历已内置 2024-2026 年主要休市日，超出范围时回退到工作日规则。
- 商品/期货基金需要配置 `fund_valuation_profiles` 才能使用对应估值模型。
- Google 登录依赖合法 OAuth Client ID，且部署域名必须加入 Google Cloud Console 的授权 JavaScript 来源。
- 当前认证限流为单实例进程内实现；多实例部署应接入共享限流或 WAF。

### 更多文档

- [`CHANGELOG.md`](CHANGELOG.md)：版本与功能变更记录
- [`DESIGN.md`](DESIGN.md)：产品和设计决策
- [`todo_list.md`](todo_list.md)：当前任务边界与后续计划
- [`docs/overseas-data-source-selection.md`](docs/overseas-data-source-selection.md)：海外行情数据源评估

---

## English

### Overview

FundLive estimates intraday fund movement from public holdings, target ETFs, real-time quotes, and local snapshots. It helps users track watchlists, personal holdings, sector/theme exposure, and quantitative signals.

> This project is for data observation and research only. It is not investment advice.

### Key Features

- **Real-time estimation**: Estimates intraday fund movement from top holdings, feeder ETF targets, and QDII overseas holdings.
- **Fund search and profile**: Supports catalog sync, availability status, detail/holding warm-up, and intraday charts.
- **Watchlist and portfolio**: Supports grouped watchlists, holding records, official NAV view, and intraday estimate view.
- **Quant dashboard**: Provides scores, recommendation distribution, supporting/opposing evidence, risks, events, and confidence breakdowns.
- **Sector and theme exposure**: Aggregates holdings by sector/theme and supports admin classification overrides.
- **Authentication**: Supports email/password login, Google Sign-In, HttpOnly cookie sessions, and rate limiting.
- **VIP/report foundation**: Keeps membership, task, report, and WeChat Pay integration boundaries for future production use.

### Stack

| Layer | Technology |
| --- | --- |
| Backend | Go 1.26.3, Gin, GORM |
| Database | PostgreSQL, with in-memory repositories for development paths |
| Frontend | Next.js 16 App Router, React 19, TypeScript |
| UI | Tailwind CSS 4, Radix Slot, lucide-react, Recharts |
| State/Data | SWR, Zustand |
| Deployment | Docker Compose / systemd + PM2 + Nginx |

### Project Layout

```text
cmd/                         Backend entrypoints and operational tools
  server/                    API server
  crawler/                   Fund catalog/detail/holding sync
  audit-fund-analysis/       Quant-analysis audit command
internal/                    Backend domain, services, repositories, adapters, handlers
web/                         Next.js frontend
  src/app/                   App Router pages and API proxy
  src/components/            Shared UI and business components
  public/favicon.svg         Project icon / browser favicon
scripts/                     Deployment scripts
docs/                        Design and data-source documents
fundlive.example.yaml        Backend config template
```

### Local Development

#### 1. Prepare config

```bash
cp fundlive.example.yaml fundlive.yaml
```

Edit PostgreSQL, auth, and payment settings as needed. Environment variables override YAML values.

Google Sign-In requires the same Web Client ID on both backend and frontend:

```yaml
# fundlive.yaml
auth:
  google_client_id: your-google-client-id.apps.googleusercontent.com
```

```env
# web/.env.local
NEXT_PUBLIC_GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
```

#### 2. Start backend

```bash
go run ./cmd/server
```

The default port comes from `server.port` in `fundlive.yaml`; the example uses `8080`.

#### 3. Start frontend

```bash
cd web
npm install
npm run dev
```

Open `http://localhost:3000`.

### Common Commands

```bash
# Backend tests
go test ./...

# Frontend lint and production build
cd web
npm run lint
npm run build

# Fast catalog-only fund sync
go run ./cmd/crawler --list all --save-db --catalog-only
```

### Deployment Notes

- Production should run behind HTTPS with `auth.cookie_secure=true`.
- A typical production path is: Nginx -> Next.js frontend -> Go API.
- If the frontend uses the Next.js API proxy, set `BACKEND_URL` in the PM2/runtime environment.
- Deployment scripts live in `scripts/deploy-backend.sh` and `scripts/deploy-frontend.sh`.

### Current Limits

- Estimates depend on public holdings, quote sources, and cached snapshots, so they can differ from official NAV.
- The A-share trading calendar includes major 2024-2026 holidays; dates outside that range fall back to weekday rules.
- Commodity/futures funds require `fund_valuation_profiles` before specialized pricing models can be used.
- Google Sign-In requires a valid OAuth Client ID and authorized JavaScript origins in Google Cloud Console.
- Auth rate limiting is currently process-local; multi-instance deployment should use shared rate limiting or WAF rules.

### More Documents

- [`CHANGELOG.md`](CHANGELOG.md): release and feature history
- [`DESIGN.md`](DESIGN.md): product and design decisions
- [`todo_list.md`](todo_list.md): current scope and upcoming work
- [`docs/overseas-data-source-selection.md`](docs/overseas-data-source-selection.md): overseas quote data-source evaluation

---

## License

MIT License
