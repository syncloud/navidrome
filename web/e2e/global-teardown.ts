import * as fs from 'node:fs'
import * as path from 'node:path'

export default async function globalTeardown () {
  const artifactRoot = process.env.PLAYWRIGHT_ARTIFACT_DIR
  if (!artifactRoot) {
    throw new Error('PLAYWRIGHT_ARTIFACT_DIR is not set')
  }

  const label = (process.env.PLAYWRIGHT_DOMAIN ?? 'app').replace(/\.com$/, '')
  const shots = path.join(artifactRoot, `screenshots-${label}`)
  const videos = path.join(artifactRoot, `videos-${label}`)
  fs.mkdirSync(shots, { recursive: true })
  fs.mkdirSync(videos, { recursive: true })

  const resultsDir = 'test-results'
  let pngs = 0
  for (const name of fs.readdirSync(resultsDir)) {
    const dir = path.join(resultsDir, name)
    if (!fs.statSync(dir).isDirectory()) continue
    for (const f of fs.readdirSync(dir)) {
      const src = path.join(dir, f)
      if (f.endsWith('.png')) {
        fs.copyFileSync(src, path.join(shots, `${name}-${f}`))
        pngs++
      }
      if (f.endsWith('.webm')) {
        fs.copyFileSync(src, path.join(videos, `${name}-${f}`))
      }
    }
  }

  if (pngs === 0) {
    throw new Error(`no screenshots collected into ${shots}`)
  }
}
