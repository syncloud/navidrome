import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'
import { loginViaAuthelia } from '../helpers/auth'

const username = process.env.PLAYWRIGHT_USER!
const password = process.env.PLAYWRIGHT_PASSWORD!

test('logs in via Syncloud SSO and lands in Navidrome', async ({ page, baseURL }, info) => {
  await loginViaAuthelia(page, baseURL!, username, password)
  await expect(page).toHaveTitle(/navidrome/i, { timeout: 45_000 })
  await shoot(page, info, 'navidrome-home')
})
