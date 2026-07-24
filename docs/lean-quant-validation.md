# Lean 量化验证

FundLive 继续负责评分、事件解释与基金映射。QuantConnect Lean 只承担组合回测、成交、费用、滑点、回撤和基准比较，不参与线上 V4 分数计算。

当前固定实验为 `risk-v1`，所有变体共享同一标的池、信号、样本区间、费用和成交规则：

| 变体 | 规则 |
| --- | --- |
| A | 周度 Top 5 等权；跌出 Top 5 后退出 |
| B | 周度 Top 5 进入；跌出 Top 8 后退出 |
| C | 月度调仓、Top 8 退出缓冲；目标权重变化不足 250bps 不交易 |
| D | C + 20 日逆波动率、单只 30% 上限、沪深300 120 日趋势过滤；弱市目标敞口 50% |

## 数据边界

- `quant_event_versions` 按版本保存事件，历史查询必须满足 `known_at <= decision_at`。
- `expected` 表示已有官方日程或已公告计划，不代表分析师一致预期。
- 新闻和无来源规则只作为补充；影子事件分不会改写 V4 总分、推荐比例或排行榜。
- `quant_signal_history` 每个基金、版本、日期和模式只写一次。历史代理使用 `historical_proxy_v1`，真实前向信号使用 `full_v4_forward`。
- 公共行情只用于试点。正式对外提供策略结论前，应复核数据许可、复权、分红和停牌质量。

## 初始化五年试点数据

先启动 PostgreSQL 与 Dragonfly：

```bash
docker compose up -d db cache
go run ./cmd/sync-quant-market-data --years 5
go run ./cmd/build-quant-proxy-signals
```

行情同步会初始化 `pilot-v1`：25只宽基、风格、行业主题、债券/现金、黄金和境外指数 ETF，并同步沪深300基准。日线同时保存原始价格、前复权收盘价、推导复权因子、成交量和成交额。

检查原生统计：

```bash
curl 'http://127.0.0.1:8080/api/v1/quant/validation?mode=historical_proxy'
curl 'http://127.0.0.1:8080/api/v1/quant/universe'
```

## 启动 Lean Worker

Worker 镜像从 Lean 提交 `0136529cd8d9194f401aa5322bf90e547d1f0b56` 构建，不挂载宿主机 Docker Socket：

```bash
docker compose --profile quant up -d --build lean-worker
docker compose logs -f lean-worker
```

上面的 Compose 配置只用于本地环境。生产服务器使用既有 PostgreSQL 与 Dragonfly 时，运行：

```bash
./scripts/deploy-lean-worker.sh
docker logs -f fundlive_lean_worker
```

部署脚本以只读方式挂载 `/etc/fund-live/fundlive.yaml`，通过 Docker host gateway 连接生产 PostgreSQL，并通过 `fund_default` 网络连接 Dragonfly。数据库密码不会写入镜像或 Docker 环境变量。可通过 `LEAN_CONFIG_PATH`、`LEAN_ENV_PATH`、`LEAN_DOCKER_NETWORK` 和 `LEAN_REDIS_URL` 覆盖生产拓扑。

该 Lean 版本自带的结果诊断固定请求美股 SPY，不适用于 FundLive 的中国 ETF 基准，镜像构建时会关闭这项诊断。Lean 的组合净值、订单、成交、费用、回撤和统计结果仍照常生成；基准比较使用任务内导出的沪深300、试点池等权与现金曲线。

Worker 配置：

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `REDIS_URL` | `redis://127.0.0.1:16380/0` | Dragonfly 地址 |
| `REDIS_KEY_PREFIX` | `fundlive` | Stream 前缀 |
| `LEAN_ENGINE_VERSION` | 固定 Lean 提交 | 写入回测结果 |
| `LEAN_DATA_ROOT` | `/Lean/Data` | Lean 内置市场时段与标的属性数据 |
| `LEAN_JOB_ROOT` | `/var/lib/fundlive/lean-jobs` | 点时输入与结果目录 |
| `LEAN_JOB_TIMEOUT_MINUTES` | `30` | 单任务超时 |
| `LEAN_WORKER_GROUP` | `lean-workers` | Redis Streams 消费组 |

数据库变量继续使用 `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME` 和 `DB_SSLMODE`。

## 创建回测

创建任务是管理员接口，浏览器登录管理员后可通过同源 API 调用：

```bash
curl -X POST 'http://127.0.0.1:8080/api/v1/admin/quant/backtests' \
  -H 'Content-Type: application/json' \
  -H 'Cookie: fundlive_session=<session>' \
  -d '{
    "start_date":"2021-07-01",
    "end_date":"2026-07-01",
    "universe_version":"pilot-v1",
    "signal_mode":"historical_proxy",
    "initial_cash":"1000000",
    "top_n":5,
    "commission_bps":"3",
    "minimum_commission_cny":"5",
    "slippage_bps":"5",
    "minimum_listing_days":120,
    "minimum_average_amount":"20000000"
  }'
```

同一参数会返回同一任务，避免重复消耗。任务状态和结果：

```bash
curl 'http://127.0.0.1:8080/api/v1/quant/backtests?limit=10'
curl 'http://127.0.0.1:8080/api/v1/quant/backtests/<job-id>'
```

策略使用每周最后一个交易日收盘信号，在下一交易日以市场单成交；选取 Top 5、每只目标权重20%。上市不足120个交易日、近20日平均成交额低于2000万元或缺少价格的标的不参与，不足5只时保留现金。

## 运行固定 A-D 实验

先确保历史代理信号已生成且 Lean Worker 正常消费 Dragonfly Stream：

```bash
go run ./cmd/run-quant-experiment --preset risk-v1
```

命令会按共同可用日期自动确定区间，幂等创建四个任务并入队。也可显式传入 `--start` 和 `--end`。输出的 `experiment` ID 可通过以下接口和页面查看：

```bash
curl 'http://127.0.0.1:8080/api/v1/quant/backtest-experiments?limit=6'
curl 'http://127.0.0.1:8080/api/v1/quant/backtest-experiments/<experiment-id>'
```

```text
/analysis/experiments/<experiment-id>
```

Worker 会保留 Lean 原始结果，并写入稳定的 `quant-backtest.v1` 结果：策略、沪深300、试点池等权、现金与回撤曲线，以及收益、CAGR、回撤、Sharpe、Sortino、信息比率、换手、费用、订单和自然年收益。曲线只保留沪深300实际交易日；调仓按 1 倍现金账户目标计算数量，先提交减仓后提交加仓，换仓批次结算后恢复 1 倍买入力，并预留 1% 组合价值覆盖费用和滑点。当前实现版本为 `risk-v1.2`，实现版本会进入任务和实验幂等键。B-D 只有同时改善 Sharpe 和最大回撤、CAGR 相对 A 少于 1 个百分点，并在至少 3 个完整年度跑赢 A，才显示“风险收益改善”。

## 生产更新与回填

从仓库根目录按顺序执行：

```bash
./scripts/deploy-backend.sh
./scripts/deploy-lean-worker.sh
./scripts/deploy-frontend.sh

set -a
source /etc/fund-live/fundlive.env
set +a
export FUNDLIVE_CONFIG=/etc/fund-live/fundlive.yaml

go run ./cmd/recompute-fund-analysis-snapshots --limit 120
go run ./cmd/run-quant-experiment --preset risk-v1
```

后端启动时会应用事件来源字段宽度和实验表迁移。回填完成后，`/api/v1/quant/validation?mode=historical_proxy` 的 `coverage` 应出现事件、映射基金、实验数量和最新数据时间；容器日志中不应再出现 `value too long for type character varying(16)` 或分时唯一键冲突。

## 外部事件研究器

TradingAgents-CN 本轮只预留 [`event-intel.v1`](event-intel-shadow-contract.md) shadow 契约。它不能直接调用 Lean、修改 V4 分数或生成生产交易信号；未来导入前还需完成来源许可、事件去重、点时泄漏和前向验证。

## 事件接口

```bash
curl 'http://127.0.0.1:8080/api/v1/fund/005827/events?status=disclosed&as_of=2026-07-23'
```

`as_of` 支持 RFC3339 或 `YYYY-MM-DD`。日期格式按上海时区当日结束处理。“最新”是按 `known_at` 排序后的展示方式，不是事件状态。
