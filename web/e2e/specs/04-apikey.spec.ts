import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'
import { loginViaAuthelia } from '../helpers/auth'
import { ssh } from '../helpers/ssh'

const username = process.env.PLAYWRIGHT_USER!
const password = process.env.PLAYWRIGHT_PASSWORD!
const domain = process.env.PLAYWRIGHT_DOMAIN ?? 'bookworm.com'
const appDomain = `navidrome.${domain}`

test('create an API key on the account page and authenticate with it', async ({ page, baseURL }, info) => {
  await loginViaAuthelia(page, baseURL!, username, password)
  await expect(page).toHaveTitle(/navidrome/i, { timeout: 45_000 })
  await shoot(page, info, '01-logged-in')

  await page.getByRole('button', { name: 'Settings' }).click()
  await page.getByRole('menuitem', { name: /api keys/i }).click()
  await expect(page.getByRole('link', { name: /add api key/i })).toBeVisible({ timeout: 30_000 })
  await shoot(page, info, '02-apikey-page')

  await page.getByRole('link', { name: /add api key/i }).click()
  await page.getByTestId('apikey-name').fill('e2e-key')
  await page.getByRole('button', { name: /save/i }).click()

  await page.getByTestId('apikey-show').first().click()
  const key = (await page.getByTestId('apikey-value').first().textContent())?.trim()
  expect(key).toMatch(/^nav_/)
  await shoot(page, info, '03-key-created')

  const out = ssh(`curl -sk "https://${appDomain}/rest/ping.view?apiKey=${key}&v=1.16.1&c=e2e&f=json"`)
  expect(out).toContain('"status":"ok"')
})
