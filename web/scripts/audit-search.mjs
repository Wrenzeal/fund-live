import { writeFile } from 'node:fs/promises'

const args = process.argv.slice(2)
const base = new URL(args.find(arg => /^https?:\/\//.test(arg)) || 'http://127.0.0.1:3101')
const outputIndex = args.indexOf('--output')
const output = outputIndex >= 0 ? args[outputIndex + 1] : undefined
const checks = []
const results = new Map()
const expectedCanonicalOrigin = 'https://fund.wrenzeal.top'

function check(name, passed, detail) { checks.push({ name, passed: Boolean(passed), detail }) }
function elements(html, tag) { return [...html.matchAll(new RegExp(`<${tag}\\b[^>]*>`, 'gi'))].map(match => match[0]) }
function attribute(tag, name) { return tag.match(new RegExp(`\\b${name}=["']([^"']*)["']`, 'i'))?.[1] }
function meta(html, key, value) { return elements(html, 'meta').find(tag => attribute(tag, key) === value) }

async function read(path) {
  if (results.has(path)) return results.get(path)
  try {
    const response = await fetch(new URL(path, base), { redirect: 'follow', signal: AbortSignal.timeout(25000) })
    const result = { status: response.status, headers: response.headers, html: await response.text(), url: response.url }
    results.set(path, result)
    return result
  } catch (error) {
    check(`${path} reachable`, false, error instanceof Error ? error.message : 'request failed')
    return undefined
  }
}

const robots = await read('/robots.txt')
check('robots.txt is a 200 text response', robots?.status === 200 && robots.headers.get('content-type')?.includes('text/plain'))
check('robots.txt advertises sitemap', robots?.html.includes(`${expectedCanonicalOrigin}/sitemap.xml`))
const sitemap = await read('/sitemap.xml')
check('sitemap is a 200 XML response', sitemap?.status === 200 && sitemap.headers.get('content-type')?.includes('xml'))
const urls = [...(sitemap?.html || '').matchAll(/<loc>(.*?)<\/loc>/g)].map(match => match[1])
check('sitemap contains public identity and methodology', urls.includes(`${expectedCanonicalOrigin}/about`) && urls.includes(`${expectedCanonicalOrigin}/methodology`))
check('sitemap excludes private and utility routes', urls.every(url => {
  try { return new URL(url).origin === expectedCanonicalOrigin && !/^\/(auth|holdings|watchlist|issues|analysis|api)(\/|$)/.test(new URL(url).pathname) } catch { return false }
}))

const titles = new Set()
for (const path of ['/', '/about', '/methodology', '/observations', '/research', '/privacy', '/terms']) {
  const page = await read(path)
  if (!page) continue
  const title = page.html.match(/<title>(.*?)<\/title>/s)?.[1]
  const canonical = elements(page.html, 'link').find(tag => attribute(tag, 'rel') === 'canonical')
  check(`${path} is 200 and indexable`, page.status === 200 && !page.headers.get('x-robots-tag')?.includes('noindex') && !attribute(meta(page.html, 'name', 'robots') || '', 'content')?.includes('noindex'))
  check(`${path} has a unique title and description`, title && !titles.has(title) && attribute(meta(page.html, 'name', 'description') || '', 'content'))
  if (title) titles.add(title)
  check(`${path} has production canonical`, attribute(canonical || '', 'href') === `${expectedCanonicalOrigin}${path === '/' ? '' : path}`)
  check(`${path} has an H1 in initial HTML`, /<h1\b/.test(page.html))
  check(`${path} has social image metadata`, Boolean(meta(page.html, 'property', 'og:image')) && attribute(meta(page.html, 'name', 'twitter:card') || '', 'content') === 'summary_large_image')
}

const methodology = await read('/methodology')
check('methodology includes readable calculation and primary-source links', methodology?.html.includes('预估涨跌幅 = Σ') && methodology.html.includes('https://www.cninfo.com.cn/'))
const login = await read('/auth/login')
check('login remains noindex without homepage canonical', login?.html.includes('content="noindex, nofollow"') && !elements(login.html, 'link').some(tag => attribute(tag, 'rel') === 'canonical'))
const missing = await read('/observations/not-a-fund')
check('invalid fund is an actual 404', missing?.status === 404 && missing.html.includes('noindex'))
for (const url of urls.filter(url => /\/(observations|announcements)\/[^/]+$/.test(new URL(url).pathname)).slice(0, 5)) {
  const path = new URL(url).pathname
  const detail = await read(path)
  check(`${path} advertised detail has indexable initial content`, detail?.status === 200 && /<h1\b/.test(detail.html) && !attribute(meta(detail.html, 'name', 'robots') || '', 'content')?.includes('noindex'))
}
const imageResponse = await fetch(new URL('/opengraph-image', base), { signal: AbortSignal.timeout(25000) }).catch(() => undefined)
check('share image is an actual PNG', imageResponse?.status === 200 && imageResponse.headers.get('content-type')?.includes('image/png'))

const report = { checkedAt: new Date().toISOString(), base: base.origin, checks, passed: checks.every(item => item.passed), note: 'Technical eligibility only; this does not verify search indexing, rankings, bot identity or AI citations.' }
console.log(JSON.stringify(report, null, 2))
if (output) await writeFile(output, JSON.stringify(report, null, 2) + '\n')
process.exitCode = report.passed ? 0 : 1
