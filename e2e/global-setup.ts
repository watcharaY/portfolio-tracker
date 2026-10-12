import config from './playwright.config'

// The API migrates the schema before it listens, so a healthy /api/health also
// means the tables exist and the seed can load.
export default async function globalSetup() {
  const base = config.use?.baseURL ?? 'http://localhost:8100'
  const deadline = Date.now() + 120_000
  let last = ''
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${base}/api/health`)
      if (res.ok) return
      last = `status ${res.status}`
    } catch (err) {
      last = (err as Error).message
    }
    await new Promise((r) => setTimeout(r, 1000))
  }
  throw new Error(`API at ${base} not healthy after 120s: ${last}`)
}
