import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'
import { loginViaAuthelia } from '../helpers/auth'
import { ssh, scpTo } from '../helpers/ssh'

const username = process.env.PLAYWRIGHT_USER!
const password = process.env.PLAYWRIGHT_PASSWORD!
const sampleWma = process.env.PLAYWRIGHT_SAMPLE_WMA!

const album = 'Syncloud Transcode Album'
const song = 'Syncloud Transcode Song'

test.beforeAll(() => {
  ssh('mkdir -p /data/navidrome/transcode')
  scpTo(sampleWma, '/data/navidrome/transcode/song.wma')
  ssh('chown -R navidrome /data/navidrome')
  ssh('snap restart navidrome.navidrome')
  ssh('for i in $(seq 1 30); do test -S /var/snap/navidrome/current/navidrome.sock && exit 0; sleep 2; done; exit 1')
})

test.afterAll(() => {
  const log = ssh("journalctl -u snap.navidrome.navidrome --no-pager | grep -iE 'ffmpeg|transcod' | tail -20", { throw: false })
  console.log('----- navidrome transcode/ffmpeg journal -----\n' + log)
})

test('play a WMA track the browser cannot decode - Navidrome must transcode', async ({ page, baseURL }, info) => {
  const transcodeStatuses: number[] = []
  page.on('response', (r) => {
    if (r.url().includes('getTranscodeStream')) transcodeStatuses.push(r.status())
  })

  await loginViaAuthelia(page, baseURL!, username, password)
  await expect(page).toHaveTitle(/navidrome/i, { timeout: 45_000 })
  await shoot(page, info, '01-logged-in')

  const albumTile = page.getByText(album, { exact: false }).first()
  await expect(albumTile).toBeVisible({ timeout: 120_000 })
  await albumTile.click()

  const songRow = page.getByText(song, { exact: false }).first()
  await expect(songRow).toBeVisible({ timeout: 30_000 })
  await songRow.click()

  await expect(page.locator('.songTitle').first()).toContainText(song, { timeout: 30_000 })
  await shoot(page, info, '02-now-playing')

  await expect
    .poll(() => transcodeStatuses.length, { timeout: 30_000, intervals: [1000] })
    .toBeGreaterThan(0)
  await shoot(page, info, '03-transcode-requested')

  expect(transcodeStatuses, `getTranscodeStream statuses: [${transcodeStatuses}]`).not.toContain(500)

  await expect
    .poll(async () => page.evaluate(() => {
      const a = document.querySelector('audio') as HTMLAudioElement | null
      return a ? a.currentTime : -1
    }), { timeout: 30_000, intervals: [1000] })
    .toBeGreaterThan(0)
  await shoot(page, info, '04-audio-progressing')
})
