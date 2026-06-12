# Vercel 前端部署说明

当前前端在浏览器里默认请求同源 `/api/v1/*`。这些请求会先进入 Next.js Route Handler，再由 `web/src/lib/backend-proxy.ts` 代理到 Go 后端。

## 必填环境变量

在 Vercel Project Settings → Environment Variables 中配置：

```bash
BACKEND_URL=https://<你的后端公网域名>
```

要求：

- `BACKEND_URL` 必须是 Vercel Serverless Function 可以访问的公网 HTTPS 地址。
- 不要填 `127.0.0.1`、`localhost`、内网 IP 或当前服务器本机端口；在 Vercel 上这些地址指向 Vercel 自己的运行环境，不是你的后端服务器。
- 如果继续使用 `fund.wrenzeal.top` 作为 Vercel 前端域名，就不能再同时依赖这个域名的 `/api` 指向后端。请给 Go 后端准备一个独立域名，例如 `api.fund.wrenzeal.top`，并让它反代到服务器上的 `127.0.0.1:13896`。

推荐形态：

```bash
# Vercel 前端域名
https://fund.wrenzeal.top

# 服务器/Nginx 后端域名
https://api.fund.wrenzeal.top

# Vercel 环境变量
BACKEND_URL=https://api.fund.wrenzeal.top
```

## 验证命令

部署后检查：

```bash
curl -i https://fund.wrenzeal.top/health
curl -i https://fund.wrenzeal.top/api/v1/auth/config
curl -i https://fund.wrenzeal.top/api/v1/market/status
```

预期这些请求返回 Go 后端 JSON，而不是 Vercel 的空 500。

如果返回：

```json
{"success":false,"error":{"code":"BACKEND_UNREACHABLE"}}
```

说明 Next.js 代理已运行，但 Vercel 仍无法访问 `BACKEND_URL`，应检查后端公网域名、HTTPS 证书、Nginx 防火墙和 Vercel 环境变量是否已重新部署生效。

## Google 登录

Google 登录按钮会请求 `/api/v1/auth/config`。如果该接口 500/502，通常不是 Google 配置本身坏了，而是 Vercel 前端没有连上 Go 后端。

## 图表 width(-1)/height(-1) 警告

当 dashboard/timeseries 接口失败时，图表容器可能在错误态或空数据态下短暂测量到非法尺寸，Recharts 会输出 `width(-1) and height(-1)`。优先修复 API 代理；若 API 正常后仍出现，再单独处理图表容器最小高度。
