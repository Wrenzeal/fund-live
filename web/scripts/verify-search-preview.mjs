import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import path from 'node:path'

// Run after a preview/noindex build. Verify actual generated artifacts and
// Next response-header rules without starting a second build in the same tree.
const root = process.cwd()
for (const route of ['index', 'about', 'methodology', 'observations', 'research']) {
  const html = await readFile(path.join(root, '.next/server/app', `${route}.html`), 'utf8')
  assert.match(html, /<meta name="robots" content="noindex, follow"/)
}
const manifest = JSON.parse(await readFile(path.join(root, '.next/routes-manifest.json'), 'utf8'))
assert(manifest.headers.some(rule => rule.source === '/:path*' && rule.headers.some(header => header.key === 'X-Robots-Tag' && header.value === 'noindex, nofollow')))
const sitemap = await readFile(path.join(root, '.next/server/app/sitemap.xml.body'), 'utf8')
assert(!sitemap.includes('<loc>'))
const robots = await readFile(path.join(root, '.next/server/app/robots.txt.body'), 'utf8')
assert(!robots.includes('Sitemap:'))
console.log('Preview artifacts contain noindex, a global X-Robots-Tag response rule, and no advertised sitemap URLs.')
