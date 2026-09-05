import { expect, test } from '@playwright/test'

const adminToken = process.env.DESKLINK_ADMIN_TOKEN

test('production role management page loads with the current administrator session', async ({ page }) => {
  test.skip(!adminToken, 'DESKLINK_ADMIN_TOKEN is required for the production role smoke test')
  await page.addInitScript((token) => localStorage.setItem('desklink_admin_token', token), adminToken!)
  await page.goto('./#/admin/roles')
  await expect(page.getByRole('heading', { name: '角色权限' })).toBeVisible()
  await expect(page.getByText('审计员', { exact: true })).toBeVisible()
  await expect(page.getByText('操作员', { exact: true })).toBeVisible()
  await page.goto('./#/admin/users')
  await expect(page.getByRole('heading', { name: '用户管理' })).toBeVisible()
  await page.getByRole('button', { name: '新增' }).click()
  const roleSelect = page.locator('.modal select').filter({ has: page.locator('option', { hasText: '操作员 (operator)' }) })
  await expect(roleSelect).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)).toBe(false)
})
