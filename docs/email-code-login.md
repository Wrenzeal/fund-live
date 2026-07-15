# FundLive 邮箱验证码登录

FundLive 支持邮箱验证码、邮箱密码和 Google 三种登录入口。验证码登录首次验证会自动创建账户，已有同邮箱账户会继续使用原用户 ID、持仓和自选数据。

## 本地运行

启动 PostgreSQL 与 FundLive 独立 DragonFly：

```bash
docker compose up -d db cache
```

DragonFly 默认监听 `127.0.0.1:16380`。开发环境可在后端环境中设置：

```env
APP_ENV=development
AUTH_EMAIL_CODE_ENABLED=true
AUTH_EMAIL_DRIVER=dev
AUTH_EMAIL_CODE_SECRET=dev-only-fundlive-email-code-secret
REDIS_URL=redis://127.0.0.1:16380/0
FUNDLIVE_REDIS_KEY_PREFIX=fundlive
```

`dev` 驱动不会发送邮件，`POST /api/v1/auth/email/start` 会返回 `dev_code`。验证码不会写入后端日志。

## 生产配置

生产环境使用 Resend SMTP，并将以下内容保存到 `/etc/fund-live/fundlive.env`。该文件不得提交到 Git：

```env
APP_ENV=production
AUTH_EMAIL_CODE_ENABLED=true
AUTH_EMAIL_DRIVER=smtp
AUTH_EMAIL_CODE_SECRET=<至少 32 字节随机密钥>
REDIS_URL=redis://127.0.0.1:16380/0
FUNDLIVE_REDIS_KEY_PREFIX=fundlive
SMTP_HOST=smtp.resend.com
SMTP_PORT=587
SMTP_USERNAME=resend
SMTP_PASSWORD=<仅发送权限的 Resend API Key>
SMTP_FROM=fundlive@mail.wrenzeal.top
SMTP_FROM_NAME=FundLive
SMTP_SECURITY=starttls
SMTP_TIMEOUT=15s
```

可使用 `openssl rand -hex 32` 生成验证码 HMAC 密钥。生产启用前必须确认 `mail.wrenzeal.top` 已在 Resend 完成 SPF/DKIM 验证。

部署脚本会把 `/etc/fund-live/fundlive.env` 作为 systemd `EnvironmentFile` 加载。DragonFly 应只监听 loopback 或私网地址；若跨主机访问，必须增加认证与网络访问控制。

## 安全规则

- 验证码为 6 位数字，10 分钟有效，成功后立即消费。
- 同一邮箱 60 秒内不能重复发送；每邮箱每小时最多 5 次，每 IP 每小时最多 20 次。
- 连续输错 5 次会使当前验证码失效。
- DragonFly 仅保存 HMAC 后的验证码，邮箱与 IP 也只作为哈希后的 key scope。
- 邮件发送失败会条件删除对应验证码和冷却锁，不会误删并发生成的新验证码。
- DragonFly 或 SMTP 不可用时，仅验证码登录降级；密码、Google 和其他业务 API 保持可用。

## API

- `POST /api/v1/auth/email/start`：发送验证码。
- `POST /api/v1/auth/email/verify`：验证验证码并设置现有 HttpOnly Session Cookie。
- `GET /api/v1/auth/config`：返回验证码与 Google 登录的当前可用状态。

错误验证码返回 HTTP 400；冷却或限流返回 HTTP 429 并包含 `retry_after_seconds`；邮件发送失败返回 502；验证码依赖不可用返回 503。
