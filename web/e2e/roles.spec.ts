import { expect, test } from '@playwright/test'

const permissions = [
  ['devices', '设备'], ['users', '用户'], ['groups', '用户组'], ['device-groups', '设备组'],
  ['address-books', '地址簿'], ['collections', '地址簿集合'], ['collection-rules', '共享规则'],
  ['tags', '标签'], ['login-logs', '登录日志'], ['connection-audit', '连接审计'],
  ['access-rules', '授权管理'], ['file-audit', '文件审计'], ['recordings', '会话录像'],
  ['tokens', '访问令牌'], ['shares', '分享记录'], ['commands', '服务指令'], ['oauth', 'OAuth / OIDC'],
  ['settings', '采集设置'],
].map(([key, label]) => ({ key, label }))

const roles = [
  { id: 1, name: '管理员', code: 'admin', built_in: true, status: 1, permissions: ['*'] },
  { id: 2, name: '审计员', code: 'auditor', built_in: true, status: 1, permissions: ['login-logs', 'connection-audit', 'file-audit', 'recordings', 'shares'] },
  { id: 3, name: '操作员', code: 'operator', built_in: true, status: 1, permissions: ['devices', 'address-books', 'collections', 'collection-rules', 'tags', 'commands'] },
]

test('administrator can manage role menu permissions', async ({ page }, testInfo) => {
  await page.addInitScript(() => localStorage.setItem('desklink_admin_token', 'role-test-token'))
  await page.route('**/api/admin/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    let data: unknown = null
    if (path.endsWith('/user/current')) {
      data = { username: 'admin', nickname: '管理员', is_admin: true, role_id: 1, role_name: '管理员', role_code: 'admin', permissions: ['*'], route_names: ['*'], token: 'role-test-token' }
    } else if (path.endsWith('/role/list')) {
      data = { page: 1, page_size: 1000, total: 3, list: roles }
    } else if (path.endsWith('/role/options')) {
      data = roles
    } else if (path.endsWith('/role/permissions')) {
      data = permissions
    } else if (path.endsWith('/user/list')) {
      data = { page: 1, page_size: 20, total: 0, list: [] }
    }
    await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'success', data }) })
  })

  await page.goto('./#/admin/roles')
  await expect(page.getByRole('heading', { name: '角色权限' })).toBeVisible()
  await expect(page.getByText('管理员', { exact: true }).last()).toBeVisible()
  await expect(page.getByText('审计员', { exact: true })).toBeVisible()
  await expect(page.getByText('操作员', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '新增角色' }).click()
  await expect(page.getByRole('heading', { name: '新增角色' })).toBeVisible()
  const modal = page.locator('.modal')
  await expect(modal.getByText('菜单权限', { exact: true })).toBeVisible()
  await expect(modal.getByText('设备', { exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)).toBe(false)
  await page.screenshot({ path: `/tmp/desklink-roles-${testInfo.project.name}.png`, fullPage: true })

  await page.locator('.modal .desklink-action').filter({ hasText: '关闭' }).click()
  await page.goto('./#/admin/users')
  await page.getByRole('button', { name: '新增' }).click()
  const roleSelect = page.locator('.modal select').filter({ has: page.locator('option', { hasText: '操作员 (operator)' }) })
  await expect(roleSelect).toBeVisible()
  await expect(roleSelect.locator('option')).toHaveCount(4)
})
