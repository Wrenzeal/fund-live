<div align="center">
  <img src="web/public/favicon.svg" width="72" height="72" alt="FundLive logo" />

  <h1>FundLive</h1>

  <p>
    <strong>Real-time fund estimation, portfolio tracking, and quantitative insight dashboard</strong>
  </p>

  <p>
    <a href="https://go.dev/"><img alt="Go 1.26.3" src="https://img.shields.io/badge/Go-1.26.3-00ADD8?logo=go" /></a>
    <a href="https://nextjs.org/"><img alt="Next.js 16.2.7" src="https://img.shields.io/badge/Next.js-16.2.7-000000?logo=next.js" /></a>
    <a href="https://react.dev/"><img alt="React 19.2" src="https://img.shields.io/badge/React-19.2-61DAFB?logo=react" /></a>
    <a href="https://tailwindcss.com/"><img alt="Tailwind CSS 4" src="https://img.shields.io/badge/Tailwind-4-06B6D4?logo=tailwindcss" /></a>
  </p>
  <p><a href="README.md">中文</a></p>
</div>

---

## Overview

FundLive estimates intraday fund movement from public holdings, target ETFs, real-time quotes, and local snapshots. It helps users track watchlists, personal holdings, sector/theme exposure, and quantitative signals.

> This project is for data observation and research only. It is not investment advice.

## Key Features

- **Real-time estimation**: Estimates intraday fund movement from top holdings, feeder ETF targets, and QDII overseas holdings.
- **Fund search and profile**: Supports catalog sync, availability status, detail/holding warm-up, and intraday charts.
- **Watchlist and portfolio**: Supports grouped watchlists, holding records, official NAV view, and intraday estimate view.
- **Quant dashboard**: Provides scores, recommendation distribution, supporting/opposing evidence, risks, events, and confidence breakdowns.
- **Sector and theme exposure**: Aggregates holdings by sector/theme and supports admin classification overrides.
- **Authentication**: Supports email/password login, Google Sign-In, HttpOnly cookie sessions, and rate limiting.

## Stack

| Layer | Technology |
| --- | --- |
| Backend | Go 1.26.3, Gin, GORM |
| Database | PostgreSQL, with in-memory repositories for development paths |
| Frontend | Next.js 16 App Router, React 19, TypeScript |
| UI | Tailwind CSS 4, Radix Slot, lucide-react, Recharts |
| State/Data | SWR, Zustand |
| Deployment | Docker Compose / systemd + PM2 + Nginx |

## Project Layout

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

## Local Development

### 1. Prepare config

```bash
cp fundlive.example.yaml fundlive.yaml
```

Edit PostgreSQL and auth settings as needed. Environment variables override YAML values.

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

### 2. Start backend

```bash
go run ./cmd/server
```

The default port comes from `server.port` in `fundlive.yaml`; the example uses `8080`.

### 3. Start frontend

```bash
cd web
npm install
npm run dev
```

Open `http://localhost:3000`.

## Common Commands

```bash
# Backend tests
go test ./...

# Frontend lint and production build
cd web
npm run lint
npm run build

# Fast catalog-only fund sync
go run ./cmd/crawler --list all --save-db --catalog-only

# Backfill the latest 30 official NAV days for tracked funds
go run ./cmd/crawler --history-only --tracked-only --history-days 30 --save-db
```

## Deployment Notes

- Production should run behind HTTPS with `auth.cookie_secure=true`.
- A typical production path is: Nginx -> Next.js frontend -> Go API.
- If the frontend uses the Next.js API proxy, set `BACKEND_URL` in the PM2/runtime environment.
- Deployment scripts live in `scripts/deploy-backend.sh` and `scripts/deploy-frontend.sh`.

## Current Limits

- Estimates depend on public holdings, quote sources, and cached snapshots, so they can differ from official NAV.
- The A-share trading calendar includes major 2024-2026 holidays; dates outside that range fall back to weekday rules.
- Commodity/futures funds require `fund_valuation_profiles` before specialized pricing models can be used.
- Google Sign-In requires a valid OAuth Client ID and authorized JavaScript origins in Google Cloud Console.
- Auth rate limiting is currently process-local; multi-instance deployment should use shared rate limiting or WAF rules.

## More Documents

- [`CHANGELOG.md`](CHANGELOG.md): release and feature history
- [`DESIGN.md`](DESIGN.md): product and design decisions
- [`todo_list.md`](todo_list.md): current scope and upcoming work
- [`docs/overseas-data-source-selection.md`](docs/overseas-data-source-selection.md): overseas quote data-source evaluation

---

## License

MIT License
