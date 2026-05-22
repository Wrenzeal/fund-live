# QDII / 海外行情数据源选型建议

更新时间：2026-05-18

## 结论

- **本阶段不接 VIP / 付费官方 API**：先把现有 QDII 固定海外行情源做成可配置边界，默认仍使用 `tencent`，保证现有行为不变。
- **短期生产策略**：保留 `tencent` / `sina` 两个既有公开行情适配器，用 `quote.overseas_source` 或 `QUOTE_OVERSEAS_SOURCE` 切换；前端和估值链路继续把海外估值标记为 `overseas_fixed`，避免被用户级国内行情源偏好影响。
- **中期正式接入首选**：优先评估 Polygon / Massive，原因是其股票 WebSocket 覆盖分钟聚合、逐笔成交、NBBO quote 等实时数据形态，最贴近 QDII 分时图和实时估值需求。
- **中期备选**：Alpaca Market Data、Twelve Data。Alpaca 适合后续扩展券商 / Broker 生态；Twelve Data 适合快速接多市场统一格式。
- **长期企业级备选**：Intrinio，适合更明确的采购、授权和企业合规流程，但接入与成本预期更高。

## 当前代码边界

当前 FundLive 的海外估值链路已经和用户级 `sina` / `tencent` 偏好隔离：

- 国内默认源：`quote.default_source` / `QUOTE_DEFAULT_SOURCE`，用于未登录用户与用户未设置偏好时的国内行情。
- 海外固定源：`quote.overseas_source` / `QUOTE_OVERSEAS_SOURCE`，用于 QDII / 海外持仓估值。
- 现阶段支持值：`tencent`、`sina`。
- 默认值：`tencent`，用于保持改造前行为。

示例：

```yaml
quote:
  default_source: sina
  overseas_source: tencent
```

如需临时切换海外固定源进行对照：

```bash
QUOTE_OVERSEAS_SOURCE=sina go run ./cmd/server
```

## 数据源对比

| 数据源 | 适合阶段 | REST / 历史分时 | WebSocket / 实时 | 授权与限制关注点 | 当前建议 |
|---|---|---|---|---|---|
| 现有 `tencent` / `sina` 固定源 | 短期过渡 | 可满足部分实时估值，不适合作为正式 SLA | 无公开稳定授权边界 | 公开网页接口稳定性、授权和再分发边界不清晰 | 只作为短期过渡，保持可切换与可降级 |
| Polygon / Massive | 中期首选 | 有股票历史与聚合数据，适合分时图 | 股票 WebSocket 覆盖分钟聚合、逐笔成交、NBBO quote 等 | 实时数据通常涉及付费计划和市场数据条款 | 首先评估，用于正式 QDII 实时估值 + 分时图 |
| Alpaca Market Data | 中期备选 | HTTP + WebSocket，支持 IEX / SIP / delayed SIP 等 feed | 股票实时 WebSocket 清晰 | 免费层 live IEX 与 SIP 权限差异明显；近期 SIP 需要订阅 | 若后续接券商 / Broker 生态可优先评估 |
| Twelve Data | 轻量备选 | 支持股票、ETF 等多资产，示例包含 1min time_series | 官方页面说明支持 WebSocket 低延迟流 | 需逐市场核对实时 / 延迟与额度限制 | 快速多市场统一格式备选 |
| Intrinio | 企业级备选 | Realtime quote API 字段完整 | 有实时数据产品体系 | 实时数据需要相应产品订阅，采购链路偏企业级 | 正式企业采购或合规授权要求更高时评估 |

## 正式 provider 接入前的验收问题

1. **实时性**：是否真能返回美股盘前、盘中、盘后 quote / trade / aggregate，且延迟口径明确。
2. **分时能力**：是否支持至少 1 分钟级 OHLCV，能支撑 QDII 基金详情分时图。
3. **授权边界**：是否允许在 FundLive 页面展示、缓存和对用户再分发；是否限制非专业 / 专业用户。
4. **覆盖范围**：是否覆盖美股普通股、ETF、ADR、港股或未来可能解析到的海外市场。
5. **批量与限流**：是否支持一次批量取多个持仓 ticker；限流是否满足持仓页、详情页和后台快照任务。
6. **降级策略**：正式源失败时是否允许回退到当前固定源，页面是否能清晰提示数据来源和可信度。
7. **成本**：是否随 ticker 数、请求数、WebSocket 连接数、用户再分发或专业用户身份计费。

## 本轮不做的事项

- 不接 VIP 权益、额度、报告或支付链路。
- 不引入新的行情 SDK 或第三方依赖。
- 不存储官方数据源密钥。
- 不把 AI / 大模型链路用于修复行情数据可信度。

## 参考官方资料

- Polygon / Massive Stocks WebSocket overview: https://www.polygon.io/docs/websocket/stocks/overview
- Polygon / Massive pricing: https://polygon.io/pricing
- Alpaca real-time stock data: https://docs.alpaca.markets/docs/real-time-stock-pricing-data
- Alpaca market data FAQ: https://docs.alpaca.markets/v1.4.2/docs/market-data-faq
- Twelve Data stocks data API: https://twelvedata.com/stocks
- Twelve Data API overview: https://twelvedata.com/docs/advanced/api-usage
- Intrinio realtime quote prices by exchange: https://docs.intrinio.com/documentation/web_api/get_stock_exchange_quote_v2
