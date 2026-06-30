import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'
import { loginViaAuthelia } from '../helpers/auth'
import { ssh, scpTo } from '../helpers/ssh'

const username = process.env.PLAYWRIGHT_USER!
const password = process.env.PLAYWRIGHT_PASSWORD!
const sample = process.env.PLAYWRIGHT_SAMPLE!

const album = 'Syncloud Test Album'
const song = 'Syncloud Test Song'

test.beforeAll(() => {
  ssh('mkdir -p /data/navidrome/sample')
  scpTo(sample, '/data/navidrome/sample/song.mp3')
  ssh('chown -R navidrome /data/navidrome')
  ssh('snap restart navidrome.navidrome')
  ssh('for i in $(seq 1 30); do test -S /var/snap/navidrome/current/navidrome.sock && exit 0; sleep 2; done; exit 1')
})

test('log in, find the scanned album, open it and play the song', async ({ page, baseURL }, info) => {
  await loginViaAuthelia(page, baseURL!, username, password)
  await expect(page).toHaveTitle(/navidrome/i, { timeout: 45_000 })
  await shoot(page, info, '01-logged-in')

  const albumTile = page.getByText(album, { exact: false }).first()
  await expect(albumTile).toBeVisible({ timeout: 120_000 })
  await shoot(page, info, '02-album-scanned')

  await albumTile.click()
  const songRow = page.getByText(song, { exact: false }).first()
  await expect(songRow).toBeVisible({ timeout: 30_000 })
  await shoot(page, info, '03-album-opened')

  await songRow.click()

  await expect(page.locator('.songTitle').first()).toContainText(song, { timeout: 30_000 })
  await shoot(page, info, '04-now-playing')

  await expect.poll(async () => {
    return await page.evaluate(() => {
      const a = document.querySelector('audio') as HTMLAudioElement | null
      return a ? a.currentTime : -1
    })
  }, { timeout: 30_000, intervals: [1000] }).toBeGreaterThan(0)
  await shoot(page, info, '05-audio-progressing')
})
