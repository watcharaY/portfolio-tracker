import { expect, test } from '@playwright/test'
import { expected, resetDB, signIn, sql } from '../fixtures'

type Klass = { value: number; cost: number; pl: number; plPct: number; allocationPct: number }

const money = (satang: number) => (satang / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const signed = (satang: number) => (satang > 0 ? '+' : '') + money(satang)
const pct = (ratio: number, digits = 2) => `${(ratio * 100).toFixed(digits)}%`
const cells = (k: Klass) => [money(k.value), money(k.cost), signed(k.pl), pct(k.plPct), pct(k.allocationPct, 1)]

const labels: Record<string, string> = {
  stocks: 'Stocks (Brokerage)',
  funds: 'Mutual Funds',
  pvd: 'Provident Fund',
  savings: 'Savings',
  insurance: 'Insurance (premiums)',
}

test.beforeEach(async ({ context, baseURL }) => {
  await resetDB()
  await signIn(context, baseURL!, expected.users.ready.username)
})

test('every asset class and the total match the hand-computed figures', async ({ page }) => {
  await page.goto('/')
  const d = expected.dashboard
  await expect(page.getByRole('heading', { level: 1 })).toContainText(money(d.total.value))
  await expect(page.locator('.hero-note')).toContainText(money(d.estSavingsInterestPerYear))

  const table = page.locator('section.ledger table')
  for (const [key, label] of Object.entries(labels)) {
    const row = table.locator('tbody tr').filter({ hasText: label })
    await expect(row.locator('td.num'), label).toHaveText(cells(d.classes[key]))
  }
  await expect(table.locator('tfoot td.num')).toHaveText(cells(d.total))

  const chart = (await page.getByRole('img', { name: /^Allocation:/ }).getAttribute('aria-label')) ?? ''
  for (const label of Object.values(labels)) expect(chart).toContain(label)
})

test('an asset class worth zero stays in the table but leaves the chart', async ({ page }) => {
  await sql('DELETE FROM savings_accounts')
  await page.goto('/')
  const row = page.locator('section.ledger tbody tr').filter({ hasText: 'Savings' })
  await expect(row.locator('td.num').first()).toHaveText('0.00')
  await expect(page.getByRole('img', { name: /^Allocation:/ })).not.toHaveAccessibleName(/Savings/)
})

test('an empty portfolio shows the getting-started message', async ({ page }) => {
  await sql('TRUNCATE brokers, funds, pvd_years, savings_accounts, insurers CASCADE')
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('สมุดพอร์ตยังว่างอยู่')
  await page.getByRole('link', { name: 'โบรกเกอร์' }).click()
  await expect(page).toHaveURL(/\/brokerage$/)
})
