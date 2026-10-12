import { expect, test } from '@playwright/test'
import { expected, resetDB, signIn } from '../fixtures'

const menu = [
  { link: 'Brokerage', path: '/brokerage', heading: 'Brokerage Accounts' },
  { link: 'Funds', path: '/funds', heading: 'Mutual Funds' },
  { link: 'Provident fund', path: '/pvd', heading: 'Provident Fund (PVD)' },
  { link: 'Savings', path: '/savings', heading: 'Savings Accounts' },
  { link: 'Insurance', path: '/insurance', heading: 'Insurance Premiums' },
  { link: 'Salary & tax', path: '/salary', heading: 'Salary & Tax' },
]

test.beforeAll(resetDB)
test.beforeEach(async ({ context, baseURL }) => {
  await signIn(context, baseURL!, expected.users.ready.username)
})

test('the overview shows the seeded portfolio total', async ({ page }) => {
  await page.goto('/')
  const total = (expected.dashboard.total.value / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })
  await expect(page.getByRole('heading', { level: 1 })).toContainText(total)
  await expect(page.getByRole('link', { name: 'Overview' })).toHaveAttribute('aria-current', 'page')
})

for (const item of menu) {
  test(`the ${item.link} menu link opens ${item.path}`, async ({ page }) => {
    await page.goto('/')
    const nav = page.getByRole('navigation', { name: 'Sections' })
    await nav.getByRole('link', { name: item.link, exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`${item.path}$`))
    await expect(page.getByRole('heading', { level: 1 })).toHaveText(item.heading)
    await expect(nav.getByRole('link', { name: item.link, exact: true })).toHaveAttribute('aria-current', 'page')
  })
}

test('an unknown path falls back to the overview', async ({ page }) => {
  await page.goto('/no-such-page')
  await expect(page).toHaveURL(/\/$/)
})
