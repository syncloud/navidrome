import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'
import { loginViaAuthelia } from '../helpers/auth'

const username = process.env.PLAYWRIGHT_USER!
const password = process.env.PLAYWRIGHT_PASSWORD!

const album = 'Syncloud Test Album'
const song = 'Syncloud Test Song'

test('log in, find the scanned album, open it and play the song', async ({ page, baseURL }, info) => {
  await loginViaAuthelia(page, baseURL!, username, password)
  await expect(page).toHaveTitle(/navidrome/i, { timeout: 45_000 })
  await shoot(page, info, '01-logged-in')

  // The album shows up once Navidrome has scanned the library (give the scan time).
  const albumTile = page.getByText(album, { exact: false }).first()
  await expect(albumTile).toBeVisible({ timeout: 120_000 })
  await shoot(page, info, '02-album-scanned')

  await albumTile.click()
  const songRow = page.getByText(song, { exact: false }).first()
  await expect(songRow).toBeVisible({ timeout: 30_000 })
  await shoot(page, info, '03-album-opened')

  // Clicking a song row plays it (Navidrome rowClick -> playTracks).
  await songRow.click()

  // The player shows the now-playing title.
  await expect(page.locator('.songTitle').first()).toContainText(song, { timeout: 30_000 })
  await shoot(page, info, '04-now-playing')

  // The audio actually progresses, i.e. /rest/stream works through the auth proxy.
  await expect.poll(async () => {
    return await page.evaluate(() => {
      const a = document.querySelector('audio') as HTMLAudioElement | null
      return a ? a.currentTime : -1
    })
  }, { timeout: 30_000, intervals: [1000] }).toBeGreaterThan(0)
  await shoot(page, info, '05-audio-progressing')
})
