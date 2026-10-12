import { expect, test } from '@playwright/test'
import { expected, resetDB, signIn } from '../fixtures'
import { resourceTable } from '../resource-table'

// Every editable table, with a seeded row and one input to change.
const tables = [
  { path: '/brokerage', title: 'Brokers', row: 'Broker Alpha', field: 'Note' },
  { path: '/brokerage', title: 'Deposit / Withdraw log', row: 'initial', field: 'Note' },
  { path: '/funds', title: 'Funds', row: 'Fund Gold E', field: 'Note' },
  { path: '/pvd', title: 'Yearly contributions', row: '2025', field: 'Fund Allocation' },
  { path: '/savings', title: 'Accounts', row: 'Bank B', field: 'Note' },
  { path: '/insurance', title: 'Companies', row: 'Insurer One', field: 'Note' },
  { path: '/insurance', title: 'Premium log', row: 'satang, no year', field: 'Note' },
  { path: '/salary', title: 'Monthly income 2025', row: 'Mar', field: 'Bonus', year: 2025 },
  { path: '/salary', title: 'Deductions 2025', row: 'Life insurance', field: 'Item', year: 2025 },
  { path: '/dividends', title: 'Payments 2025', row: '2025-04-20', field: 'Note', year: 2025 },
]

test.beforeAll(resetDB)
test.beforeEach(async ({ context, baseURL }) => {
  await signIn(context, baseURL!, expected.users.ready.username)
})

for (const t of tables) {
  test(`${t.title}: Edit opens the row without saving, Cancel discards the change`, async ({ page }) => {
    const writes: string[] = []
    page.on('request', (r) => {
      if (['POST', 'PUT', 'DELETE'].includes(r.method()) && r.url().includes('/api/')) writes.push(`${r.method()} ${r.url()}`)
    })
    await page.goto(t.path)
    if (t.year) await page.locator('header.page-head').getByLabel('Year').fill(String(t.year))
    const table = resourceTable(page, t.title)
    // Ref columns (Broker, Company) fill in once their option lists load.
    await page.waitForLoadState('networkidle')
    await expect(table.row(t.row)).toBeVisible()
    const before = await table.row(t.row).locator('td').allInnerTexts()

    await table.startEdit(t.row)
    await page.waitForTimeout(300)
    await expect(table.editRow).toBeVisible()
    await table.fill({ [t.field]: 'changed-then-cancelled' })
    await table.cancel()

    await expect(table.row(t.row).locator('td')).toHaveText(before)
    expect(writes, 'Edit and Cancel must not write').toEqual([])
  })
}
