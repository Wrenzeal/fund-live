# Design

## Source of truth
- Status: Active
- Last refreshed: 2026-06-04
- Primary product surfaces: Fund home `/`, analysis detail `/analysis/[fundId]`, analysis rankings `/analysis/rankings`, holdings `/holdings`, watchlist `/watchlist`, issues, and announcements.
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
  - Help users understand real-time fund valuation, holdings exposure, and quantified evidence quickly.
  - Let holding users see portfolio-level total value and cumulative return before drilling into per-fund details.
  - Make analysis pages readable in layers: conclusion first, then evidence, events, risks, and raw data.
  - Keep rule-based scores auditable and visually scannable.
  - Let administrators correct inaccurate fund classification labels without hiding the automatic holdings-derived exposure.
  - Keep login and account-protected data surfaces resistant to common brute-force, oversized payload, and cookie transport risks.
- Non-goals:
  - Do not present analysis as a direct buy/sell instruction.
  - Do not hide data limitations behind AI copy.
  - Do not add new design-system dependencies unless explicitly justified.
- Success signals:
  - Users can identify the current analysis direction, main evidence, and key risk in the first viewport.
  - Long analysis pages have clear section rhythm and no duplicated explanation blocks.
  - Mobile and laptop layouts avoid clipped cards or overly dense grids.

## Personas and jobs
- Primary personas:
  - Retail fund holders tracking daily valuation and exposure.
  - Users comparing funds through watchlist, holdings, and rankings.
  - Power users reviewing evidence behind a quantitative score.
- User jobs:
  - “What is the current structure/risk direction?”
  - “Why did the system reach this observation?”
  - “Which events or holdings are driving the score?”
  - “What data is stale, missing, or limited?”
- Key contexts of use:
  - Mobile quick checks during market hours.
  - Desktop/laptop deeper review after market close.
  - Slow or partial data states where confidence and limitations matter.

## Information architecture
- Primary navigation: Home, Analysis Rankings, Holdings, Watchlist, Issues, Announcements.
- Core routes/screens:
  - `/`: fund search, valuation, intraday chart, holdings, analysis summary.
  - `/analysis/[fundId]`: full analysis board.
  - `/analysis/rankings`: grouped quantitative ranking lists with hero overview, observation-pool stats, featured top sample, and sectioned lists.
  - `/holdings`: portfolio records, reconciliation, health, reminders.
- Content hierarchy:
  - Analysis detail page order: hero conclusion → quick insight strip → recommendation distribution/modules → primary and counter evidence → event signal chain → risk breakdown → structure/quarterly change → valuation → holding classification → data context → holdings → method limitations.
  - Rankings page order: hero overview → observation-pool stats → featured top sample → sectioned ranking lists → method note.
  - Holding classification card order: effective category/manual tags → manual correction note → automatic sector/theme exposure modules → confidence hints.
  - Holdings overview order: principal → total value → total return → today P/L → today change; total return must state whether it is official-ready or intraday-confirmed scope.
  - Holdings page workspace order: global header/quick actions → one active workspace tab at a time. Tabs are Summary, Record, Holdings, Risk, Ledger, Tools; do not show all long modules simultaneously.

## Design principles
- Principle 1: Conclusion first, evidence second, raw data last.
- Principle 2: Compress repeated explanations into one explicit method/limitations zone.
- Principle 3: Prefer progressive disclosure and scroll reveal over stacking many equal-weight cards.
- Holdings pages should make each block's job explicit in the navigation label and keep primary actions inside the block that owns them.
- Analysis detail should stack event signal chain above risk breakdown; do not render them as side-by-side panels.
- Analysis detail should stack valuation, holding classification, and data context vertically rather than compressing them into one row.
- Rankings should keep the three observation-pool stat cards (积极池 / 观察池 / 风险池) side-by-side on wider screens, while stacking the “结构偏积极 / 最值得观察 / 高风险关注” sections vertically because those are text-heavy decision modules.
- Tradeoffs:
  - Visual motion is useful for comprehension, but must remain lightweight and not require extra dependencies.
  - Dense financial data is necessary, but each section needs a clear job and non-overlapping copy.
  - Overlay menus and dropdowns must use opaque surfaces, not translucent glass, when page content can sit directly behind the menu; if a menu can overlap sibling buttons or search results would otherwise expand a management card, render it in a high-level portal/fixed layer to prevent click-through and layout jumps.
  - Watchlist group sorting should start only from the explicit drag handle; do not make the entire group card draggable because it can steal clicks from edit/delete/collapse actions.
- Production access must keep HTTPS as the public entry: HTTP redirects to HTTPS, page traffic goes to the Next.js frontend, and `/api/` plus `/health` route directly to the Go backend.
- Authentication UX should explain password requirements before submission; backend remains the source of truth and must return generic login failures plus explicit rate-limit feedback.

## Visual language
- Color: Theme-variable based dark/classic/cyber system; cyan for neutral/data, rose for positive A-share style, emerald for defensive/negative or risk-down contexts, amber for warnings/limitations.
- Typography: Strong section titles, compact metadata labels, readable Chinese body copy at 12–14px+ depending on density.
- Spacing/layout rhythm: Max width around `max-w-7xl`; use 5–6 unit vertical section rhythm; avoid too many four-column grids on content-heavy panels.
- Shape/radius/elevation: Rounded cards (`rounded-2xl` / `rounded-3xl`) with glass surfaces; avoid stacking heavy shadows on every nested card.
- Motion: Native CSS transitions and IntersectionObserver scroll reveal; major long pages should use shared `ScrollReveal` / `ScrollRevealStack`; respect `motion-reduce` utilities.
- Imagery/iconography: Lucide icons for section identity; icons should clarify semantics, not decorate every line. Browser favicon should use the blue line-only activity mark on transparent background, with SVG preferred and ICO as fallback. Browser tab title should read `FundLive - 你的基金估值系统`; if it is too long, use a lightweight document-title ticker that respects reduced motion, avoids low-frame jumps with sub-second character steps, and pauses briefly when the full title is visible. The visible home wordmark stays “涨了多少”, and its subtitle should read `FundLive - 实时基金估值`.

## Components
- Existing components to reuse:
  - `AnimatedScoreGauge`
  - `EstimateCard`
  - `FundSectorCard`
  - `HoldingsTable`
  - `TargetETFHoldingsCard`
  - `FundAnalysisBadge`
  - `FundAnalysisEventHint`
- New/changed components:
  - Shared brand helper: `BrandMark` in `web/src/components/brand-mark.tsx`; reuse it for top-left brand blocks instead of duplicating logo icon, wordmark, and subtitle JSX.
  - Shared motion helpers: `ScrollReveal`, `ScrollRevealStack`, `useLazyReveal` in `web/src/components/scroll-reveal.tsx`.
  - Shared analysis helpers: `AnalysisReveal`, `AnalysisSectionHeading`, `useLazyReveal` in `web/src/components/analysis-layout.tsx` now delegate reveal behavior to `scroll-reveal.tsx`.
  - `FundSectorCard` now consumes `classification_override`, displays manual correction chips/tags, and exposes an administrator-only inline editor for category, sector, theme, tags, and note.
  - Analysis detail local helpers: `InsightStrip`, `QuickStat`, `DataContextPanel`, `MethodCompactCard`.
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
  - Desktop: two-column analysis sections with no hard widths that can overflow.
  - Wide screens: reserve multi-column grids for compact cards, not text-heavy evidence blocks.
- Touch/hover differences: Hover lift is decorative only; core state must be visible without hover.

## Interaction states
- Loading: Keep loading text or existing spinners; analysis can render dashboard data independently.
- Empty: Empty panels should state what is missing without implying failure.
- Error: Existing SWR error surfaces should remain visible on pages that expose them.
- Success: Successful analysis should show direction, score, evidence, and limitations.
- Disabled: Not applicable for the analysis detail page in this pass.
- Offline/slow network: Avoid blocking the whole page when only analysis is still loading.

## Content voice
- Tone: Clear, conservative, explanatory.
- Terminology: Use “结构偏积极 / 适合观察 / 风险偏高” rather than direct “加仓 / 减仓” instructions.
- Microcopy rules:
  - Distinguish rule-based observations from trade advice.
  - State data limitations once in the method/limitations section.
  - Avoid repeated “当前看板负责串起来” style scaffolding.

## Implementation constraints
- Framework/styling system: Next.js App Router, React 19, TypeScript, Tailwind CSS v4, theme CSS variables.
- Design-token constraints: Reuse existing `glass`, `text-theme-*`, `var(--card-*)`, and accent utility patterns.
- Performance constraints: No new frontend dependency for motion; use IntersectionObserver and CSS transitions.
- Compatibility constraints: Keep backend API payloads and analysis semantics unchanged unless separately planned.
- Deployment constraints: production frontend runtime must set `BACKEND_URL` to the actual backend listener (`http://127.0.0.1:13896` in the current server), and deployment health checks must target the same backend port.
- Classification constraints: Manual classification labels are an overlay; never present manual tags as automatic holdings weights, and keep automatic sector/theme breakdown visible.
- Test/screenshot expectations: At minimum run `npm run lint` and `npm run build`; for high-risk visual changes, capture browser screenshots or run visual QA when available.

## Open questions
- [ ] Whether `RevealSection` and `SectionHeading` should be promoted from route-local helpers to shared components after reuse appears on two or more pages / owner: frontend / impact: component governance.
