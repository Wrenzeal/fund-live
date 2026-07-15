import { expect, test, type Page, type Route } from '@playwright/test'

type SessionState = 'anonymous' | 'authenticated'

async function mockAppApi(page: Page, initialSession: SessionState = 'anonymous') {
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
              avatar_url: '',
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
})
