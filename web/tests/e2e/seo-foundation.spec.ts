import { expect, test } from '@playwright/test'

const origin = 'https://fund.wrenzeal.top'

test.describe('search metadata without JavaScript', () => {
  test.use({ javaScriptEnabled: false })

  test('serves real robots and a sitemap containing only approved public entrances', async ({ request }) => {
    const robots = await request.get('/robots.txt')
    expect(robots.status()).toBe(200)
    expect(robots.headers()['content-type']).toContain('text/plain')
    expect(await robots.text()).toContain(`Sitemap: ${origin}/sitemap.xml`)
    expect(await robots.text()).toMatch(/User-Agent: \*/i)

    const sitemap = await request.get('/sitemap.xml')
    expect(sitemap.status()).toBe(200)
    expect(sitemap.headers()['content-type']).toContain('xml')
    const xml = await sitemap.text()
    const urls = [...xml.matchAll(/<loc>(.*?)<\/loc>/g)].map((match) => match[1])
    expect(urls).toEqual([
      origin, `${origin}/observations`, `${origin}/research`, `${origin}/methodology`,
      `${origin}/announcements`, `${origin}/privacy`, `${origin}/terms`,
    ])
    expect(xml).not.toContain('<lastmod>')
  })

  for (const [path, title] of [
    ['/observations', '基金公告与披露观察'],
    ['/research', '市场行情与历史研究'],
    ['/announcements', '产品更新公告'],
    ['/methodology', '基金估值方法与数据来源'],
    ['/privacy', '隐私政策'],
    ['/terms', '服务条款'],
  ]) {
    test(`server renders unique metadata for ${path}`, async ({ page }) => {
      const response = await page.goto(path)
      expect(response?.status()).toBe(200)
      await expect(page).toHaveTitle(`${title} | 涨了多少 FundLive`)
      await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', `${origin}${path}`)
      await expect(page.locator('meta[name="description"]')).toHaveAttribute('content', /.+/)
      await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'index, follow')
      await expect(page.locator('meta[property="og:url"]')).toHaveAttribute('content', `${origin}${path}`)
      await expect(page.locator('meta[property="og:title"]')).toHaveAttribute('content', `${title} | FundLive`)
      await expect(page.locator('meta[property="og:site_name"]')).toHaveAttribute('content', '涨了多少 · FundLive')
      await expect(page.locator('meta[property="og:locale"]')).toHaveAttribute('content', 'zh_CN')
    })
  }

  for (const path of ['/auth/login?returnTo=%2Fholdings', '/auth/register', '/holdings', '/watchlist', '/issues']) {
    test(`does not index private or utility page ${path}`, async ({ page }) => {
      await page.goto(path)
      await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex, nofollow')
      await expect(page.locator('link[rel="canonical"]')).toHaveCount(0)
    })
  }

  test('keeps client-only detail shells out of the index and rejects malformed fund identifiers', async ({ page }) => {
    for (const path of ['/observations/005827', '/observations/999999']) {
      await page.goto(path)
      const id = path.split('/').at(-1)
      await expect(page).toHaveTitle(`${id} 基金公告与持仓披露 | 涨了多少 FundLive`)
      await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex, follow')
      await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', `${origin}${path}`)
    }
    await page.goto('/announcements/example')
    await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex, follow')
    await expect(page.locator('link[rel="canonical"]')).toHaveCount(0)

    const response = await page.goto('/observations/not-a-fund')
    expect(response?.status()).toBe(404)
    // Next may defer an async notFound UI until hydration. Crawlers must still
    // receive an actual 404 and noindex without a misleading canonical.
    await expect(page.locator('meta[name="robots"]').first()).toHaveAttribute('content', /noindex/)
    for (const meta of await page.locator('meta[name="robots"]').all()) {
      await expect(meta).toHaveAttribute('content', /noindex/)
    }
    await expect(page.locator('link[rel="canonical"]')).toHaveCount(0)
    await expect(page.locator('meta[property="og:url"]')).toHaveCount(0)
  })

  test('serves methodology text, source links and truthful site identity without browser data fetching', async ({ page }) => {
    await page.goto('/methodology')
    await expect(page.getByRole('heading', { level: 1 })).toHaveText('基金估值如何计算，数据能说明什么？')
    await expect(page.getByRole('heading', { name: '盘中估值和官方净值有什么区别？' })).toBeVisible()
    await expect(page.getByText(/预估涨跌幅 = Σ/)).toBeVisible()
    await expect(page.getByText(/覆盖率不等于准确率/)).toBeVisible()
    await expect(page.getByRole('link', { name: '巨潮资讯披露入口' })).toHaveAttribute('href', 'https://www.cninfo.com.cn/')
    await expect(page.getByRole('navigation', { name: '页脚导航' }).getByRole('link', { name: '方法与来源' })).toHaveAttribute('href', '/methodology')
    const identity = JSON.parse(await page.locator('#website-structured-data').textContent() || '{}')
    expect(identity).toMatchObject({ '@type': 'WebSite', url: origin, name: '涨了多少 · FundLive', inLanguage: 'zh-CN' })
    expect(identity).not.toHaveProperty('aggregateRating')
  })
})

test('invalid fund pages offer a recovery path after hydration', async ({ page }) => {
  await page.route('**/api/v1/**', (route) => route.fulfill({ json: { success: true, data: null } }))
  const response = await page.goto('/observations/not-a-fund')
  expect(response?.status()).toBe(404)
  await expect(page.getByRole('heading', { name: '无法识别此基金代码' })).toBeVisible()
  await expect(page.getByRole('link', { name: '返回基金观察' })).toHaveAttribute('href', '/observations')
})

test('home title animation cannot overwrite destination titles after client navigation', async ({ page }) => {
  await page.clock.install()
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path.endsWith('/auth/me')) {
      await route.fulfill({ status: 401, json: { success: false } })
    } else {
      await route.fulfill({ json: { success: true, data: null } })
    }
  })
  await page.goto('/')
  await page.clock.fastForward(7_000)
  await page.getByRole('navigation', { name: '页脚导航' }).getByRole('link', { name: '方法与来源' }).click()
  await expect(page).toHaveURL('/methodology')
  await expect(page).toHaveTitle('基金估值方法与数据来源 | 涨了多少 FundLive')
  await page.clock.fastForward(7_000)
  await expect(page).toHaveTitle('基金估值方法与数据来源 | 涨了多少 FundLive')
})
