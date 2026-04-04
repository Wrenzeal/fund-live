# 持续优化记录

更新时间：2026-04-04

## 使用规则

- 按顺序推进，不跳过前置任务。
- 每次实际完成一轮优化后，都在本文件追加记录。
- 记录必须包含：优化方式、具体改动、解决的问题、遗留问题。
- 未完成项保留为 `pending` 或 `in_progress`，下次继续。

## 优化清单

1. `completed` 搜索索引 / 查询重写
2. `completed` 冷数据补全异步化
3. `completed` 关闭默认自动迁移，补 migration / 约束
4. `completed` 采集器并发化与 tracked fund 淘汰
5. `completed` 清理前端旧状态层 / 旧轮询
6. `completed` 修正文档、配置、运行说明漂移

## 执行记录

### 2026-04-04 第 1 项阶段 A：搜索查询重写与结果排序

- 状态：`in_progress`
- 优化方式：
  - 将原先单次 `contains` 搜索改为两阶段召回：先查精确 / 前缀命中，再查模糊命中。
  - 在仓储层增加统一的搜索命中分类与稳定排序逻辑，确保精确 ID、ID 前缀、名称前缀、名称包含、经理包含按优先级返回。
  - 为内存仓储和 PostgreSQL 仓储共用同一套排序规则，并补充单元测试。
- 本次改动：
  - 新增 `internal/repository/fund_search.go`
  - 新增 `internal/repository/fund_search_test.go`
  - 修改 `internal/repository/postgres_fund_repo.go`
  - 修改 `internal/repository/memory_fund_repo.go`
- 解决的问题：
  - 修复原搜索结果“只要包含就返回、顺序不稳定”的问题。
  - 降低 PostgreSQL 搜索对大范围 `%query%` 扫描的依赖，优先利用精确 / 前缀查询缩小候选集。
  - 统一不同仓储实现的搜索行为，便于后续继续做索引优化。
- 验证结果：
  - `go test ./internal/repository/...` 通过。
  - `go test ./...` 通过。
- 遗留问题：
  - PostgreSQL 的 trigram / pattern index 还未落地。
  - 项目内还没有统一 migration 机制，索引变更需要放到第 3 项一起完成。
  - 当前只完成了“查询重写”，第 1 项还不能标记为完成。
- 下次继续：
  - 继续完成第 1 项剩余部分，补可落地的索引方案与配套执行方式。

### 2026-04-04 第 1 项阶段 B：PostgreSQL 搜索索引落地

- 状态：`completed`
- 优化方式：
  - 为 PostgreSQL 增加 `pg_trgm` 扩展与基金搜索所需的模式索引 / trigram 索引。
  - 在数据库初始化阶段加入幂等化的搜索索引检查，避免后续环境遗漏索引。
  - 将索引同步应用到当前运行中的数据库实例，保证这轮优化不依赖下次重启才生效。
- 本次改动：
  - 新增 `internal/database/search_indexes.go`
  - 修改 `internal/database/db.go`
- 解决的问题：
  - 为基金代码前缀搜索提供可用索引。
  - 为基金名称 / 基金经理模糊匹配提供 trigram 索引基础。
  - 让第 1 项“搜索索引 / 查询重写”形成完整闭环，而不是只停留在查询逻辑层。
- 验证结果：
  - `go test ./...` 通过。
  - 当前运行库已存在 `idx_funds_id_pattern`、`idx_funds_name_trgm`、`idx_funds_manager_trgm`。
- 遗留问题：
  - 项目仍缺统一 migration 机制，当前索引保障属于初始化阶段的幂等补偿。
  - 第 3 项会继续把 schema / index 变更从启动期逻辑收敛到更正式的迁移流程。
- 下一项：
  - 第 2 项：冷数据补全异步化。

### 2026-04-04 第 2 项阶段 A：同步冷补全切换为后台预热

- 状态：`in_progress`
- 优化方式：
  - 在 `FundDataLoader` 中新增 warm cache 读取与后台预热调度，避免读请求直接同步抓取外部数据。
  - 将基金详情、持仓、估值、分时、联接基金目标 ETF 补全从“现场抓取”改成“优先读缓存，否则异步预热”。
  - 当估值/分时关键数据尚未准备好时，快速返回 `FUND_DATA_WARMING`，并带 `Retry-After` 提示，而不是把外部抓取延迟暴露给前端。
- 本次改动：
  - 修改 `internal/service/fund_data_loader.go`
  - 修改 `internal/service/valuation_service.go`
  - 修改 `internal/service/time_series_backfill.go`
  - 修改 `internal/service/fund_resolver.go`
  - 修改 `internal/handler/fund_handler.go`
  - 修改 `internal/service/fund_data_loader_test.go`
  - 修改 `internal/handler/fund_handler_test.go`
- 解决的问题：
  - 移除了估值 / 分时 / 基金详情 / 持仓接口中的同步冷抓取主路径，避免首个冷请求阻塞到外部数据源超时。
  - 为后台预热增加去重机制，避免同一基金被并发请求重复抓取。
  - 为前端增加可识别的预热状态：`warm_cache` / `warming` / `FUND_DATA_WARMING`。
- 验证结果：
  - `go test ./...` 通过。
- 遗留问题：
  - 前端尚未针对 `FUND_DATA_WARMING` 做专门的自动重试和友好提示。
  - 第 2 项还需要继续补 UI 侧的预热体验与可能的轮询回退策略。
- 下次继续：
  - 继续完成第 2 项剩余部分，处理前端对预热状态的展示与自动重试。

### 2026-04-04 第 2 项阶段 B：前端预热提示与自动重试

- 状态：`completed`
- 优化方式：
  - 在前端请求层识别 `FUND_DATA_WARMING`、`Retry-After` 和 `cache_status`。
  - 对估值和分时请求增加按 `Retry-After` 自动重试，不再把预热中的 503 当成普通报错。
  - 首页切换基金时，如果遇到预热中状态，不再回滚到上一个基金，而是保留当前选择并展示预热提示。
- 本次改动：
  - 修改 `web/src/hooks/use-fund-data.ts`
  - 修改 `web/src/app/page.tsx`
  - 修改 `web/src/components/loading-indicator.tsx`
- 解决的问题：
  - 冷基金首次打开时，前端不会再把预热中的 503 当成致命错误处理。
  - 页面会明确提示“数据预热中，正在自动重试”，并在成功后自动恢复正常显示。
  - 首页基金切换不再因为冷数据预热而回退到上一只基金。
- 验证结果：
  - `npm run lint` 通过。
  - `npm run build` 通过。
- 遗留问题：
  - 目前 warmup 提示只在首页接入；如果后续其他页面直接消费这些 hook，可按同样方式接入提示。
- 下一项：
  - 第 3 项：关闭默认自动迁移，补 migration / 约束。

### 2026-04-04 第 3 项：关闭默认自动迁移，补 migration / 约束

- 状态：`completed`
- 优化方式：
  - 将数据库默认配置的 `AutoMigrate` 从开启改为关闭，避免共享环境在启动时隐式修改 schema。
  - 新增受控 SQL migration 执行器，使用 `schema_migrations` 记录已应用版本。
  - 将搜索索引、`fund_history` 唯一索引、`fund_time_series` 去重与唯一索引纳入 migration。
  - 将 `fund_history` 与 `fund_time_series` 写路径改为基于唯一键的 upsert。
- 本次改动：
  - 新增 `internal/database/migrations.go`
  - 修改 `internal/database/db.go`
  - 修改 `internal/database/search_indexes.go`
  - 修改 `internal/database/models.go`
  - 修改 `internal/repository/postgres_fund_repo.go`
- 解决的问题：
  - 修复默认配置仍开启 `AutoMigrate` 的风险。
  - 为关键表补上受控唯一约束，避免重复写入和竞争窗口。
  - 运行库中的重复 `fund_time_series` 数据已完成去重，并成功补上唯一索引。
- 验证结果：
  - `go test ./...` 通过。
  - 当前运行库 `schema_migrations` 已记录 3 条迁移。
  - 当前运行库 `fund_time_series` 与 `fund_history` 重复键检查均为 0。
  - 当前运行库已存在 `uq_fund_history_fund_id_date`、`uq_fund_time_series_fund_id_time`。
- 遗留问题：
  - 首次空库建表目前仍依赖显式打开 `database.auto_migrate=true` 或后续补更完整的建表 migration。
- 下一项：
  - 第 4 项：采集器并发化与 tracked fund 淘汰。

### 2026-04-04 第 4 项：采集器并发化与 tracked fund 淘汰

- 状态：`completed`
- 优化方式：
  - 将后台采集从串行循环改为 `errgroup` 有限并发执行。
  - 为 tracked fund 增加 `LastTrackedAt` 与 TTL，长时间未访问的基金自动淘汰。
  - 在读取 tracked 集合前清理过期项，避免底层 slice 长期积累废数据。
- 本次改动：
  - 修改 `internal/service/valuation_service.go`
  - 修改 `internal/service/valuation_service_collector_test.go`
- 解决的问题：
  - 降低后台采集器随着 tracked 基金增长而整体变慢的风险。
  - 修复 tracked fund 只增不减、长时间运行后集合膨胀的问题。
- 验证结果：
  - `go test ./internal/service/...` 通过。
  - `go test ./...` 通过。
- 遗留问题：
  - 目前并发度仍是静态配置，后续如需更细粒度压测，可再做环境化参数。
- 下一项：
  - 第 5 项：清理前端旧状态层 / 旧轮询。

### 2026-04-04 第 5 项：清理前端旧状态层 / 旧轮询

- 状态：`completed`
- 优化方式：
  - 删除已脱离主链路、且只彼此互相引用的 Zustand `fund-store` 与 `refresh-timer` 组件。
  - 保持前端主数据流统一由 SWR 管理，避免未来误接回造成双请求和双状态源。
- 本次改动：
  - 删除 `web/src/store/fund-store.ts`
  - 删除 `web/src/components/refresh-timer.tsx`
- 解决的问题：
  - 清除前端遗留的双数据流隐患。
  - 降低未来维护者误启旧轮询逻辑的风险。
- 验证结果：
  - `rg -n "useFundStore|refresh-timer|fund-store" web/src` 无结果。
  - `npm run lint` 通过。
  - `npm run build` 通过。
- 遗留问题：
  - 无。
- 下一项：
  - 第 6 项：修正文档、配置、运行说明漂移。

### 2026-04-04 第 6 项：修正文档、配置、运行说明漂移

- 状态：`completed`
- 优化方式：
  - 对齐 README 顶部版本、当前能力说明和数据库迁移说明。
  - 为 CHANGELOG 增加 `2026.4.4` 条目，记录搜索、预热、migration 和旧状态层清理。
  - 对齐后端健康检查版本字段，避免 README / CHANGELOG / 运行时版本不一致。
- 本次改动：
  - 修改 `README.md`
  - 修改 `CHANGELOG.md`
  - 修改 `cmd/server/main.go`
- 解决的问题：
  - 修复 README 顶部版本号仍停留在旧版本的问题。
  - 修复文档未体现当前 migration、唯一约束和预热行为的问题。
  - 修复健康检查返回版本与文档版本不一致的问题。
- 验证结果：
  - `go test ./...` 通过。
  - `npm run build` 通过。
- 遗留问题：
  - 无。
