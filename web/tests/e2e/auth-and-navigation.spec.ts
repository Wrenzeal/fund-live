import { expect, test, type Page, type Route } from '@playwright/test'

type SessionState = 'anonymous' | 'authenticated'

interface MockAppOptions {
  avatarURL?: string
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
    await expect(page.getByPlaceholder('例如 005827、蓝筹、易方达')).toBeFocused()

    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog', { name: '搜索基金' })).toBeHidden()
  })

  test('does not overflow the viewport on the login page at mobile width', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await mockAppApi(page)
    await page.goto('/auth/login')

    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)
    expect(overflow).toBe(false)
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
