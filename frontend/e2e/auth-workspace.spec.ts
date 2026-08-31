// 工作区 E2E 基线：覆盖注册、重新登录、核心数据加载和装备配置保存。
import { expect, test, type Page, type TestInfo } from '@playwright/test'

const password = 'e2e-password-123'

async function registerUser(page: Page, testInfo: TestInfo): Promise<string> {
  const username = `e2e_${Date.now()}_${testInfo.workerIndex}`

  await page.goto('/')
  await expect(page.getByRole('heading', { name: '行动终端' })).toBeVisible()
  await page.getByRole('button', { name: '注册', exact: true }).click()
  await page.getByLabel('用户名').fill(username)
  await page.getByLabel('密码').fill(password)
  await page.getByRole('button', { name: '创建账号', exact: true }).click()

  await expect(page.getByText('服务器在线', { exact: true })).toBeVisible({ timeout: 45_000 })
  await expect(page.getByRole('heading', { name: '探索', exact: true })).toBeVisible({ timeout: 45_000 })
  return username
}

test('注册后退出并重新登录仍能进入工作区', async ({ page }, testInfo) => {
  const username = await registerUser(page, testInfo)

  await page.getByRole('button', { name: '退出', exact: true }).click()
  await expect(page.getByRole('heading', { name: '行动终端' })).toBeVisible()
  await page.getByLabel('用户名').fill(username)
  await page.getByLabel('密码').fill(password)
  await page.getByRole('button', { name: '进入终端', exact: true }).click()

  await expect(page.getByRole('heading', { name: '探索', exact: true })).toBeVisible({ timeout: 45_000 })
  await expect(page.getByRole('button', { name: '探索', exact: true })).toHaveAttribute('aria-current', 'page')
})

test('角色页可以保存装备配置', async ({ page }, testInfo) => {
  await registerUser(page, testInfo)
  await page.getByRole('button', { name: '角色', exact: true }).click()
  await expect(page.getByRole('heading', { name: '玩家角色', exact: true })).toBeVisible({ timeout: 45_000 })

  const saveResponse = page.waitForResponse((response) => (
    response.url().endsWith('/api/loadout') && response.request().method() === 'PUT'
  ))
  await page.getByRole('button', { name: '保存装备配置', exact: true }).click()
  expect((await saveResponse).status()).toBe(200)
  await expect(page.getByText('装备配置已保存', { exact: true })).toBeVisible()
})
