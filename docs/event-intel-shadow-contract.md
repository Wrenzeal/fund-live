# 事件情报 Shadow 契约

`event-intel.v1` 是 FundLive 为外部事件研究器预留的离线交换格式。TradingAgents-CN 可以作为候选生产者，但本轮没有运行时导入器、定时任务或线上评分写入路径。

契约文件：[`contracts/event-intel.v1.schema.json`](contracts/event-intel.v1.schema.json)

## 边界

- `mode` 固定为 `shadow`。外部结果不能修改 V4 总分、推荐比例、持仓或订单。
- 每条事件必须保留可访问的原始来源、来源发布时间、`known_at` 和映射依据。
- `known_at` 不得晚于 envelope 的 `as_of`；未来导入器还必须拒绝来源发布时间晚于回测决策时点的数据。
- `external_id + source.url` 应稳定指向同一逻辑事件。内容修订进入现有 `quant_event_versions`，不得覆盖旧版本。
- 模型推断放在可选的 `inference` 字段；事实来源和模型结论不能混为一个字段。
- 契约刻意不包含生产分数、目标仓位、买卖方向或 Lean 参数。

## 预期链路

```text
TradingAgents-CN / 其他研究器
  -> event-intel.v1 文件
  -> 隔离校验与人工抽查（后续实现）
  -> quant_events / quant_event_versions / quant_event_assets
  -> shadow_event_score
  -> 点时信号与 Lean 对照实验
```

只有完成来源许可、重复事件合并、点时泄漏测试和前向验证后，才考虑实现导入器。Lean 继续消费 FundLive 已冻结的信号与行情文件，不直接调用 TradingAgents-CN。

## 示例

```json
{
  "schema_version": "event-intel.v1",
  "mode": "shadow",
  "producer": {
    "name": "tradingagents-cn-adapter",
    "version": "0.1.0",
    "run_id": "20260724T090000Z"
  },
  "generated_at": "2026-07-24T09:00:00Z",
  "as_of": "2026-07-24T08:55:00Z",
  "events": [
    {
      "external_id": "example-announcement-001",
      "event_type": "earnings_forecast",
      "event_status": "disclosed",
      "title": "示例公司披露业绩预告",
      "summary": "用于说明契约结构的示例，不代表真实事件。",
      "impact": "positive",
      "strength": "medium",
      "horizon": "short",
      "announced_at": "2026-07-24T08:00:00Z",
      "known_at": "2026-07-24T08:00:00Z",
      "known_at_basis": "source_published_at",
      "source": {
        "tier": "official",
        "name": "示例交易所",
        "url": "https://example.com/announcement/001",
        "published_at": "2026-07-24T08:00:00Z",
        "confidence": "high"
      },
      "targets": [
        {
          "asset_type": "stock",
          "asset_code": "600000",
          "mapping_basis": "公告主体证券代码"
        }
      ]
    }
  ]
}
```
