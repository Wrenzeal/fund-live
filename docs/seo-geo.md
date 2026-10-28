# FundLive 搜索发现与 AI 引用

实施日期：2026-10-28。代码改动部署到前端后才会对线上生效；生成 sitemap、通过测试或提交 URL 都不等于已经收录，更不保证排名或 AI 引用。

## 共用基础与平台设置

搜索引擎共用同一套 HTML、canonical、标题、正文、来源和 sitemap，不为不同爬虫输出不同内容。Google 的 AI Overviews / AI Mode 沿用搜索基础要求，不需要专门的 GEO 标签或 `llms.txt`。这项结论不能外推为所有 AI 产品都使用同一爬虫或同一索引。

- Google Search Console：验证 `fund.wrenzeal.top` 的 URL 前缀资源或域名资源，提交 `https://fund.wrenzeal.top/sitemap.xml`，检查公开页的 URL Inspection 和索引报告。
- Bing Webmaster Tools：单独验证站点，提交同一个 sitemap，检查抓取与索引；后续有内容发布事件时再评估 IndexNow，不为行情自动刷新反复提交通知。
- 百度搜索资源平台：面向中文用户时单独添加、验证站点，并使用该账号实际可用的链接提交和诊断功能。不同账号可用功能可能不同。
- AI 搜索：按目标平台的官方爬虫说明单独核对搜索抓取权限。搜索抓取和模型训练用途分开决策；未配置的平台不宣称已经接入。

缺少 robots.txt 或 sitemap 本身不代表不能被 Google 收录；robots 404 通常不施加抓取限制。Google 的 `site:` 查询不是权威收录统计，实际诊断以 Search Console 为主。

## 本轮代码

- `web/src/app/robots.ts`：统一允许抓取并发布 sitemap 地址。没有把 noindex 页面同时写成 Disallow，避免爬虫看不到页面的 noindex。API 与个人数据仍依靠后端认证，robots 和 noindex 都不是隐私防护。
- `web/src/app/sitemap.ts`：只列首页、观察入口、研究入口、方法页、产品公告入口和政策页。未伪造 `lastModified`、基金清单或更新频率。
- `web/src/lib/site-metadata.ts`：集中正式域名与公开页 OpenGraph 元数据，防止内页丢失站点身份。
- 公开栏目拥有独立标题、描述、canonical；页脚可到达 `/methodology`。
- 登录、自选、持仓、问题反馈和旧分析区域为 noindex。基金观察详情和产品公告详情暂时 noindex, follow，因为实体和主体正文仍由客户端请求确认。语法错误的基金 ID 返回 404；六位但不存在的代码尚不能仅凭语法识别，不伪装成已验证实体。
- 根布局提供真实 WebSite 身份信息，不声明不存在的金融资质、评分、评价或收益。方法页正文在服务端输出，说明普通持仓的归一化算法、覆盖率、目标 ETF/海外基金、来源及限制。
- 首页原有标题动画保留，但路由切换后停止，不覆盖内页标题。首页动画仍可能被渲染型爬虫看到，属于后续可去除的展示取舍。

## 站点所有权验证

优先使用平台提供的 DNS 验证；若选择 HTML meta 验证，可以在前端部署环境添加：

```text
GOOGLE_SITE_VERIFICATION=<Google 提供的 content 值>
BING_SITE_VERIFICATION=<Bing 提供的 msvalidate.01 content 值>
BAIDU_SITE_VERIFICATION=<百度提供的 content 值>
```

只填写 `content` 属性值，不粘贴整个 meta 标签；无需 `NEXT_PUBLIC_` 前缀。这些值用于公开所有权证明，不是账号密码，也不是 API key。未提供时不会生成占位标签。静态页面需要重新 build / deploy 才能体现新的验证值。不要把 Search Console OAuth 凭据、统计服务密钥或服务器密码放入这些变量。

接入平台账号、DNS 修改和线上部署尚未执行。本轮不修改生产数据库、Nginx 或 Go 后端；只需发布前端。

## 发布后验收

```bash
curl -i https://fund.wrenzeal.top/robots.txt
curl -i https://fund.wrenzeal.top/sitemap.xml
curl -fsS https://fund.wrenzeal.top/methodology
curl -fsS https://fund.wrenzeal.top/auth/login
```

验收：前两项分别为 200 文本与 200 XML；方法页初始 HTML 包含公式和来源；登录页 robots 为 noindex 且不带首页 canonical。页面 noindex 不能用于保护未认证可访问的真实个人数据。再检查 CDN、Vercel / Nginx 是否意外为正式站添加 noindex 或阻挡爬虫，预览环境则应保持不收录。

本地回归入口：`web/tests/e2e/seo-foundation.spec.ts`。它覆盖真实响应、无 JavaScript 的公开元数据、私有页边界、方法页正文和首页动画离开路由后的标题稳定性。

### 2026-10-28 本地验收

- ESLint、TypeScript 和 Next.js 16.2.7 Turbopack 生产构建通过；没有新增运行时或开发依赖。
- 开发服务器上 SEO、登录导航与基金观察合计 39 项回归通过；正式构建启动后，16 项 SEO 验收再次全部通过。使用已安装的 Microsoft Edge，由本地临时测试配置指定浏览器，不修改项目默认 Playwright 配置。
- 验证了 robots/sitemap 的真实 200 响应、公开栏目 HTML 元数据、个人区域 noindex、方法页无 JavaScript 正文、非法基金 ID 的 404 与恢复入口。异步 404 的提示 UI 可能在 hydration 后显示，禁用 JavaScript 时仍返回 404 和 noindex，不携带错误 canonical。
- 方法页 1440px、360px 截图均无文档横向溢出；截图保存在忽略目录 `logs/seo-qa/`。
- 本地生产验收不代表线上已经部署或被搜索平台收录。Go 测试未重跑，因为本轮未修改后端；数据库和外部平台账号未变动。

## 下一阶段：让正文可被发现和引用

1. 服务端读取并缓存匿名基金观察快照，同一次读取生成基金名称 metadata 和初始正文，再交给客户端刷新。只对已确认不存在的基金返回 404；上游超时与基金不存在分开处理。
2. 建立包含真实可用基金的公开目录和站内链接。分批开放有完整正文、稳定 canonical、来源/报告期/行情时间的基金详情，再纳入动态 sitemap；不批量生成只有代码的空页。
3. 首页 `/?fund=` 目前 canonical 收敛到首页，不能作为每只基金的独立 SEO 入口。若新增 `/funds/[fundId]`，保留旧链接并用明确重定向收口，避免所有详情都 canonical 到首页。
4. 研究工具当前的选中标的、公式与区间主要是客户端状态。先提供可复现且有稳定 URL 的公开说明/案例，不把私人本地方案或未经验证的历史胜率包装成未来收益。
5. 公开作者/维护方、更新与更正机制、计算方法和原始来源。日期来自实际资料更新，不在每次请求时假装内容有新变化。
6. 观察搜索展示/点击、有效公开页索引、搜索后使用观察工具的行为；AI 引荐流量与实际引用记录作为辅助指标。没有数据时不报告“GEO 分数”或提升百分比。

## 官方依据

- Google AI 搜索：[AI features and your website](https://developers.google.com/search/docs/appearance/ai-features)。Google 的 AI 搜索没有额外技术准入标签，正文应可读、内部链接可发现，结构化数据应与可见内容一致。
- Google 抓取：[robots.txt status handling](https://developers.google.com/search/docs/crawling-indexing/robots/robots_txt#handling-of-errors-and-http-status-codes)。区分 robots 404、403、429、5xx 的行为。
- Google noindex：[Block Search indexing](https://developers.google.com/search/docs/crawling-indexing/block-indexing)。爬虫必须能访问页面才能读取 noindex。
- Next.js：[robots](https://nextjs.org/docs/app/api-reference/file-conventions/metadata/robots)、[sitemap](https://nextjs.org/docs/app/api-reference/file-conventions/metadata/sitemap)、[metadata](https://nextjs.org/docs/app/api-reference/functions/generate-metadata)。本项目安装版本为 Next.js 16.2.7。
- Bing：[Webmaster Tools](https://www.bing.com/webmasters/about)、[IndexNow](https://www.indexnow.org/documentation)。
- 百度：[搜索资源平台](https://ziyuan.baidu.com/)。

本说明不承诺任何平台的收录、排名或引用。金融内容的可信度靠可复核资料与方法，不靠免责声明或 JSON-LD 自动获得。
