import { expect, test, type Page, type Route } from '@playwright/test'

type SessionState = 'anonymous' | 'authenticated'

interface MockAppOptions {
  avatarURL?: string
  fundSearchResults?: Record<string, Array<{ id: string; name: string }>>
  onFundSearch?: (query: string) => void
}

async function mockAppApi(page: Page, initialSession: SessionState = 'anonymous', options: MockAppOptions = {}) {
  let session = initialSession

  await page.route('**/api/v1/**', async (route: Route) => {
    const path = new URL(route.request().url()).pathname

    if (path.endsWith('/auth/config')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            google_client_id: '',
            google_login_enabled: false,
            email_code_login_enabled: true,
          },
        }),
      })
      return
    }

    if (path.endsWith('/auth/me')) {
      if (session === 'anonymous') {
        await route.fulfill({ status: 401, contentType: 'application/json', body: '{}' })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            user: {
              id: 'test-user',
              email: 'tester@example.com',
              display_name: '测试用户',
              avatar_url: options.avatarURL ?? '',
              is_admin: false,
              preferred_quote_source: 'sina',
              provider: 'email_code',
              email_verified: true,
              created_at: '2026-01-01T00:00:00Z',
              updated_at: '2026-01-01T00:00:00Z',
            },
            expires_at: '2099-01-01T00:00:00Z',
          },
        }),
      })
      return
    }

    if (path.endsWith('/auth/email/start')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { email: 'tester@example.com', expires_in_seconds: 600, resend_after_seconds: 60 },
        }),
      })
      return
    }

    if (path.endsWith('/auth/email/verify')) {
      session = 'authenticated'
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { user: { id: 'test-user' } } }),
      })
      return
    }

    if (path.endsWith('/fund/search')) {
      const query = new URL(route.request().url()).searchParams.get('q') ?? ''
      options.onFundSearch?.(query)
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: options.fundSearchResults?.[query] ?? [] }),
      })
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: null }),
    })
  })
}

test.describe('authentication and navigation', () => {
  test('completes the email-code login flow without exposing a development code', async ({ page }) => {
    await mockAppApi(page)
    await page.goto('/auth/login')

    await expect(page.getByRole('heading', { name: '登录账户' })).toBeVisible()
    await page.getByLabel('邮箱').fill('tester@example.com')
    await page.getByRole('button', { name: /发送验证码/ }).click()

    await expect(page.getByText('验证码已发送')).toBeVisible()
    await expect(page.getByLabel('验证码')).toHaveAttribute('autocomplete', 'one-time-code')
    await expect(page.getByRole('button', { name: /后重发|重新发送/ })).toBeDisabled()

    await page.getByLabel('验证码').fill('123456')
    await page.getByRole('button', { name: /登录/ }).last().click()
    await expect(page).toHaveURL(/\/$/)
  })

  test('keeps the primary task destinations reachable from the shell', async ({ page }) => {
    await mockAppApi(page)
    await page.goto('/')

    await expect(page.getByRole('link', { name: '自选' })).toBeVisible()
    await expect(page.getByRole('link', { name: '持仓' })).toBeVisible()
    await expect(page.getByRole('link', { name: '量化' })).toBeVisible()
  })

  test('keeps primary navigation visible and selected across breakpoints', async ({ page }) => {
    await mockAppApi(page)

    for (const viewport of [
      { name: 'desktop', width: 1440, height: 900 },
      { name: 'tablet', width: 1024, height: 768 },
      { name: 'mobile', width: 390, height: 844 },
    ]) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await page.goto('/watchlist')

      const navigation = page.getByRole('navigation', { name: '主要页面' })
      await expect(navigation, `${viewport.name} navigation`).toBeVisible()

      for (const label of ['首页', '自选', '持仓', '量化']) {
        await expect(navigation.getByRole('link', { name: label, exact: true })).toBeVisible()
      }

      await expect(navigation.getByRole('link', { name: '自选', exact: true })).toHaveAttribute('aria-current', 'page')
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)
      expect(overflow, `${viewport.name} horizontal overflow`).toBe(false)
    }
  })

  test('theme switching is keyboard reachable and persists across reloads', async ({ page }) => {
    await mockAppApi(page)
    await page.goto('/')

    const themeButton = page.getByRole('button', { name: /主题|暖白|深色/ }).last()
    await themeButton.focus()
    await expect(themeButton).toBeFocused()
    await themeButton.press('Enter')
    await page.locator('.switcher-dropdown-panel button').filter({ hasText: /^深色/ }).first().click()
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')

    await page.reload()
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  })

  test('opens search as a focused dialog and closes with Escape', async ({ page }) => {
    await mockAppApi(page)
    await page.goto('/')

    await page.getByRole('button', { name: '搜索基金代码或名称' }).first().click()
    await expect(page.getByRole('dialog', { name: '搜索基金' })).toBeVisible()
    await expect(page.getByPlaceholder('例如 005827、蓝、易方达')).toBeFocused()

    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog', { name: '搜索基金' })).toBeHidden()
  })

  test('searches name fragments from one Han character without false empty states', async ({ page }) => {
    const searchQueries: string[] = []
    await mockAppApi(page, 'anonymous', {
      onFundSearch: (query) => searchQueries.push(query),
      fundSearchResults: {
        蓝: [{ id: '005827', name: '易方达蓝筹精选混合' }],
        '00': [{ id: '005827', name: '易方达蓝筹精选混合' }],
      },
    })
    await page.goto('/')
    await page.getByRole('button', { name: '搜索基金代码或名称' }).first().click()

    const dialog = page.getByRole('dialog', { name: '搜索基金' })
    const input = dialog.getByRole('textbox')

    for (const query of ['a', '0']) {
      await input.fill(query)
      await page.waitForTimeout(450)
      expect(searchQueries).toEqual([])
      await expect(dialog.getByText('中文名称可输入 1 个字；基金代码或英文至少输入 2 个字符。')).toBeVisible()
      await expect(dialog.getByText('没有找到匹配基金')).toHaveCount(0)
    }

    await input.fill('蓝')
    await expect.poll(() => searchQueries).toContain('蓝')
    await expect(dialog.getByText('易方达蓝筹精选混合')).toBeVisible()

    await input.fill('00')
    await expect.poll(() => searchQueries).toContain('00')
  })

  test('does not overflow the viewport on the login page at mobile width', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await mockAppApi(page)
    await page.goto('/auth/login')

    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)
    expect(overflow).toBe(false)
  })

  test('keeps the mobile app bar compact and touch friendly', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await mockAppApi(page)
    await page.goto('/')

    const appHeader = page.getByRole('banner').first()
    await expect(appHeader).toBeVisible()
    await expect(appHeader.getByRole('link', { name: '登录', exact: true })).toBeVisible()
    await expect(appHeader.getByRole('link', { name: '注册', exact: true })).toBeHidden()
    await expect(appHeader.getByRole('button', { name: '界面设置' })).toBeVisible()

    const headerHeight = await appHeader.evaluate((element) => element.getBoundingClientRect().height)
    expect(headerHeight).toBeLessThanOrEqual(128)

    const undersizedTargets = await appHeader.locator('a:visible, button:visible').evaluateAll((elements) => (
      elements
        .map((element) => {
          const rect = element.getBoundingClientRect()
          return { label: element.getAttribute('aria-label') || element.textContent?.trim() || element.tagName, width: rect.width, height: rect.height }
        })
        .filter(({ width, height }) => width < 44 || height < 44)
    ))
    expect(undersizedTargets).toEqual([])

    await appHeader.getByRole('button', { name: '界面设置' }).click()
    const settings = page.getByRole('dialog', { name: '界面设置' })
    await expect(settings.getByText('主题')).toBeVisible()
    await expect(settings.getByText('视图模式')).toBeVisible()
  })

  test('keeps the login task in the first mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await mockAppApi(page)
    await page.goto('/auth/login')

    const sendButton = page.getByRole('button', { name: '发送验证码' })
    const capabilityDisclosure = page.locator('details').filter({ hasText: '登录后可使用' })

    await expect(sendButton).toBeVisible()
    await expect(capabilityDisclosure).toBeVisible()
    await expect(capabilityDisclosure).not.toHaveAttribute('open', '')

    const metrics = await page.evaluate(() => {
      const header = document.querySelector('header')?.getBoundingClientRect()
      const send = Array.from(document.querySelectorAll('button')).find((button) => button.textContent?.includes('发送验证码'))?.getBoundingClientRect()
      return { headerHeight: header?.height ?? 999, sendBottom: send?.bottom ?? 9999, viewportHeight: window.innerHeight }
    })
    expect(metrics.headerHeight).toBeLessThanOrEqual(72)
    expect(metrics.sendBottom).toBeLessThan(metrics.viewportHeight)
  })

  test('removes duplicate account navigation only on mobile', async ({ page }) => {
    await mockAppApi(page)

    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto('/watchlist')
    await expect(page.getByRole('navigation', { name: '账户页面' })).toBeHidden()

    await page.setViewportSize({ width: 1440, height: 900 })
    await page.reload()
    await expect(page.getByRole('navigation', { name: '账户页面' })).toBeVisible()
  })

  test('keeps analysis pages inside the global mobile shell', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await mockAppApi(page)

    await page.goto('/analysis/rankings')
    await expect(page.getByRole('navigation', { name: '主要页面' })).toBeVisible()
    await expect(page.getByRole('navigation', { name: '主要页面' }).getByRole('link', { name: '量化' })).toHaveAttribute('aria-current', 'page')
    await expect(page.getByRole('link', { name: '返回首页' })).toHaveCount(0)

    await page.goto('/analysis/005827')
    await expect(page.getByRole('navigation', { name: '主要页面' })).toBeVisible()
    await expect(page.getByRole('link', { name: '返回基金详情' })).toBeVisible()
  })

  test('uses a two-by-two feedback summary on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await mockAppApi(page)
    await page.goto('/issues')

    const cards = page.locator('article')
    await expect(cards).toHaveCount(4)
    const boxes = await cards.evaluateAll((elements) => elements.map((element) => {
      const rect = element.getBoundingClientRect()
      return { x: Math.round(rect.x), y: Math.round(rect.y), width: Math.round(rect.width) }
    }))

    expect(boxes[0].y).toBe(boxes[1].y)
    expect(boxes[2].y).toBe(boxes[3].y)
    expect(boxes[2].y).toBeGreaterThan(boxes[0].y)
    expect(boxes[0].width).toBeLessThan(180)
  })

  test('avoids horizontal overflow across common mobile widths and key routes', async ({ page }) => {
    test.slow()
    await mockAppApi(page)

    for (const viewport of [
      { width: 360, height: 800 },
      { width: 390, height: 844 },
      { width: 430, height: 932 },
    ]) {
      await page.setViewportSize(viewport)

      for (const route of ['/', '/watchlist', '/holdings', '/auth/login', '/analysis/rankings', '/analysis/005827', '/issues']) {
        await page.goto(route)
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)
        expect(overflow, `${route} at ${viewport.width}px`).toBe(false)
      }
    }
  })

  test('falls back to initials when a saved avatar cannot load', async ({ page }) => {
    await page.route('**/broken-avatar.png', async (route) => {
      await route.fulfill({ status: 404, contentType: 'image/png', body: '' })
    })
    await mockAppApi(page, 'authenticated', {
      avatarURL: 'http://127.0.0.1:3100/broken-avatar.png',
    })
    await page.goto('/')

    const accountButton = page.getByRole('button', { name: '打开测试用户的账户菜单' })
    await expect(accountButton.locator('[data-avatar-state="fallback"]')).toBeVisible()
    await expect(accountButton.locator('img')).toHaveCount(0)

    await accountButton.click()
    await expect(page.locator('[data-avatar-state="fallback"]')).toHaveCount(2)
  })
})
