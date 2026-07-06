import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'
import { loginViaAuthelia } from '../helpers/auth'
import { ssh } from '../helpers/ssh'

const username = process.env.PLAYWRIGHT_USER!
const password = process.env.PLAYWRIGHT_PASSWORD!
const domain = process.env.PLAYWRIGHT_DOMAIN ?? 'bookworm.com'
const appDomain = `navidrome.${domain}`
const client = 'e2e-apikey'

test.beforeAll(() => {
  ssh(`curl -sk "https://${appDomain}/rest/ping.view?u=${username}&p=${password}&v=1.16.1&c=${client}&f=json"`)
})

test('generate an API key on a player and authenticate with it', async ({ page, baseURL }, info) => {
  await loginViaAuthelia(page, baseURL!, username, password)
  await expect(page).toHaveTitle(/navidrome/i, { timeout: 45_000 })
  await shoot(page, info, '01-logged-in')

  await page.getByRole('menuitem', { name: /players/i }).click()
  const playerRow = page.getByText(client, { exact: false }).first()
  await expect(playerRow).toBeVisible({ timeout: 30_000 })
  await shoot(page, info, '02-players-list')

  await playerRow.click()
  await page.getByTestId('generate-api-key').click()
  const keyField = page.getByTestId('api-key-value')
  await expect(keyField).toBeVisible({ timeout: 15_000 })
  const key = await keyField.inputValue()
  expect(key).toMatch(/^nav_/)
  await shoot(page, info, '03-key-generated')

  const out = ssh(`curl -sk "https://${appDomain}/rest/ping.view?apiKey=${key}&v=1.16.1&c=${client}&f=json"`)
  expect(out).toContain('"status":"ok"')
})
