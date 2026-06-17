# FundLive Web

FundLive 前端应用，基于 Next.js App Router、React、TypeScript 和 Tailwind CSS 构建。完整项目说明请看根目录 [`../README.md`](../README.md)。

FundLive frontend app built with Next.js App Router, React, TypeScript, and Tailwind CSS. See the root [`../README.md`](../README.md) for the full project overview.

## 本地开发 / Local Development

```bash
npm install
npm run dev
```

默认访问 / Open: `http://localhost:3000`

## 常用命令 / Commands

```bash
npm run lint
npm run build
npm run start
```

## 环境变量 / Environment Variables

```env
NEXT_PUBLIC_GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
BACKEND_URL=http://127.0.0.1:8080
```

Vercel 生产环境应设置：

```env
BACKEND_URL=https://api.fund.wrenzeal.top
```

- `NEXT_PUBLIC_GOOGLE_CLIENT_ID`：浏览器侧 Google 登录 Client ID。
- `BACKEND_URL`：Next.js API 代理访问 Go 后端时使用；Vercel 生产环境必须指向公网 API 子域 `https://api.fund.wrenzeal.top`，不要填前端域名 `https://fund.wrenzeal.top`。
- 通常不要设置 `NEXT_PUBLIC_API_URL`；浏览器默认走同源 `/api/v1/*`，避免跨域 Cookie 问题。

- `NEXT_PUBLIC_GOOGLE_CLIENT_ID`: browser-side Google Sign-In Client ID.
- `BACKEND_URL`: Go backend base URL used by the Next.js API proxy; Vercel production should use the public API subdomain, not the frontend domain.
