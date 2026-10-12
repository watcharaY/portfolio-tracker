import { test } from '@playwright/test'
import { fileURLToPath } from 'node:url'
import { expected, resetDB, signIn } from '../fixtures'

// Regenerates docs/images from the made-up e2e seed, so screenshots never show
// real data. Runs only with UPDATE_SCREENSHOTS=1 (make docs-screenshots).
test.skip(!process.env.UPDATE_SCREENSHOTS, 'set UPDATE_SCREENSHOTS=1 to refresh docs/images')

const image = (name: string) => fileURLToPath(new URL(`../../docs/images/${name}`, import.meta.url))
const shots: Record<'light' | 'dark', [string, string][]> = {
  light: [['/', 'dashboard-light.png'], ['/brokerage', 'brokerage-light.png'], ['/insurance', 'insurance-light.png']],
  dark: [['/', 'dashboard-dark.png']],
}

test.beforeAll(resetDB)

for (const scheme of ['light', 'dark'] as const) {
  test(`docs screenshots (${scheme})`, async ({ browser, baseURL }) => {
    const context = await browser.newContext({ viewport: { width: 1280, height: 900 }, colorScheme: scheme })
    const page = await context.newPage()
    if (scheme === 'dark') {
      await page.goto('/login')
      await page.screenshot({ path: image('login-dark.png') })
    }
    await signIn(context, baseURL!, expected.users.ready.username)
    for (const [path, file] of shots[scheme]) {
      await page.goto(path)
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: image(file) })
    }
    await context.close()
  })
}
