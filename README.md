<div align="center">
  <img src="web/public/favicon.svg" width="72" height="72" alt="FundLive logo" />

  <h1>涨了多少 · FundLive</h1>

  <p>
    <strong>实时基金估值、持仓管理与公告观察系统</strong>
  </p>

  <p>
    <a href="https://go.dev/"><img alt="Go 1.26.3" src="https://img.shields.io/badge/Go-1.26.3-00ADD8?logo=go" /></a>
    <a href="https://nextjs.org/"><img alt="Next.js 16.2.7" src="https://img.shields.io/badge/Next.js-16.2.7-000000?logo=next.js" /></a>
    <a href="https://react.dev/"><img alt="React 19.2" src="https://img.shields.io/badge/React-19.2-61DAFB?logo=react" /></a>
    <a href="https://tailwindcss.com/"><img alt="Tailwind CSS 4" src="https://img.shields.io/badge/Tailwind-4-06B6D4?logo=tailwindcss" /></a>
  </p>
  <p><a href="README.en.md">English</a></p>
</div>

---

## 项目简介

FundLive（涨了多少）通过基金公开持仓、目标 ETF、实时行情与本地快照，估算基金盘中涨跌，帮助用户观察自选基金、个人持仓、行业/主题暴露和近期公告。

> 本项目只做数据整理与观察辅助，不构成任何投资建议。

## 核心能力

- **实时估值**：基于前十大重仓股、联接基金目标 ETF、QDII 海外持仓等链路估算盘中涨跌。
- **基金搜索与详情**：支持基金目录同步、可用状态治理、详情/持仓预热与分时图展示。
- **自选与持仓**：支持自选分组、持仓记录、官方口径与盘中预估口径切换。
- **基金观察**：查看基金及已披露重仓股的近期公告、原文来源、持仓披露日期和覆盖范围；不输出综合评分、买卖比例或投资排名。
- **市场研究**：大盘、通信/CPO、半导体与 AI 行情，走势对比、技术状态、板块资金、可保存的公式曲线与组合事件验证；无需登录或数据库。
- **分类与主题暴露**：按行业/主题聚合持仓，可叠加管理员人工分类修正。
- **账号与认证**：支持邮箱验证码、邮箱密码、Google 登录、HttpOnly Cookie 会话和认证限流。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go 1.26.3, Gin, GORM |
| 数据库 | PostgreSQL（开发也保留内存仓储能力） |
| 缓存 | DragonFly（Redis 协议，验证码、冷却与共享限流） |
| 前端 | Next.js 16 App Router, React 19, TypeScript |
| UI | Tailwind CSS 4, Radix Slot, lucide-react, Recharts |
| 状态/请求 | SWR, Zustand |
| 部署 | Docker Compose / systemd + PM2 + Nginx |

## 目录结构

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

## 本地启动

### 1. 准备配置

```bash
cp fundlive.example.yaml fundlive.yaml
```

按需修改 PostgreSQL 和认证配置。环境变量会覆盖 YAML 中的同名配置。

本地验证码登录需要 DragonFly：

```bash
docker compose up -d cache
```

开发环境设置 `AUTH_EMAIL_CODE_ENABLED=true`、`AUTH_EMAIL_DRIVER=dev`、`AUTH_EMAIL_CODE_SECRET` 和 `REDIS_URL=redis://127.0.0.1:16380/0` 后，发送接口会返回仅开发可见的 `dev_code`。生产 Resend 配置见 [`docs/email-code-login.md`](docs/email-code-login.md)。

## Lean 量化验证

Lean 作为隔离的回测 Worker 运行历史代理策略实验，不参与正式基金观察页面，也不证明旧 V4 评分的预测能力。固定 A-D 实验、ETF 五年日线、历史代理信号、Dragonfly 队列与部署步骤见 [`docs/lean-quant-validation.md`](docs/lean-quant-validation.md)。旧分析 API 和实验档案保留兼容，生产前端不再请求评分或显示投资排名。TradingAgents-CN 等外部研究器仅预留 [`event-intel.v1`](docs/event-intel-shadow-contract.md) shadow 契约。

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

### 2. 启动后端

```bash
go run ./cmd/server
```

默认端口来自 `fundlive.yaml` 的 `server.port`，本地模板为 `8080`。

### 3. 启动前端

```bash
cd web
npm install
npm run dev
```

打开 `http://localhost:3000`。

Windows 本机市场研究可运行 `powershell -ExecutionPolicy Bypass -File scripts/start-local.ps1`，打开 `http://127.0.0.1:3100/research`。启动方式、数据来源和历史统计口径见 [`docs/market-research.md`](docs/market-research.md)。

## 常用命令

```bash
# 后端测试
go test ./...

# 前端检查与构建
cd web
npm run lint
npm run build

# 快速刷新基金目录（只更新目录元数据）
go run ./cmd/crawler --list all --save-db --catalog-only

# 手动补采重点基金近 30 日官方净值历史（持仓/收藏/自选）
go run ./cmd/crawler --history-only --tracked-only --history-days 30 --save-db

# 幂等创建并运行固定 A-D 风险收益实验
go run ./cmd/run-quant-experiment --preset risk-v1
```

## 部署提示

- 生产环境建议使用 HTTPS，并设置 `auth.cookie_secure=true`。
- 邮箱验证码生产环境必须使用 SMTP 驱动，并将 DragonFly、验证码密钥和 Resend 凭据放入 `/etc/fund-live/fundlive.env`。
- 当前生产链路通常为：Nginx -> Next.js 前端 -> Go API。
- 若前端通过 Next.js API 代理访问后端，PM2 / 运行环境需要设置 `BACKEND_URL`。
- 部署脚本位于 `scripts/deploy-backend.sh`、`scripts/deploy-frontend.sh` 与 `scripts/deploy-lean-worker.sh`。

## 当前边界

- 实时估值依赖公开持仓、行情源和缓存快照，可能与基金真实净值存在偏差。
- A 股交易日历已内置 2024-2026 年主要休市日，超出范围时回退到工作日规则。
- 商品/期货基金需要配置 `fund_valuation_profiles` 才能使用对应估值模型。
- Google 登录依赖合法 OAuth Client ID，且部署域名必须加入 Google Cloud Console 的授权 JavaScript 来源。
- 密码、注册和 Google 失败限流仍为进程内实现；邮箱验证码发送冷却与邮箱/IP 限流已由 DragonFly 共享。

## 更多文档

- [`CHANGELOG.md`](CHANGELOG.md)：版本与功能变更记录
- [`DESIGN.md`](DESIGN.md)：产品和设计决策
- [`docs/seo-geo.md`](docs/seo-geo.md)：搜索索引边界、站点验证、公开方法页及 AI 引用建设
- [`todo_list.md`](todo_list.md)：当前任务边界与后续计划
- [`docs/overseas-data-source-selection.md`](docs/overseas-data-source-selection.md)：海外行情数据源评估
- [`docs/email-code-login.md`](docs/email-code-login.md)：邮箱验证码、DragonFly 与 Resend 部署说明
- [`docs/lean-quant-validation.md`](docs/lean-quant-validation.md)：Lean Worker、A-D 实验、结果口径与生产运行
- [`docs/easyk-workspace.md`](docs/easyk-workspace.md)：公式曲线、组合事件、本地方案和板块资金研究
- [`docs/event-intel-shadow-contract.md`](docs/event-intel-shadow-contract.md)：外部事件研究器 shadow 接入边界

## 许可证

MIT License
