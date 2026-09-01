import { expect, test } from '@playwright/test'

const adminPassword = process.env.DESKLINK_ADMIN_PASSWORD

async function expectControlsAligned(page: import('@playwright/test').Page) {
  const result = await page.evaluate(() => {
    const fields = Array.from(document.querySelectorAll<HTMLElement>('.desklink-field'))
    const fieldErrors = fields.flatMap((field, index) => {
      const label = field.querySelector<HTMLElement>(':scope > .label')
      const control = field.querySelector<HTMLElement>(':scope > .input, :scope > .select, :scope > .textarea, :scope > span:nth-child(2)')
      if (!label || !control) return [`field ${index} is missing a label or control`]
      const labelRect = label.getBoundingClientRect()
      const controlRect = control.getBoundingClientRect()
      const errors: string[] = []
      if (controlRect.top < labelRect.bottom - 0.5) errors.push(`field ${index} label overlaps its control`)
      if (Math.abs(controlRect.left - field.getBoundingClientRect().left) > 1) errors.push(`field ${index} control is horizontally offset`)
      return errors
    })
    const buttonErrors = Array.from(document.querySelectorAll<HTMLElement>('button.btn')).flatMap((button, index) => {
      if (button.scrollWidth > button.clientWidth + 1 || button.scrollHeight > button.clientHeight + 1) return [`button ${index} content overflows`]
      return []
    })
    return [...fieldErrors, ...buttonErrors]
  })
  expect(result).toEqual([])
}

async function fieldControlRects(locator: import('@playwright/test').Locator) {
  return locator.evaluateAll((fields) => fields.map((field) => {
    const control = field.matches('input')
      ? field
      : field.querySelector<HTMLElement>(':scope > .input, :scope > .select, :scope > span:nth-child(2)')
    const rect = control?.getBoundingClientRect()
    return {
      left: Math.round(rect?.left || 0),
      width: Math.round(rect?.width || 0),
      height: Math.round(rect?.height || 0),
    }
  }))
}

test('renders the branded login page without overflow', async ({ page }, testInfo) => {
  await page.goto('./#/login')
  await expect(page.getByRole('heading', { name: '登录管理中心' })).toBeVisible()
  await expect(page.getByText('DeskLink', { exact: true }).first()).toBeVisible()
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)
  expect(overflow).toBe(false)
  await page.screenshot({ path: `/tmp/desklink-login-${testInfo.project.name}.png`, fullPage: true })
})

test('admin can log in and open core management pages', async ({ page }, testInfo) => {
  test.skip(!adminPassword, 'DESKLINK_ADMIN_PASSWORD is required for the authenticated smoke test')
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.message))
  page.on('console', (message) => { if (message.type() === 'error') errors.push(message.text()) })

  await page.goto('./#/login')
  await page.getByPlaceholder('请输入用户名').fill('admin')
  await page.getByPlaceholder('请输入密码').fill(adminPassword!)
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page).toHaveURL(/#\/$/)
  await expect(page.getByRole('heading', { name: /你好/ })).toBeVisible()
  await expect(page.getByText('DeskLink 1.4.9', { exact: true })).toBeVisible()

  if (testInfo.project.name === 'desktop') {
    const routes = [
      ['profile', '个人资料'],
      ['my/devices', '我的设备'],
      ['my/collections', '我的地址簿集合'],
      ['my/address-books', '我的地址簿'],
      ['my/tags', '我的标签'],
      ['my/collection-rules', '我的共享规则'],
      ['my/shares', '我的分享记录'],
      ['my/login-logs', '我的登录记录'],
      ['admin/devices', '设备管理'],
      ['admin/users', '用户管理'],
      ['admin/groups', '用户组'],
      ['admin/device-groups', '设备组'],
      ['admin/address-books', '地址簿'],
      ['admin/collections', '地址簿集合'],
      ['admin/collection-rules', '共享规则'],
      ['admin/tags', '标签管理'],
      ['admin/login-logs', '登录日志'],
      ['admin/connection-audit', '连接审计'],
      ['admin/file-audit', '文件审计'],
      ['admin/tokens', '访问令牌'],
      ['admin/shares', '分享记录'],
      ['admin/commands', '服务指令'],
      ['admin/oauth', 'OAuth / OIDC'],
      ['settings', '系统信息'],
    ]
    for (const [route, heading] of routes) {
      await page.goto(`./#/${route}`)
      await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible()
      await expectControlsAligned(page)
      if (route === 'profile') {
        const inputRects = await fieldControlRects(page.getByTestId('password-form').locator('input'))
        expect(new Set(inputRects.map((rect) => rect.left)).size).toBe(1)
        expect(new Set(inputRects.map((rect) => rect.width)).size).toBe(1)
        expect(new Set(inputRects.map((rect) => rect.height))).toEqual(new Set([36]))
        await page.screenshot({ path: '/tmp/desklink-profile-aligned.png', fullPage: true })
      }
      if (route === 'admin/users') {
        const filterControlTops = await page.locator('form.desklink-card').first().locator('input, button').evaluateAll((controls) => controls.map((control) => Math.round(control.getBoundingClientRect().top)))
        expect(new Set(filterControlTops).size).toBe(1)
        await page.getByRole('button', { name: '新增' }).click()
        await expect(page.getByRole('heading', { name: '新增用户' })).toBeVisible()
        await expectControlsAligned(page)
        const controlRects = await fieldControlRects(page.locator('.modal .desklink-field'))
        expect(new Set(controlRects.map((rect) => rect.height))).toEqual(new Set([36]))
        await page.waitForTimeout(250)
        await page.screenshot({ path: '/tmp/desklink-user-modal-aligned.png', fullPage: true })
        await page.getByRole('button', { name: '取消' }).click()
      }
    }
  } else {
    await page.getByRole('button', { name: '打开导航' }).click()
    await expect(page.getByRole('link', { name: '我的设备', exact: true })).toBeVisible()
    await page.getByRole('link', { name: '我的设备', exact: true }).click()
    await expect(page.getByRole('heading', { name: '我的设备' })).toBeVisible()
    await page.goto('./#/profile')
    await expect(page.getByRole('heading', { name: '个人资料', exact: true })).toBeVisible()
    await expectControlsAligned(page)
    const inputRects = await fieldControlRects(page.getByTestId('password-form').locator('input'))
    expect(new Set(inputRects.map((rect) => rect.left)).size).toBe(1)
    expect(new Set(inputRects.map((rect) => rect.width)).size).toBe(1)
    await page.screenshot({ path: '/tmp/desklink-profile-mobile-aligned.png', fullPage: true })
  }

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)
  expect(overflow).toBe(false)
  expect(errors).toEqual([])
  await page.screenshot({ path: `/tmp/desklink-admin-${testInfo.project.name}.png`, fullPage: true })
})
