import { expect, test } from '@playwright/test'

test.describe('anonymous observation SSR', () => {
  test.use({ javaScriptEnabled: false })

  test('renders the verified name, sources and disclosure in initial HTML and opens indexing', async ({ page, context }) => {
    await context.addCookies([{ name: 'fundlive_session', value: 'must-never-leave-the-browser', domain: '127.0.0.1', path: '/' }])
    const response = await page.goto('/observations/000001')
    expect(response?.status()).toBe(200)
    await expect(page.getByRole('heading', { level: 1 })).toHaveText('服务端观察测试基金')
    await expect(page).toHaveTitle('服务端观察测试基金（000001）公告与持仓披露 | 涨了多少 FundLive')
    await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'index, follow')
    await expect(page.getByText('42.62%', { exact: true })).toBeVisible()
    await expect(page.getByText('2026-09-30', { exact: true })).toBeVisible()
    await expect(page.getByRole('link', { name: '可追溯的基金公开披露' })).toHaveAttribute('href', 'https://example.com/fund/disclosure')
    const schema = JSON.parse(await page.locator('#observation-structured-data').textContent() || '{}')
    expect(schema['@graph'][0]).toMatchObject({ '@type': 'WebPage', about: { name: '服务端观察测试基金', identifier: '000001' } })
    expect(schema['@graph'][0].citation).toContain('https://example.com/fund/disclosure')
    expect(schema['@graph'][1]['@type']).toBe('BreadcrumbList')
    expect(schema['@graph'][0]).not.toHaveProperty('dateModified')
  })

  test('serves a real 404 only for confirmed absence', async ({ page }) => {
    const response = await page.goto('/observations/000002')
    expect(response?.status()).toBe(404)
    for (const meta of await page.locator('meta[name="robots"]').all()) await expect(meta).toHaveAttribute('content', /noindex/)
    await expect(page.locator('link[rel="canonical"]')).toHaveCount(0)
  })

  for (const id of ['000004', '000005', '000008']) {
    test(`upstream failure or wrong contract ${id} is recoverable and never a false missing fund`, async ({ page }) => {
      const response = await page.goto(`/observations/${id}`)
      expect(response?.status()).toBe(200)
      await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex, follow')
      await expect(page.getByText('未能读取公告与披露快照，这不表示基金不存在或没有公告。页面会尝试重新读取。')).toBeVisible()
      await expect(page.locator('#observation-structured-data')).toHaveCount(0)
    })
  }

  for (const id of ['000003', '000006']) {
    test(`partial or old snapshot ${id} remains readable but not indexing-ready`, async ({ page }) => {
      await page.goto(`/observations/${id}`)
      await expect(page.getByRole('heading', { level: 1 })).toHaveText('服务端观察测试基金')
      await expect(page.getByRole('link', { name: '可追溯的基金公开披露' })).toBeVisible()
      await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex, follow')
    })
  }

  test('metadata and JSON-LD cannot become executable HTML', async ({ page }) => {
    await page.goto('/observations/000007')
    await expect(page.locator('#injected-script')).toHaveCount(0)
    await expect(page.getByRole('heading', { level: 1 })).toContainText('</script>')
    await expect(page.locator('a[href^="javascript:"]')).toHaveCount(0)
    const schema = JSON.parse(await page.locator('#observation-structured-data').textContent() || '{}')
    expect(schema['@graph'][0].name).toContain('</script>')
  })

  test('publishes only eligible snapshots as crawlable directory links and sitemap URLs', async ({ page, request }) => {
    await page.goto('/observations')
    await expect(page.getByRole('link', { name: /公开目录测试基金/ })).toHaveAttribute('href', '/observations/320007')
    const xml = await (await request.get('/sitemap.xml')).text()
    expect(xml).toContain('https://fund.wrenzeal.top/observations/320007')
    expect(xml).not.toContain('https://fund.wrenzeal.top/observations/005827')
    expect(xml).not.toContain('https://fund.wrenzeal.top/observations/000003')
  })

  test('public identity and image previews have real response contracts', async ({ page, request }) => {
    await page.goto('/about')
    await expect(page.getByRole('heading', { name: '涨了多少 · FundLive', exact: true })).toBeVisible()
    await expect(page.locator('meta[name="twitter:card"]')).toHaveAttribute('content', 'summary_large_image')
    const imagePath = await page.locator('meta[property="og:image"]').getAttribute('content')
    expect(imagePath).toContain('/opengraph-image')
    const image = await request.get(new URL(imagePath!).pathname)
    expect(image.status()).toBe(200)
    expect(image.headers()['content-type']).toContain('image/png')
    const body = await image.body()
    expect(body.readUInt32BE(16)).toBe(1200)
    expect(body.readUInt32BE(20)).toBe(630)
    expect((await request.get('/api/v1/auth/config')).headers()['x-robots-tag']).toBe('noindex, nofollow')
  })

  test('product announcements are real anonymous HTML with author and dates', async ({ page, request }) => {
    await page.goto('/announcements')
    await expect(page.getByRole('link', { name: /公开产品更新测试/ })).toHaveAttribute('href', '/announcements/release-one')
    await page.goto('/announcements/release-one')
    await expect(page.getByRole('heading', { level: 1 })).toHaveText('公开产品更新测试')
    await expect(page.getByText('可在禁用 JavaScript 后直接阅读的完整公告正文。')).toBeVisible()
    await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'index, follow')
    await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', 'https://fund.wrenzeal.top/announcements/release-one')
    const schema = JSON.parse(await page.locator('#announcement-structured-data').textContent() || '{}')
    expect(schema).toMatchObject({ '@type': 'Article', headline: '公开产品更新测试', author: { name: 'FundLive 项目' } })
    await expect(page.locator(`time[datetime="${schema.datePublished}"]`).first()).toBeVisible()
    const missing = await page.goto('/announcements/missing-one')
    expect(missing?.status()).toBe(404)
    const xml = await (await request.get('/sitemap.xml')).text()
    expect(xml).toContain('/announcements/release-one')
    expect(xml).not.toContain('/announcements/missing-one')
  })
})
