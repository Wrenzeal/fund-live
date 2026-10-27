# FundLive 多服务服务器部署说明

适用：Ubuntu 24.04 LTS、x86_64。服务器可以运行其他应用；PostgreSQL、DragonFly 独立安装，FundLive 使用专属数据库、角色和缓存命名空间。

顺序：检查现有服务 → 安装或复用 PG / DragonFly → 创建 FundLive 数据库和表结构 → 部署后端 → 接入前端。建表使用独立 SQL，不依赖启动 FundLive API。

以下服务器命令均在 root 的 Bash 终端执行，Windows 上传命令单独标明。本说明不停止、覆盖或删除其他应用资源。已有服务与本文名字、目录、端口冲突时，先改示例再执行。

## 1. 拓扑与现有服务检查

| 组件 | 推荐方式 | 新装示例 |
| --- | --- | --- |
| PostgreSQL 15 | 独立 Docker；已有实例可复用 | 宿主机 127.0.0.1:15432；Docker 内 postgres:5432 |
| DragonFly 1.39.0 | 独立 Docker；已有实例可复用 | 宿主机 127.0.0.1:16380；Docker 内 dragonfly:6379 |
| Go API | 宿主机二进制 + systemd | 13896，公网由反向代理转发 |
| Nginx / HTTPS | 复用现有代理或新装 Nginx | API 域名 80/443 |
| 前端 | Vercel | BACKEND_URL 指向 API |
| Lean Worker | 可选 Docker | 加入数据服务网络 |

~~~bash
cat /etc/os-release
uname -m
ss -lntp
command -v docker || true
command -v nginx || true
~~~

已有 Docker 时检查：

~~~bash
docker version
docker compose version
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Ports}}'
docker volume ls
docker network ls
~~~

- 已有 PG：使用其管理员账号创建独立的 fundlive 库和角色，保留其他数据库；跳过新装 PG。
- 已有 DragonFly：复用实际地址和凭据，选择未占用的数据库编号及 fundlive 前缀。下文默认编号 1。
- 缓存编号、前缀只是逻辑分隔，不是安全隔离。不同应用不应共享访问权限或资源时，使用独立缓存实例。
- 已有 80/443 代理时增加 API 域名，不另起 Nginx 抢占端口。15432、16380、13896 被占用时同步更改端口和应用配置。
- 本文的新装基础设施使用 /opt/service-infra、/etc/service-infra、Compose 项目 infra，与应用目录独立。已有同名资源时更换名称，不能覆盖。

云安全组开放实际 SSH 端口、80 和 443。数据库、缓存仅映射 loopback。当前 API 监听所有网卡，13896 不对公网开放。

## 2. 安装 Docker 与工具

没有 Docker 时使用 Ubuntu 软件源：

~~~bash
apt update
apt install -y docker.io docker-compose-v2 curl ca-certificates openssl
systemctl enable --now docker
docker version
docker compose version
~~~

已有 Docker 时复用安装，只补缺失工具，不替换或重启其他应用正在使用的 Engine。若包不存在，检查 noble / noble-updates 和 universe 软件源；也可择一采用 [Docker 官方安装](https://docs.docker.com/engine/install/ubuntu/)，不要混装。

本文需要 Compose v2 的 up --wait。运行预编译后端不需要 Go、Node.js、npm、PM2。采用 Docker 数据服务后不需要安装宿主机 postgresql、redis-server；DragonFly 提供本项目需要的 Redis 协议服务。

## 3. 新装 PG 与 DragonFly

已有可用实例时跳过本节。只缺其中一个时，仅保留对应 service 和 volume。仓库根目录 docker-compose.yml 是开发示例，不作为生产配置。

### 3.1 独立目录和密钥

以下只用于首次安装，已有同名资源时停止，改路径或复用原配置：

~~~bash
bash -euo pipefail <<'SH'
if [ -e /opt/service-infra ] || [ -e /etc/service-infra ] ||
   docker volume inspect infra_postgres15_data >/dev/null 2>&1 ||
   docker volume inspect infra_dragonfly139_data >/dev/null 2>&1; then
  echo "已有基础设施资源；请先核对用途或更换名称。"
  exit 1
fi
umask 077
install -d -m 755 /opt/service-infra
install -d -m 700 /etc/service-infra
infra_pg_password=$(openssl rand -hex 24)
infra_cache_password=$(openssl rand -hex 24)
cat > /etc/service-infra/compose.env <<EOF
POSTGRES_ADMIN_PASSWORD=$infra_pg_password
DRAGONFLY_PASSWORD=$infra_cache_password
EOF
unset infra_pg_password infra_cache_password
chmod 600 /etc/service-infra/compose.env
SH
~~~

### 3.2 Compose 安装配置

~~~bash
cat > /opt/service-infra/compose.yml <<'YAML'
services:
  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    shm_size: 256mb
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${POSTGRES_ADMIN_PASSWORD:?missing POSTGRES_ADMIN_PASSWORD}
      POSTGRES_DB: postgres
      TZ: Asia/Shanghai
    ports:
      - "127.0.0.1:15432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d postgres"]
      interval: 5s
      timeout: 5s
      retries: 24
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

  dragonfly:
    image: docker.dragonflydb.io/dragonflydb/dragonfly:v1.39.0
    restart: unless-stopped
    environment:
      DFLY_requirepass: ${DRAGONFLY_PASSWORD:?missing DRAGONFLY_PASSWORD}
    command:
      - "--logtostderr"
      - "--dir=/data"
      - "--dbfilename=dump"
      - "--snapshot_cron=*/5 * * * *"
    ports:
      - "127.0.0.1:16380:6379"
    volumes:
      - dragonfly_data:/data
    healthcheck:
      test: ["CMD-SHELL", "REDISCLI_AUTH=\"$$DFLY_requirepass\" redis-cli ping | grep -qx PONG"]
      interval: 5s
      timeout: 5s
      retries: 24
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  postgres_data:
    name: infra_postgres15_data
  dragonfly_data:
    name: infra_dragonfly139_data
YAML
chmod 644 /opt/service-infra/compose.yml

cat > /usr/local/sbin/infra-compose <<'SH'
#!/usr/bin/env bash
set -euo pipefail
exec /usr/bin/docker compose \
  --env-file /etc/service-infra/compose.env \
  --project-name infra \
  --file /opt/service-infra/compose.yml "$@"
SH
chmod 755 /usr/local/sbin/infra-compose
infra-compose config --quiet
~~~

后续统一使用 infra-compose。不要公开未加 --quiet 的渲染配置或 docker inspect 输出，可能包含密码。

### 3.3 独立启动与开机恢复

~~~bash
cat > /etc/systemd/system/service-infra.service <<'UNIT'
[Unit]
Description=Shared PostgreSQL and DragonFly infrastructure
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
TimeoutStartSec=240
ExecStart=/usr/local/sbin/infra-compose up -d --wait --wait-timeout 180 postgres dragonfly

[Install]
WantedBy=multi-user.target
UNIT

chmod 644 /etc/systemd/system/service-infra.service
systemctl daemon-reload
systemctl enable --now service-infra
infra-compose ps
infra-compose exec -T postgres pg_isready -U postgres -d postgres
infra-compose exec -T dragonfly sh -ec 'REDISCLI_AUTH="$DFLY_requirepass" redis-cli ping'
~~~

预期容器健康、PG 接受连接、DragonFly 返回 PONG。网络名为 infra_default。基础设施独立运行，停止 FundLive 不会停止它们。

PG 数据保存在 infra_postgres15_data，缓存快照保存在 infra_dragonfly139_data。快照每 5 分钟执行，异常退出可能丢失快照后的写入；不是队列零丢失保证。容量、连接数、缓存内存按整台服务器规划；volume 不能替代异地备份。

### 3.4 修复旧密码环境变量配置

DragonFly 1.39.0 按大小写敏感的参数名解析 DFLY_ 环境变量。密码使用 DFLY_requirepass，不能使用 DFLY_PASSWORD；后者会报 unknown environment variable 并以 SIGABRT（退出码 134）终止。服务器密钥文件中的 DRAGONFLY_PASSWORD 保持原名，不需要重新生成密码。

如果此前按照旧版文档部署，在原 Compose 文件上修复，仅重新创建 DragonFly，保留 PG 和所有 volume：

~~~bash
cp /opt/service-infra/compose.yml /opt/service-infra/compose.yml.before-auth-fix
sed -i 's/DFLY_PASSWORD/DFLY_requirepass/g' /opt/service-infra/compose.yml
infra-compose config --quiet
infra-compose up -d --no-deps --wait --wait-timeout 120 dragonfly
infra-compose exec -T dragonfly sh -ec 'REDISCLI_AUTH="$DFLY_requirepass" redis-cli ping'
~~~

确认返回 PONG 后，重新启动此前失败的基础设施单元并核对状态：

~~~bash
systemctl restart service-infra
infra-compose ps
~~~

## 4. 在部署应用前建库建表

下面先创建 fundlive 角色和库，随后导入表结构，不安装、不启动 API。

### 4.1 FundLive 专属凭据

新装实例使用以下命令。已有 PG/缓存时按实际地址、端口和凭据准备同格式的环境文件，不重置共享实例的管理员密码：

~~~bash
bash -euo pipefail <<'SH'
if [ -e /etc/fund-live/fundlive.env ]; then
  echo "已有 FundLive 配置；先核对，避免覆盖密码。"
  exit 1
fi
umask 077
install -d -m 700 /etc/fund-live
. /etc/service-infra/compose.env
fundlive_db_password=$(openssl rand -hex 24)
cat > /etc/fund-live/fundlive.env <<EOF
APP_ENV=production
FUNDLIVE_CONFIG=/etc/fund-live/fundlive.yaml
STORAGE_MODE=postgres
PORT=13896
DB_HOST=127.0.0.1
DB_PORT=15432
DB_USER=fundlive
DB_NAME=fundlive
DB_PASSWORD=$fundlive_db_password
DB_SSLMODE=disable
DB_AUTO_MIGRATE=false
REDIS_URL=redis://:$DRAGONFLY_PASSWORD@127.0.0.1:16380/1
FUNDLIVE_REDIS_KEY_PREFIX=fundlive
AUTH_COOKIE_SECURE=true
AUTH_EMAIL_CODE_ENABLED=false
EOF
unset fundlive_db_password
chmod 600 /etc/fund-live/fundlive.env
SH
~~~

编号 1 和前缀 fundlive 必须未被其他应用占用。示例密码是随机十六进制，可直接放入 URL；复用缓存时，特殊字符密码需 URL 编码。

### 4.2 创建角色与数据库

先查看现有资源：

~~~bash
infra-compose exec -T postgres psql -U postgres -d postgres -c '\du'
infra-compose exec -T postgres psql -U postgres -d postgres -c '\l'
~~~

确认 fundlive 角色和库不存在后，只执行一次：

~~~bash
bash -euo pipefail <<'SH'
. /etc/fund-live/fundlive.env
printf "CREATE ROLE fundlive LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD '%s';\nCREATE DATABASE fundlive OWNER fundlive;\n" \
  "$DB_PASSWORD" |
  /usr/local/sbin/infra-compose exec -T postgres \
  psql -U postgres -d postgres -v ON_ERROR_STOP=1
SH
~~~

fundlive 是本库所有者，不是 PG 超级用户，不允许创建其他数据库或角色。已有实例时让该实例管理员执行同样 SQL；把 infra-compose exec 替换为已有容器的 docker exec 或原生 psql 入口。不修改其他应用角色或数据库。

### 4.3 独立导入表结构

仓库提供 [fundlive-schema.sql](../scripts/sql/fundlive-schema.sql)，从当前 26 项迁移按顺序生成，包含 42 张业务表、schema_migrations 表、索引和迁移版本。它不包含旧用户或行情数据。

在 Windows PowerShell 单独上传 SQL，此时不需要应用程序：

~~~powershell
scp D:\Codex\fund-live\scripts\sql\fundlive-schema.sql root@服务器IP:/tmp/
~~~

服务器使用应用角色导入，单个事务、报错即停止：

~~~bash
infra-compose exec -T postgres \
  psql -U fundlive -d fundlive -v ON_ERROR_STOP=1 \
  < /tmp/fundlive-schema.sql
~~~

命令通过容器内本地 Unix socket 由管理员执行。已有实例要求密码时按其认证规则提供，不关闭认证。原生 PG 可使用 psql 或 sudo -u postgres psql，但应 SET ROLE fundlive 后导入，不能用 postgres 角色直接导入此文件。

SQL 会确认库名与角色均为 fundlive，拒绝已有用户表、序列或视图的数据库。它只用于全新空库，不重复导入，不当成升级脚本。需要 pg_trgm 扩展；PG 15 的数据库所有者可安装该 trusted 扩展，已有实例限制安装时由管理员在此库准备。

~~~bash
infra-compose exec -T postgres psql -U fundlive -d fundlive -v ON_ERROR_STOP=1 \
  -c "SELECT count(*) AS tables FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE';" \
  -c "SELECT count(*) AS applied_migrations FROM schema_migrations;" \
  -c "SELECT extname FROM pg_extension WHERE extname='pg_trgm';"
~~~

当前版本预期为 43 张表、26 项迁移和 pg_trgm。全部成功再继续；失败时不能通过启动 API 代替修复。

迁移源码变更时在开发环境重新导出：

~~~powershell
go run ./cmd/export-db-schema --output scripts/sql/fundlive-schema.sql
~~~

导出不连接数据库、不启动 API，服务器只需要 SQL 和 psql，不需要 Go。

## 5. 部署 FundLive 后端

### 5.1 安装程序与配置

库和表已准备完成，现在上传后端包。在 Windows PowerShell：

~~~powershell
scp D:\Codex\fund-live\target\fund-live-backend-linux-amd64.tar.gz root@服务器IP:/tmp/
~~~

~~~bash
id -u fundlive >/dev/null 2>&1 ||
  useradd --system --home /var/lib/fund-live --shell /usr/sbin/nologin fundlive
install -d -m 755 /opt/fund-live
install -d -m 750 -o fundlive -g fundlive /var/lib/fund-live
chown root:fundlive /etc/fund-live
chmod 750 /etc/fund-live
tar -xzf /tmp/fund-live-backend-linux-amd64.tar.gz -C /opt/fund-live
chmod 755 /opt/fund-live/fund-live-backend /opt/fund-live/fund-live-crawler
~~~

环境文件保持 root:root、600，由 systemd 管理进程读取。服务用户只需读取无密码的 YAML：

~~~bash
cat > /etc/fund-live/fundlive.yaml <<'YAML'
server:
  port: 13896
  allowed_origins:
    - https://fund.wrenzeal.top
storage:
  mode: postgres
cache:
  key_prefix: fundlive
quote:
  default_source: sina
  overseas_source: tencent
database:
  host: 127.0.0.1
  port: 15432
  user: fundlive
  name: fundlive
  ssl_mode: disable
  timezone: Asia/Shanghai
  log_level: warn
  auto_migrate: false
auth:
  cookie_name: fundlive_session
  cookie_secure: true
  session_ttl_hours: 720
  google_client_id: ""
  email_code_enabled: false
YAML
chown root:fundlive /etc/fund-live/fundlive.yaml
chmod 640 /etc/fund-live/fundlive.yaml
~~~

实际数据库与缓存地址由环境文件覆盖，不使用包里的旧原生数据库配置。

### 5.2 启动应用

新装数据服务时依赖 service-infra.service；复用已有实例时改为其实际 systemd 单元，远程数据库只保留 network-online.target。不在应用部署时重建基础设施。

~~~bash
cat > /etc/systemd/system/fund-live-backend.service <<'UNIT'
[Unit]
Description=FundLive API
Requires=service-infra.service
After=service-infra.service network-online.target
Wants=network-online.target

[Service]
Type=simple
User=fundlive
Group=fundlive
WorkingDirectory=/opt/fund-live
Environment=FUNDLIVE_CONFIG=/etc/fund-live/fundlive.yaml
EnvironmentFile=/etc/fund-live/fundlive.env
ExecStart=/opt/fund-live/fund-live-backend
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
UNIT
chmod 644 /etc/systemd/system/fund-live-backend.service
systemctl daemon-reload
systemctl enable --now fund-live-backend
journalctl -u fund-live-backend -n 100 --no-pager
curl -fsS http://127.0.0.1:13896/health
~~~

health 应返回 status: ok、storage_mode: postgres，日志不应有迁移失败。

当前程序仍会检查独立迁移版本并执行 CREATE TABLE IF NOT EXISTS schema_migrations。auto_migrate: false 只关闭 GORM 自动迁移；预导入全部迁移后，业务表迁移会跳过已应用版本。启动仍写入估值档案、分类、主题和试点标的等业务种子。

本文实现先建库建表再部署，不宣称运行时完全没有 DDL 检查。若要求应用角色完全禁止建表、启动也禁止迁移，需另外修改迁移开关和角色设计。

## 6. 基金数据与账号

空库无旧用户、持仓、自选和流水；无备份时需重建私人数据。公开数据可以采集：

~~~bash
bash -euo pipefail <<'SH'
set -a
. /etc/fund-live/fundlive.env
set +a
cd /opt/fund-live
./fund-live-crawler --list all --save-db --catalog-only
./fund-live-crawler --codes 020465,018123,016874 --save-db
./fund-live-crawler --history-only --tracked-only --history-days 30 --save-db
SH
~~~

按需替换基金代码。tracked-only 仅处理已有持仓、收藏、自选；空库没有对应用户样本。

首个注册用户不是管理员，注册后对真实邮箱赋权：

~~~bash
infra-compose exec -T postgres psql -U fundlive -d fundlive -v ON_ERROR_STOP=1 \
  -c "UPDATE tb_user SET is_admin = true WHERE email = '你的管理员邮箱';"
~~~

预期 UPDATE 1，重新登录核对。

## 7. 域名、反向代理与 Vercel

复用现有代理，不另抢 80/443。没有代理时安装宿主机 Nginx：

~~~bash
apt update
apt install -y nginx certbot python3-certbot-nginx
systemctl enable --now nginx
~~~

已有 Nginx 只补缺失的 Certbot。API 示例域名 api.fund.wrenzeal.top 指向服务器 IP，前端 fund.wrenzeal.top 由 Vercel 提供。修正旧 A/AAAA，确认同名站点未占用，再添加 API：

~~~bash
install -m 644 /opt/fund-live/nginx-api.conf /etc/nginx/sites-available/fund-live-api
ln -sfn /etc/nginx/sites-available/fund-live-api /etc/nginx/sites-enabled/fund-live-api
nginx -t
systemctl reload nginx
certbot --nginx -d api.fund.wrenzeal.top
curl -fsS https://api.fund.wrenzeal.top/health
certbot renew --dry-run
~~~

Caddy、Traefik 或容器代理使用它们自己的配置，不执行本段的 Nginx 安装。代理访问 API 的地址取决于所在网络；宿主机 Nginx 使用 127.0.0.1:13896。

Vercel 设置后重新部署：

~~~text
BACKEND_URL=https://api.fund.wrenzeal.top
~~~

不使用 localhost、127.0.0.1 或前端域名，通常不设 NEXT_PUBLIC_API_URL。研究页由 Next.js 提供，上传后端包不会发布前端改动。

~~~bash
curl -i https://fund.wrenzeal.top/health
curl -i https://fund.wrenzeal.top/api/v1/auth/config
curl -i https://fund.wrenzeal.top/api/v1/market/status
~~~

详见 [Vercel 说明](vercel-frontend-deploy.md)。

## 8. 验证码与 Google 登录（可选）

默认关闭验证码，密码注册登录仍可用。确认 SMTP 发件域名和凭据后，将环境文件 AUTH_EMAIL_CODE_ENABLED 改为 true，再添加：

~~~text
AUTH_EMAIL_DRIVER=smtp
AUTH_EMAIL_CODE_SECRET=由openssl_rand_hex_32生成的随机密钥
SMTP_HOST=smtp.resend.com
SMTP_PORT=587
SMTP_USERNAME=resend
SMTP_PASSWORD=你的Resend发送密钥
SMTP_FROM=fundlive@mail.wrenzeal.top
SMTP_FROM_NAME=FundLive
SMTP_SECURITY=starttls
SMTP_TIMEOUT=15s
~~~

密钥用 openssl rand -hex 32 生成，环境文件保持 root:root、600，可被 Bash 和 systemd 读取。Google 登录需要后端 GOOGLE_CLIENT_ID 与 Vercel NEXT_PUBLIC_GOOGLE_CLIENT_ID 一致，Google Console 允许前端来源。

~~~bash
systemctl restart fund-live-backend
curl -fsS https://fund.wrenzeal.top/api/v1/auth/config
~~~

随后验证真实登录，见 [邮箱验证码说明](email-code-login.md)。

## 9. Lean Worker（可选）

另上传完整源码至 /opt/fund-live-source。Docker 多阶段构建无需宿主机 Go / .NET；后端包不含 Lean 或试点数据初始化工具。

本例网络 infra_default，PG 地址 postgres:5432，缓存 dragonfly:6379/1；已有实例必须改为实际网络。不能用 host.docker.internal 访问仅绑定宿主机 loopback 的 PG 映射。

~~~bash
bash -euo pipefail <<'SH'
. /etc/fund-live/fundlive.env
umask 077
case "$REDIS_URL" in
  *@127.0.0.1:16380/1) ;;
  *) echo "已有缓存拓扑需单独准备 Worker 配置。"; exit 1 ;;
esac
worker_cache_url="${REDIS_URL%@127.0.0.1:16380/1}@dragonfly:6379/1"
cat > /etc/fund-live/lean-worker.yaml <<EOF
database:
  host: postgres
  port: 5432
  user: $DB_USER
  password: "$DB_PASSWORD"
  name: $DB_NAME
  ssl_mode: disable
  timezone: Asia/Shanghai
  auto_migrate: false
cache:
  redis_url: "$worker_cache_url"
  key_prefix: fundlive
EOF
cat > /etc/fund-live/lean-worker.env <<EOF
LEAN_REDIS_URL=$worker_cache_url
EOF
chmod 600 /etc/fund-live/lean-worker.yaml /etc/fund-live/lean-worker.env
SH

bash -euo pipefail <<'SH'
set -a
. /etc/fund-live/lean-worker.env
set +a
cd /opt/fund-live-source
LEAN_CONFIG_PATH=/etc/fund-live/lean-worker.yaml \
LEAN_DOCKER_NETWORK=infra_default \
LEAN_DB_HOST=postgres \
LEAN_JOB_VOLUME=fundlive_lean_jobs \
bash scripts/deploy-lean-worker.sh
SH
docker logs --tail=100 fundlive_lean_worker
~~~

预期 Lean worker ready。缓存编号和前缀与 API 一致。按 [Lean 说明](lean-quant-validation.md) 准备历史行情、信号和任务；其中 go run 工具需准备工具链或预编译命令。仅启动 Worker 不生成历史数据。

## 10. 备份、恢复与更新

只备份 fundlive 库，不导出其他应用数据：

~~~bash
bash -euo pipefail <<'SH'
umask 077
install -d -m 700 /var/backups/fund-live
fundlive_backup="/var/backups/fund-live/fundlive-$(date +%F-%H%M%S).dump"
/usr/local/sbin/infra-compose exec -T postgres \
  pg_dump -U fundlive -d fundlive -Fc > "$fundlive_backup.partial"
mv "$fundlive_backup.partial" "$fundlive_backup"
SH
~~~

定期执行并异地保存，配置和密钥加密备份。已有备份核对格式和 PG 版本，恢复到新库，不覆盖其他库：

~~~bash
systemctl stop fund-live-backend
# 仅已运行 Lean 时执行：docker stop fundlive_lean_worker
infra-compose exec -T postgres createdb -U postgres -O fundlive fundlive_restore
infra-compose exec -T postgres \
  pg_restore -U fundlive -d fundlive_restore --no-owner --no-privileges --exit-on-error \
  < /var/backups/fund-live/你的备份.dump
infra-compose exec -T postgres psql -U fundlive -d fundlive_restore -c '\dt'
~~~

成功核对后更新 API 的 DB_NAME、YAML 和 Worker 库名并重启。以上为 pg_dump -Fc 格式；纯 SQL 使用 psql。不要对已恢复库再次导入空库初始化文件。

缓存维护由基础设施管理员安排，不因 FundLive 更新停止共享缓存或删除 volume。主动快照：

~~~bash
infra-compose exec -T dragonfly sh -ec 'REDISCLI_AUTH="$DFLY_requirepass" redis-cli SAVE'
~~~

共享环境不执行 down -v、FLUSHALL 或无范围的 FLUSHDB。

后端更新先上传新包、备份 FundLive 库，只停止 API 后替换：

~~~bash
install -d -m 755 /tmp/fund-live-update
tar -xzf /tmp/fund-live-backend-linux-amd64.tar.gz -C /tmp/fund-live-update
systemctl stop fund-live-backend
cp /opt/fund-live/fund-live-backend /opt/fund-live/fund-live-backend.previous
cp /opt/fund-live/fund-live-crawler /opt/fund-live/fund-live-crawler.previous
install -m 755 /tmp/fund-live-update/fund-live-backend /opt/fund-live/fund-live-backend
install -m 755 /tmp/fund-live-update/fund-live-crawler /opt/fund-live/fund-live-crawler
systemctl start fund-live-backend
curl -fsS http://127.0.0.1:13896/health
~~~

schema 升级在启动新版前单独审核、应用该版本迁移，不重跑空库文件。程序回滚不撤销数据库迁移，先确认旧程序兼容。配置不从压缩包示例覆盖。

~~~bash
infra-compose ps
infra-compose logs --tail=100 postgres dragonfly
systemctl status fund-live-backend --no-pager
journalctl -u fund-live-backend -n 100 --no-pager
ss -lntp | grep -E ':(13896|15432|16380)\b'
~~~

## 参考与验证范围

- [Compose 插值](https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/)：统一项目、环境文件和路径。
- [PostgreSQL 镜像入口](https://github.com/docker-library/postgres/blob/master/docker-entrypoint.sh)：只初始化空数据目录，环境变量不会重置已有库密码。
- [DragonFly 参数](https://github.com/dragonflydb/dragonfly/blob/v1.39.0/src/server/server_family.cc)：持久化参数与 requirepass。
- [DragonFly 环境变量解析](https://github.com/dragonflydb/dragonfly/blob/v1.39.0/src/server/dfly_main.cc)：DFLY_ 后必须是合法参数名，例如 DFLY_requirepass；未知名称会中止启动。

本说明包含新装与复用路径，SQL 从当前迁移生成。实际服务器需验证容器、表结构、迁移记录、API/前端、登录持仓以及备份恢复。
