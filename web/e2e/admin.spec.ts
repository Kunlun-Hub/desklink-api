import { expect, test } from '@playwright/test'

const adminPassword = process.env.DESKLINK_ADMIN_PASSWORD

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
    }
  } else {
    await page.getByRole('button', { name: '打开导航' }).click()
    await expect(page.getByRole('link', { name: '我的设备', exact: true })).toBeVisible()
    await page.getByRole('link', { name: '我的设备', exact: true }).click()
    await expect(page.getByRole('heading', { name: '我的设备' })).toBeVisible()
  }

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)
  expect(overflow).toBe(false)
  expect(errors).toEqual([])
  await page.screenshot({ path: `/tmp/desklink-admin-${testInfo.project.name}.png`, fullPage: true })
})
