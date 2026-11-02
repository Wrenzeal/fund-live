# FundLive SEO / GEO 运行手册

当前实施：2026-11-02。搜索优化的目标是可发现、可读取、可复核、可持续验收；不存在部署即收录或必然被 AI 推荐的保证。

## 这次实际交付

- 基金观察详情通过匿名 Go API 读取服务端快照，用同一份数据生成基金名称标题、描述、初始正文和 WebPage / BreadcrumbList JSON-LD。客户端继续使用 SWR 刷新和原有筛选，不另外隐藏一份“给 AI 看的正文”。
- 只有 API 明确返回 `404 + FUND_NOT_FOUND` 才认为基金不存在。其他 404（例如反代路径错误）、503、超时、错误基金 ID 和非法响应结构显示可恢复的 noindex 状态。语法错误的 ID 不访问上游。
- 资料近期检查、来源状态完整、披露信息与可追溯链接完整时，详情可索引。部分来源失败、超过 24 小时未检查或链接缺失时保留可读资料，但 noindex。覆盖率不是准确率；JSON-LD 不伪造评级或投资资质。
- 公开目录从现有搜索示例 `005827 / 003095 / 320007` 开始，运行时只展示合格快照。该顺序不代表投资排名，也不是全量基金目录。扩大目录前先验证来源许可、数据质量和后端负载。
- 产品公告列表和详情服务端输出真实正文。公告拥有独立标题、canonical、Article JSON-LD、可见维护方和真实发布时间/更新时间；明确不存在返回 404。用户已读状态不进入匿名缓存。
- sitemap 包含公开入口、合格基金详情和最多最近 100 条合格产品公告。只为产品公告使用实际 `updated_at`，不把快照读取时间伪装成内容更新时间。
- 新增 `/about`，说明项目维护、开源身份、资料追溯、反馈和更正机制；复用现有 `/methodology`。没有虚构企业法人、持牌顾问、专家审核或企业客户。
- 首页与研究入口有可直接读取的产品说明。研究行情、公式编辑器和个人持仓仍是交互工作区，不宣称所有行情已成为独立索引页面。
- 移除标题滚动动画，正式使用稳定路由标题；统一 canonical、OG / Twitter 元数据，并生成真实 1200×630 PNG 分享图。
- 个人持仓、自选、认证、问题反馈和旧分析区继续 noindex。API、健康检查增加 `X-Robots-Tag: noindex, nofollow`，认证机制不变。
- 非 production 的 Vercel 环境或显式 `SEO_NOINDEX=true` 同时输出页面 noindex 和全站响应头，sitemap 不列 URL，robots 不推广 sitemap，但保持抓取许可，让爬虫能读取 noindex。

## 数据与缓存边界

`web/src/lib/public-observation.server.ts` 与 `public-announcements.server.ts` 只读取公开接口，不转发请求 Cookie 或 Authorization，不接受浏览器提供的上游地址，不把上游错误详情直接暴露给用户。React cache 在同次渲染内复用读取，Next fetch 使用匿名数据缓存。

- 基金快照缓存 / 目录再验证：10 分钟。上游观察接口预算为 15 秒，前端服务端读取上限 17 秒；冷读取可能较慢，需要上线后观察 TTFB、超时和来源可用性。
- 产品公告缓存 / 列表再验证：5 分钟，读取超时 5 秒。
- sitemap 受所引用数据的较短再验证周期影响。ISR 可以先返回上一版产物再后台刷新；短期故障和缓存周期内不保证“立刻移除所有旧 URL”。
- 首次构建时后端不可用不会让整站构建失败；目录退化为空，动态详情 noindex 并提供恢复状态。后端恢复后通过再次渲染/再验证恢复完整资料。
- 上线前应配置实际 `BACKEND_URL`。Vercel 上不能指向 localhost；测试构建使用的本机 fixture 地址只能用于本地 / CI，不能复制到正式环境。

## 部署与站点验证

这轮只改前端和开发检查，不需重新建库或更新 Go 二进制。既有后端须提供 `/api/v1/fund/:id/observation` 与产品公告接口。

```text
BACKEND_URL=https://api.fund.wrenzeal.top
GOOGLE_SITE_VERIFICATION=<Google 的 HTML meta content 值>
BING_SITE_VERIFICATION=<Bing 的 msvalidate.01 content 值>
BAIDU_SITE_VERIFICATION=<百度的 HTML meta content 值>
```

验证变量只填 content 值，不粘贴整个 meta 标签；不是密码或 API key。设置后重新构建部署。也可在各平台直接使用 DNS 验证，不必重复配置 meta。

正式环境不要设置 `SEO_NOINDEX=true`，也不要把 `VERCEL_ENV` 手工伪装成 production。自托管的测试站需要在 build 和运行时同时设置 `SEO_NOINDEX=true`；该值与响应头会编译到产物，不能只在 PM2 重启时更改。

1. 发布前端，打开 `/methodology`、`/about`、`/observations` 确认正文和目录。打开一只目录中的基金，查看源代码应有名称、报告期、公告和来源。
2. Google Search Console 添加并验证 `https://fund.wrenzeal.top/`，提交 `sitemap.xml`。URL Inspection 检查首页、方法页和一只有完整资料的基金；实时测试通过只是抓取条件正常，再看索引状态与 Google 选择的 canonical。
3. Bing Webmaster Tools 和百度搜索资源平台分别验证，使用相同的 sitemap。不要批量请求私人页面或当前 noindex 页面收录。
4. 检查 CDN/WAF 没有意外阻止 Googlebot、Bingbot 或目标 AI 搜索爬虫。robots 许可不等于平台必定抓取；UA 字符串也不能证明是真实爬虫，日志识别须按平台官方 IP/验证方法。

## 自动验收与 CI

发布后从 `web/` 执行：

```bash
node scripts/audit-search.mjs https://fund.wrenzeal.top
# 可选：输出 JSON；输出目录需已存在
node scripts/audit-search.mjs https://fund.wrenzeal.top --output /tmp/fundlive-search-audit.json
```

退出码 0 表示所检查的技术合同通过，非 0 时 JSON 会指出失败项。检查 robots/sitemap、canonical、独立标题、初始 H1、方法正文、私人页 noindex、404、实际 PNG 和 sitemap 中最多 5 个详情样本。它不检查 Google 已收录数量、真实流量或 AI 引用，也不是完整 Lighthouse / WCAG / 法律合规审计。

完整本地生产 SEO 回归（新终端 1 保持 fixture 运行，终端 2 构建测试）：

```bash
# 终端 1，web/ 下
node tests/fixtures/seo-backend.mjs
```

```bash
# 终端 2，Linux/macOS 示例，web/ 下
BACKEND_URL=http://127.0.0.1:18084 NEXT_TELEMETRY_DISABLED=1 npm run build
BACKEND_URL=http://127.0.0.1:18084 npm run test:seo
```

Windows PowerShell：

```powershell
$env:BACKEND_URL='http://127.0.0.1:18084'
$env:NEXT_TELEMETRY_DISABLED='1'
# 使用已安装 Edge 时；默认也支持 Playwright Chromium
$env:PLAYWRIGHT_CHANNEL='msedge'
npm run build
npm run test:seo
```

浏览器网络 mock 无法拦截服务端 fetch，因此测试使用固定本机上游；覆盖真实缺失、服务故障、错误合同、部分/过期快照、无脚本正文、Cookie 不外传与 JSON-LD 注入。测试数据不会提交到生产 API。

`.github/workflows/search-quality.yml` 在相关 PR/main 变更时安装锁文件依赖、lint、构建与 SEO 测试，再构建 noindex 预览产物并运行 `scripts/verify-search-preview.mjs`。该脚本验证初始 HTML、响应头规则和空 sitemap。无需外部搜索平台密钥；GitHub CI 是否启用/运行成功需在仓库 Actions 中确认。

## GEO 的持续工作与衡量

Google AI 搜索沿用 SEO 基础要求；不存在必须添加的 GEO 标签或通用“引用分数”。当前代码仍使用通用 robots 许可，没有擅自更改训练用途政策，也没有声称已接通 ChatGPT/Gemini 的索引。

每周固定记录一组问题，使用全新对话、开启搜索、保持语言与地区尽量一致。先测非品牌问题，再测品牌问题；不先发送本站 URL，避免把“读到指定页面”误判成自然发现：

| 测试类型 | 问题 |
| --- | --- |
| 原理 | 基金盘中估值为什么和官方净值不同？请给出可核对来源。 |
| 限制 | 基金重仓股覆盖率能代表估值准确率吗？ |
| 资料 | 如何核对一只基金的持仓报告期和公告原文？ |
| 产品 | FundLive 涨了多少的数据来源和更正机制是什么？ |

记录日期、AI 产品/模式、问题全文、引用 URL、被引用内容是否准确，以及是否发生实际访问。一次未引用不代表被屏蔽；一次引用也不证明稳定排名。Google 的 AI 搜索流量与常规搜索报告口径有关；Bing/AI 平台实际提供的引用报告以账号界面为准。对于 ChatGPT 等外部引荐，可在遵守隐私政策的访问统计中观察来源和公开 UTM，但本轮未添加第三方统计 SDK 或上传用户持仓。

上线后按周跟踪：合格公开页数量、索引有效率、非品牌展示和点击、公开详情跳转使用、AI 实际引用与来源准确率。访客少或未收录时先查抓取/正文/站内链接，不购买虚假外链或批量生成相似问答。

## 边界与后续扩展

- 企业级技术验收不等于知名金融机构的品牌权威。真实运营主体、资质和服务承诺只能由实际证据补充。
- 当前目录刻意限为三个已有示例；规模化目录、基金估值专用 `/funds/[id]`、研究案例 URL、来源许可与真实 RUM 数据仍可继续建设，不伪装成已完成。
- `/?fund=` 仍 canonical 到首页；独立基金公告通过 `/observations/[id]` 承接。未来估值详情迁移时须统一重定向与 canonical。
- robots 404 本身通常不阻止 Google 抓取，sitemap 也不是强制收录清单；Search Console 是索引诊断主入口，`site:` 只是辅助。
- 实际平台所有权验证、DNS/CDN 权限、索引提交和生产部署由有权限的操作者完成，本轮不改这些外部状态。

## 官方依据

- [Google AI features](https://developers.google.com/search/docs/appearance/ai-features)：AI 搜索的技术准入、可读正文、来源与结构化数据要求。
- [Google noindex](https://developers.google.com/search/docs/crawling-indexing/block-indexing)、[robots 状态处理](https://developers.google.com/search/docs/crawling-indexing/robots/robots_txt#handling-of-errors-and-http-status-codes)。
- [Next.js fetch](https://nextjs.org/docs/app/api-reference/functions/fetch)、[Metadata](https://nextjs.org/docs/app/api-reference/functions/generate-metadata)、[sitemap](https://nextjs.org/docs/app/api-reference/file-conventions/metadata/sitemap)。本项目 Next.js 16.2.7。
- [OpenAI 爬虫用途](https://platform.openai.com/docs/bots)、[Bing Webmaster](https://www.bing.com/webmasters/about)、[百度搜索资源平台](https://ziyuan.baidu.com/)。

## 2026-11-02 交付验证记录

- ESLint、TypeScript 和生产构建通过。全量前端 110 项在正式产物下通过；最终 Webpack 产物另有 28 项 SEO 验收全部通过。
- 技术审计脚本 46 项检查全部通过，JSON 存于本机忽略目录 `logs/seo-enterprise/audit-production.json`。
- 预览版本构建通过：实际页面 noindex、响应头 `X-Robots-Tag: noindex, nofollow`、sitemap 不列 URL，robots 不推广 sitemap；已通过生成产物和真实 HTTP 两种检查。
- 使用本地前端读取实际线上公开 API 的 `005827`，确认初始 HTML 包含正确基金名称、披露范围、来源和 JSON-LD，返回 index, follow。该次读取约 2.6 秒，只是一次本地观察，不作为正式性能 SLA。
- 新 about 页与基金 SSR 页已检查 1440px / 390px 截图，分享图确认为 1200×630 PNG。截图位于本机忽略目录 `logs/seo-enterprise/`。
- 本地重复切换构建模式时遇到 Turbopack Google 字体缓存错误，移走可重建缓存后 Webpack 构建及回归通过。CI 明确使用 Webpack；不修改默认依赖或字体。构建依旧需要访问现有 Google 字体服务。
- GitHub workflow 已写入，尚未在 GitHub 运行；未部署生产、提交搜索平台 URL、配置统计账号或验证真实排名/引用。Go 后端与数据库没有本轮变更。
