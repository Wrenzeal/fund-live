# Vercel 前端部署说明

当前前端在浏览器里默认请求同源 `/api/v1/*`。这些请求会先进入 Next.js Route Handler，再由 `web/src/lib/backend-proxy.ts` 代理到 Go 后端。

## 必填环境变量

在 Vercel Project Settings → Environment Variables 中配置：

```bash
BACKEND_URL=https://api.fund.wrenzeal.top
```

要求：

- `BACKEND_URL` 必须是 Vercel Serverless Function 可以访问的公网 HTTPS 地址；当前项目应填写 `https://api.fund.wrenzeal.top`，不要填写前端域名 `https://fund.wrenzeal.top`。
- 不要填 `127.0.0.1`、`localhost`、内网 IP 或当前服务器本机端口；在 Vercel 上这些地址指向 Vercel 自己的运行环境，不是你的后端服务器。
- 浏览器侧通常不要设置 `NEXT_PUBLIC_API_URL`。前端代码默认请求同源 `/api/v1/*`，再由 Vercel Route Handler 读取 `BACKEND_URL` 转发，这样登录 Cookie 留在前端域名下。
- 如果历史上把 `NEXT_PUBLIC_API_URL` 设置成 `https://fund.wrenzeal.top`，请删除它；代码现在会把这个误配视为未设置，避免请求绕回前端域名。
- `api.fund.wrenzeal.top` 必须在 DNS / Nginx / HTTPS 证书层面真实可访问，并反代到服务器上的 Go 后端 `127.0.0.1:13896`。仅在 Vercel 添加环境变量不能替代后端子域名的 DNS 和反代配置。

推荐形态：

```bash
# Vercel 前端域名
https://fund.wrenzeal.top

# 服务器/Nginx 后端域名
https://api.fund.wrenzeal.top

# Vercel 环境变量
BACKEND_URL=https://api.fund.wrenzeal.top
```

## 后端子域名准备

Vercel 只能把请求转发到 `BACKEND_URL`，不能自动创建或修复 `api.fund.wrenzeal.top`。后端服务器还需要满足：

1. DNS：`api.fund.wrenzeal.top` 解析到运行 Go 后端/Nginx 的服务器。
2. Nginx：`server_name api.fund.wrenzeal.top`，并把 `/` 反代到 `http://127.0.0.1:13896`。
3. TLS：为 `api.fund.wrenzeal.top` 签发有效 HTTPS 证书。
4. 后端 CORS：如果未来改成浏览器直接请求 API 子域，需要把 `https://fund.wrenzeal.top` 加入 `server.allowed_origins` 或 `CORS_ALLOWED_ORIGINS`；当前同源代理形态不需要浏览器跨域访问。

## 验证命令

部署后检查：

```bash
# 先确认后端子域名本身可达
curl -i https://api.fund.wrenzeal.top/health

# 再确认 Vercel 前端同源代理可达
curl -i https://fund.wrenzeal.top/health
curl -i https://fund.wrenzeal.top/api/v1/auth/config
curl -i https://fund.wrenzeal.top/api/v1/market/status
```

预期这些请求返回 Go 后端 JSON，而不是 Vercel 的空 500/502。

如果返回：

```json
{"success":false,"error":{"code":"BACKEND_UNREACHABLE"}}
```

说明 Next.js 代理已运行，但 Vercel 仍无法访问 `BACKEND_URL`，应检查 `api.fund.wrenzeal.top` 的 DNS、HTTPS 证书、Nginx 反代、防火墙，以及 Vercel 环境变量是否已重新部署生效。

## Google 登录

Google 登录按钮会请求 `/api/v1/auth/config`。如果该接口 500/502，通常不是 Google 配置本身坏了，而是 Vercel 前端没有连上 Go 后端。

## 图表 width(-1)/height(-1) 警告

当 dashboard/timeseries 接口失败时，图表容器可能在错误态或空数据态下短暂测量到非法尺寸，Recharts 会输出 `width(-1) and height(-1)`。优先修复 API 代理；若 API 正常后仍出现，再单独处理图表容器最小高度。
