# Design

## Source of truth
- Status: Active
- Last refreshed: 2026-11-03
- Primary product surfaces: Fund home `/`, fund observations `/observations` and `/observations/[fundId]`, holdings `/holdings`, watchlist `/watchlist`, market research `/research`, issues, and product announcements.
- Current scope: `.omx/plans/fundlive-2-implementation.md`, the user-approved paper-white/pine system and fund holding structure. Old fund analysis detail/rankings URLs redirect to observations; scoring and Lean remain compatibility/research internals, not production decision UI.
- Evidence reviewed:
  - `web/src/app/analysis/[fundId]/page-client.tsx`
  - `web/src/components/fund-analysis-card.tsx`
  - `web/src/app/analysis/rankings/page.tsx`
  - `web/src/components/fund-sector-card.tsx`
  - `web/src/hooks/use-fund-data.ts`
  - `internal/handler/fund_handler.go`
  - `internal/service/fund_sector_store.go`
  - `web/src/app/globals.css`
  - `README.md`, `CHANGELOG.md`, `todo_list.md`, `.omx/context/current-state.md`, `.omx/context/features.md`, `.omx/context/conventions.md`

## Brand
- Personality: Clear, data-heavy, trustworthy, modern financial cockpit; energetic but not speculative.
- Trust signals: Evidence-first summaries, visible data freshness, source/coverage limitations, conservative wording for quantitative observations.
- Avoid: Repeated “AI demo” gradients, duplicated explanation blocks, investment-command wording, dense long-text walls, layout grids that overflow on medium screens.

## Product goals
- Goals:
  - Help users understand real-time valuation, disclosed holdings exposure, dated source documents, and data gaps quickly.
  - Let holding users see current held-fund market value and holding P/L on the same priced positions before drilling into per-fund details.
  - Give fund observations a distinct role: source announcements and disclosure availability, with links to existing fund valuation and research surfaces.
  - Display facts and source dates without unvalidated scores, probabilities, sentiment or fund investment rankings.
  - Let administrators correct inaccurate fund classification labels without hiding the automatic holdings-derived exposure.
  - Keep login and account-protected data surfaces resistant to common brute-force, oversized payload, and cookie transport risks.
- Non-goals:
  - Do not present analysis as a direct buy/sell instruction.
  - Do not hide data limitations behind AI copy.
  - Do not add new design-system dependencies unless explicitly justified.
- Success signals:
  - Users can identify the selected fund, disclosure date, coverage and source availability in the first viewport.
  - Observations do not repeat the home page's charts, portfolio accounting or research formulas.
  - Mobile and laptop layouts avoid clipped cards or overly dense grids.

## Personas and jobs
- Primary personas:
  - Retail fund holders tracking daily valuation and exposure.
  - Users following selected funds through watchlist, holdings and source documents.
  - Power users reviewing market research separately from their fund disclosures.
- User jobs:
  - “Which announcements were published for this fund and its disclosed holdings?”
  - “Where is the original document and when was it published?”
  - “How much of the fund's disclosed portfolio is represented?”
  - “What data is stale, missing, or limited?”
- Key contexts of use:
  - Mobile quick checks during market hours.
  - Desktop/laptop deeper review after market close.
  - Slow or partial data states where confidence and limitations matter.

## Information architecture
- Primary navigation: Home, Watchlist, Holdings, Observations, Research; Issues and product Announcements are secondary destinations. Keep one restrained text-navigation group with a short bottom indicator for the active route.
- Core routes/screens:
  - `/`: fund search, valuation, intraday and official NAV charts, disclosed holdings and exposure, compact link to fund announcements.
  - `/observations`: shared fund search and recently viewed fund entry points; no ranking or separate watchlist management.
  - `/observations/[fundId]`: disclosure context, source availability, dated deduplicated announcement list and links to existing fund detail/research.
  - `/analysis/[fundId]`, `/analysis/rankings`: compatibility redirects to the corresponding observation route.
  - `/research`: market price charts, technical indicators and formula experiments; these are not repeated in observations.
  - `/holdings`: portfolio records, reconciliation, health, reminders.
- Content hierarchy:
  - Observation detail order: fund identity and raw disclosure metadata → source status → notice filters and dated source list → concise method boundary.
  - Observation entry order: concise module purpose and shared search → recently viewed funds → existing valuation/research links.
  - Fund holding structure: fund type remains beside fund identity. Industry rows show raw returned weights on a fixed 0–100% scale; related themes and manual annotations are separate keyboard-accessible views. Keep each snapshot date/source and qualitative mapping coverage beside its data; never infer remainder from Top 3.
  - Holdings overview order: current market value → holding P/L and dated current-position change → remaining cost. P/L compares only positions priced under the chosen scope and is not lifetime account profit.
  - Holdings page: summary and grouped fund rows share the default view. Current market value leads, followed by holding P/L and dated daily movement; remaining cost is secondary. One primary “记一笔” opens the buy form. Ledger, review/reminders and import/reconciliation stay secondary. Expand fund details for correction, sale, dividends and share adjustment.

## Design principles
- Principle 1: Facts, source dates and explicit data gaps first; no inferred investment conclusion.
- Principle 2: Compress repeated explanations into one explicit method/limitations zone.
- Principle 3: Use progressive disclosure instead of equal-weight nested cards. Product data is visible immediately; do not hide money or classification behind scroll-entry choreography.
- Holdings pages should make each block's job explicit in the navigation label and keep primary actions inside the block that owns them.
- Observations do not render scores, gauges, recommendation distributions, shadow score comparisons, sentiment labels or investment ranking pools.
- Date-only publication fields and disclosure periods must be explicit; a fetch timestamp cannot substitute for a source's publication date.
- Empty successful notice results and unavailable/partial sources must have distinct states. Do not infer that there are no announcements from a failed fetch.
- Tradeoffs:
  - Visual motion is useful for comprehension, but must remain lightweight and not require extra dependencies.
  - Dense financial data is necessary, but each section needs a clear job and non-overlapping copy.
  - Overlay menus and dropdowns must use opaque surfaces, not translucent glass, when page content can sit directly behind the menu; if a menu can overlap sibling buttons or search results would otherwise expand a management card, render it in a high-level portal/fixed layer to prevent click-through and layout jumps.
  - Watchlist group sorting should start only from the explicit drag handle; do not make the entire group card draggable because it can steal clicks from edit/delete/collapse actions.
- Production access must keep HTTPS as the public entry: HTTP redirects to HTTPS, page traffic goes to the Next.js frontend, and `/api/` plus `/health` route directly to the Go backend.
- Authentication UX should explain password requirements before submission; backend remains the source of truth and must return generic login failures plus explicit rate-limit feedback.

## Visual language
- Color: Paper-white/pine. classic: canvas #F3F5F2, panel #FFFFFF, ink #202B26, accent #285D49, secondary #58665E. dark: canvas #131A17, panel #1B2420, accent #9CC4AF. cyber keeps the existing preference ID as a higher-contrast pine-dark variant. Red-up/green-down remain independent from brand color; warnings use an amber semantic token. Primary actions pair accent with --on-accent.
- Typography: Existing Geist / Noto Sans SC sans stack, restrained semibold titles, 14–16px operational copy, 12px provenance, 36–48px primary financial values and tabular numbers. Supporting text has explicit contrast tokens; large type does not make all labels bold.
- Spacing/layout rhythm: Max width around `max-w-7xl`; use 5–6 unit vertical section rhythm; avoid too many four-column grids on content-heavy panels.
- Shape/radius/elevation: 8px shared buttons, 12px default Surface panels, restrained 16px larger surfaces. Opaque surfaces and one subtle border establish hierarchy; elevation is reserved for popovers. Legacy glass call sites now render flat panels, without blur, gradients or glow.
- Motion: Keep functional transitions for menus, loading, submission and states. ScrollReveal remains a compatibility layout wrapper and does not animate product-data entry; research chart lazy activation is separate. Respect reduced motion, and verify theme contrast after finite color transitions settle.
- Imagery/iconography: Lucide icons for section identity; icons should clarify semantics, not decorate every line. Keep the activity icon and brand name, using an outlined/tinted mark without the former glow or gradient wordmark. Browser titles use stable, route-specific metadata; do not animate or rewrite document.title after hydration. Social previews use the generated 1200×630 FundLive image and shared OG/Twitter metadata.

## Components
- Existing components to reuse:
  - `FundSearch`, `AppTopBar`, `SiteFooter`, `Disclosure`
  - `EstimateCard`
  - `FundSectorCard`
  - `HoldingsTable`
  - `TargetETFHoldingsCard`
- New/changed components:
  - Shared brand helper: `BrandMark` in `web/src/components/brand-mark.tsx`; reuse it for top-left brand blocks instead of duplicating logo icon, wordmark, and subtitle JSX.
  - Shared motion helpers: `ScrollReveal`, `ScrollRevealStack`, `useLazyReveal` in `web/src/components/scroll-reveal.tsx`.
  - Shared analysis helpers: `AnalysisReveal`, `AnalysisSectionHeading`, `useLazyReveal` in `web/src/components/analysis-layout.tsx` now delegate reveal behavior to `scroll-reveal.tsx`.
  - `FundSectorCard` now consumes `classification_override`, displays manual correction chips/tags, and exposes an administrator-only inline editor for category, sector, theme, tags, and note.
  - Observations use the dedicated observation response; do not rebuild them from production score/evidence snapshots.
- Variants and states:
  - Loading: preserve skeleton/empty states and non-blocking placeholders.
  - Empty: show concise empty panels; avoid repeating method copy.
  - Warning/limitations: amber tone and explicit text.
- Token/component ownership: Keep route-local helper components until reuse across pages is proven; do not introduce a new component library layer.

## Auth and security UX
- Login defaults to a two-step email-code flow when `/api/v1/auth/config` reports it available; password remains an explicit tab and Google remains a separate provider action.
- The code stage must show the destination email, resend countdown, change-email action, six-digit numeric input, one-time-code autocomplete, and a clear development-only code banner when returned by the API.
- When DragonFly or SMTP is unavailable, switch to password login with a concise temporary notice; do not render a dead primary action.
- Authentication redirects may preserve only validated same-origin relative paths. Reject protocol URLs, protocol-relative paths, backslashes, and control characters.
- Register password copy must match backend policy: at least 10 characters, include letters and numbers, and contain no whitespace.
- Login failures should not disclose whether an email format, account existence, or password check failed; use generic invalid credentials in UI.
- Rate-limited authentication should surface as a temporary wait/retry state, not as a permanent account error.
- Production session cookies must be Secure + HttpOnly + SameSite=Lax under HTTPS; local HTTP can keep Secure disabled only for development.

## Accessibility
- Target standard: Practical WCAG AA intent for contrast and semantic readability.
- Keyboard/focus behavior: Links and controls must retain visible focus from browser/Tailwind defaults or component styles.
- Contrast/readability: Theme variables must remain legible in classic/dark/cyber; avoid pale text on intense gradients.
- Screen-reader semantics: Use section headings and meaningful link text; charts should include aria labels when SVG conveys data.
- Reduced motion and sensory considerations: Use `motion-reduce` classes for scroll reveal and avoid mandatory animation for understanding.

## Responsive behavior
- Supported breakpoints/devices: Mobile, tablet, desktop/laptop.
- Layout adaptations:
  - Mobile: single-column sections; quick stats become two-column chips where space allows.
  - Primary navigation: render inline in the main toolbar at `xl` and above; below `xl`, keep all five destinations visible in an equal-width second row without horizontal scrolling.
  - Desktop: two-column analysis sections with no hard widths that can overflow.
  - Wide screens: reserve multi-column grids for compact cards, not text-heavy evidence blocks.
- Touch/hover differences: Hover lift is decorative only; core state must be visible without hover.

## Interaction states
- Loading: Independent notice loading and source states; never keep fund A's details visible under fund B's identity.
- Empty: Empty panels should state what is missing without implying failure.
- Error: Existing SWR error surfaces should remain visible on pages that expose them.
- Success: Successful observations show source documents, publication dates and coverage limitations.
- Disabled: A missing upstream source does not disable available sources or existing valuation/research routes.
- Offline/slow network: Show retry and explicit partial results instead of presenting failure as an empty successful feed.

## Content voice
- Tone: Clear, conservative, direct; explain only when the user needs a decision or a boundary.
- Terminology: Use “基金观察 / 公告与资料 / 已披露持仓 / 来源暂不可用”. Avoid directional investment labels in the production observation module.
- Microcopy rules:
  - Distinguish rule-based observations from trade advice.
  - State data limitations once in the method/limitations section.
  - Prefer Chinese UI copy; keep only brand names and established technical terms in English.
  - Use natural-language loading/error states with a next action; keep diagnostic codes out of the user-facing surface.
  - Use progressive disclosure for method, source, and risk details instead of repeating explanations beside every metric.
  - Avoid repeated “当前看板负责串起来” style scaffolding.

## Implementation constraints
- Framework/styling system: Next.js App Router, React 19, TypeScript, Tailwind CSS v4, theme CSS variables.
- Design-token constraints: Use Surface/ActionButton and --surface-subtle / --card-bg / --card-border / --on-accent. text-accent, text-warning, text-danger, text-success and their soft backgrounds derive from semantic theme tokens. Avoid hardcoded palette utilities and new theme-specific overrides; retained glass/text-theme-* names are compatibility APIs.
- Performance constraints: No new frontend dependency for motion; use IntersectionObserver and CSS transitions.
- Compatibility constraints: Existing analysis/research backend APIs and database data remain intact. The new observation endpoint does not invoke score calculation; old public detail/rankings links redirect.
- Deployment constraints: production frontend runtime must set `BACKEND_URL` to the actual backend listener (`http://127.0.0.1:13896` in the current server), and deployment health checks must target the same backend port.
- Classification constraints: Snapshot primary labels may already have administrator overrides; derive automatic labels from unchanged breakdown entries. Manual annotations are a separate view and preserve edit permissions/update payloads. Theme weights use the existing mapped split, not official theme allocation. Target ETF provenance is explicit; never infer total holdings coverage from returned rows.
- Test/screenshot expectations: At minimum run `npm run lint` and `npm run build`; for high-risk visual changes, capture browser screenshots or run visual QA when available.

## Search and AI discovery
- Public search metadata uses the production domain, stable distinct route titles and canonical URLs. The previous home title ticker is removed because search identity now takes priority.
- `/methodology` is a server-rendered explanation of estimation, source dates, coverage and limitations, reachable from the footer. Reuse the static information-page components and existing theme tokens.
- Private account and utility areas remain noindex. Fund observation and product announcements reuse anonymous validated server data for metadata and visible initial HTML; only source-complete, fresh observation snapshots open indexing. A bounded public directory exposes existing sample funds, not rankings. Partial, unavailable or wrong-entity responses remain recoverable/noindex; only confirmed absence is 404.
- `/about` documents the real open-source project, maintenance and correction process without invented corporate credentials. Match schema to visible body, maintain source dates, and escape JSON-LD.
- Non-production Vercel environments and explicit `SEO_NOINDEX=true` remain noindex in page metadata and response headers, without blocking bots from reading that directive. User cookies are never forwarded to anonymous snapshot caches.
- `robots.txt` and sitemap are shared discovery infrastructure; platform ownership verification and index reports remain separate. See `docs/seo-geo.md`. Neither structured data nor an AI-specific file guarantees citations or financial authority.

## Open questions
- Holdings refactor (2026-11-02): `docs/holdings-ux-redesign.md`. Current held-fund market value is the primary amount. Keep official and preview modes explicit; missing data stays unavailable. Partial coverage must use matching priced cost for P/L. Never label cost reduction as redemption proceeds or current holding P/L as lifetime profit. Preserve the three themes, 44px controls, and a first-record flow without demo writes.
- [ ] Whether strict empirical validation will justify any future scoring product. No unvalidated score may be reintroduced through observation or portfolio labels.


## FundLive 2.0 delivery evidence — 2026-11-03
- Direction approved from docs/design/fundlive-2-concept.html A, adapted to the real product navigation and existing API data rather than copying demo-only routes or content.
- Shared palette, typography, buttons, Surface, brand and floating surfaces apply across home, holdings, watchlist, observations, research, auth and existing utility screens.
- Source cleanup removes the prior theme-override/glow/gradient layer. Functional research chart colors and financial calculations are preserved.
- Local browser fixtures cover original industry weights, ETF denominator, separate dates, keyboard views, empty/error states, administrator save and failure drafts. Existing auction, accounting, auth and SEO regressions remain required.
- Readability checks measure rendered primary-action/supporting text in all three themes, and classification labels/metadata. This is bounded evidence, not a claim of a complete WCAG audit.
- Screenshot captures are test fixtures, never real user portfolios. They are stored under ignored logs/test-results; no generated raster ships with the interface.
