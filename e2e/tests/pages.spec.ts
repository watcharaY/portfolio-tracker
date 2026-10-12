import { expect, test, type Page } from '@playwright/test'
import { expected, resetDB, signIn } from '../fixtures'
import { resourceTable } from '../resource-table'

// Expected cells come from backend/testdata/e2e-seed.sql, worked out by hand.

test.beforeEach(async ({ context, baseURL }) => {
  await resetDB()
  await signIn(context, baseURL!, expected.users.ready.username)
})

async function expectCells(t: ReturnType<typeof resourceTable>, row: string, cells: Record<string, string>) {
  for (const [label, text] of Object.entries(cells)) {
    await expect(await t.cell(row, label), `${row} / ${label}`).toHaveText(text)
  }
}

async function expectTotals(t: ReturnType<typeof resourceTable>, totals: Record<string, string>) {
  for (const [label, text] of Object.entries(totals)) {
    await expect(await t.total(label), `total / ${label}`).toHaveText(text)
  }
}

test.describe('Brokerage', () => {
  test('shows net capital and P/L computed from the transaction log', async ({ page }) => {
    await page.goto('/brokerage')
    const brokers = resourceTable(page, 'Brokers')
    await expect(brokers.rows).toHaveCount(2)
    await expectCells(brokers, 'Broker Alpha', {
      'Current Value': '1,200,000.00',
      Deposits: '1,200,000.00',
      Withdrawals: '100,000.00',
      'Net Capital': '1,100,000.00',
      'Unrealised P/L': '100,000.00',
      'P/L %': '9.09%',
    })
    await expectCells(brokers, 'Broker Beta', { 'Net Capital': '0.00', 'P/L %': '–' })
    await expectTotals(brokers, { 'Current Value': '1,200,000.00', 'Net Capital': '1,100,000.00' })
    await expect(resourceTable(page, 'Deposit / Withdraw log').rows).toHaveCount(5)
  })

  test('adds, edits and deletes a broker; a duplicate name is refused', async ({ page }) => {
    await page.goto('/brokerage')
    const brokers = resourceTable(page, 'Brokers')
    await brokers.add({ Broker: 'Broker New', 'Current Value': '1,000.50', Note: 'e2e' })
    await expectCells(brokers, 'Broker New', { 'Current Value': '1,000.50', Deposits: '0.00', Note: 'e2e' })

    await brokers.edit('Broker New', { 'Current Value': '2000' })
    await expectCells(brokers, 'Broker New', { 'Current Value': '2,000.00' })

    await brokers.startAdd()
    await brokers.fill({ Broker: 'Broker Alpha' })
    await brokers.save()
    await expect(brokers.error).toContainText('already exists')
    await brokers.cancel()

    await brokers.remove('Broker New')
    await expect(brokers.rows).toHaveCount(2)
  })

  test('a recorded deposit updates the broker totals', async ({ page }) => {
    await page.goto('/brokerage')
    const brokers = resourceTable(page, 'Brokers')
    const log = resourceTable(page, 'Deposit / Withdraw log')

    await log.add({ Date: '2026-03-01', Broker: 'Broker Beta', Type: 'Deposit', Amount: '500', Note: 'e2e-txn' })
    await expectCells(brokers, 'Broker Beta', { Deposits: '50,500.00', 'Net Capital': '500.00' })

    await log.edit('e2e-txn', { Type: 'Withdraw', Amount: '200' })
    await expectCells(brokers, 'Broker Beta', { Deposits: '50,000.00', Withdrawals: '50,200.00' })

    await log.startAdd()
    await log.fill({ Broker: 'Broker Beta', Amount: '0' })
    await log.save()
    await expect(log.error).toContainText('amount must be greater than 0')
    await log.cancel()

    await log.remove('e2e-txn')
    await expectCells(brokers, 'Broker Beta', { 'Net Capital': '0.00' })
  })
})

test.describe('Funds', () => {
  test('shows P/L per fund and totals', async ({ page }) => {
    await page.goto('/funds')
    const funds = resourceTable(page, 'Funds')
    await expect(funds.rows).toHaveCount(9)
    await expectCells(funds, 'Fund Mixed D', { Category: 'Mixed', 'Unrealised P/L': '-4,000.00', 'P/L %': '-5.00%' })
    await expectCells(funds, 'กองทุนอื่น I', { 'Unrealised P/L': '0.00', 'P/L %': '0.00%' })
    await expectTotals(funds, { Cost: '820,000.00', 'Market Value': '866,500.50', 'Unrealised P/L': '46,500.50' })
  })

  test('adds, edits and deletes a fund; a negative cost is refused', async ({ page }) => {
    await page.goto('/funds')
    const funds = resourceTable(page, 'Funds')
    await funds.add({ Fund: 'Fund New', Category: 'Gold', Cost: '1000', 'Market Value': '1100' })
    await expectCells(funds, 'Fund New', { 'Unrealised P/L': '100.00', 'P/L %': '10.00%' })

    await funds.edit('Fund New', { Category: 'RMF', 'Market Value': '900' })
    await expectCells(funds, 'Fund New', { Category: 'RMF', 'Unrealised P/L': '-100.00' })

    await funds.startEdit('Fund New')
    await funds.fill({ Cost: '-5' })
    await funds.save()
    await expect(funds.error).toContainText('cost must not be negative')
    await funds.cancel()

    await funds.remove('Fund New')
    await expect(funds.rows).toHaveCount(9)
  })
})

test.describe('Provident fund', () => {
  test('shows totals and return per year', async ({ page }) => {
    await page.goto('/pvd')
    const pvd = resourceTable(page, 'Yearly contributions')
    await expectCells(pvd, '2025', { 'Total Contrib.': '90,000.00', 'Total Profit': '4,500.00', 'Return %': '5.00%', Value: '94,500.00' })
    await expectCells(pvd, '2026', { 'Total Profit': '-1,800.00', 'Return %': '-3.33%' })
    await expectTotals(pvd, { Value: '146,700.00' })
  })

  test('edits a seeded year in place', async ({ page }) => {
    await page.goto('/pvd')
    const pvd = resourceTable(page, 'Yearly contributions')
    await pvd.edit('2025', { 'Self Contrib.': '70000' })
    await expectCells(pvd, '2025', { 'Self Contrib.': '70,000.00', Value: '104,500.00' })
  })

  test('adds and deletes a year; an out-of-range year is refused', async ({ page }) => {
    await page.goto('/pvd')
    const pvd = resourceTable(page, 'Yearly contributions')
    await pvd.add({ Year: '2030', 'Self Contrib.': '100', 'Company Contrib.': '50' })
    await expectCells(pvd, '2030', { 'Total Contrib.': '150.00' })

    await pvd.startAdd()
    await pvd.fill({ Year: '1800' })
    await pvd.save()
    await expect(pvd.error).toContainText('year is out of range')
    await pvd.cancel()

    await pvd.remove('2030')
    await expect(pvd.rows).toHaveCount(2)
  })
})

test.describe('Savings', () => {
  test('shows yearly interest from the rate', async ({ page }) => {
    await page.goto('/savings')
    const accounts = resourceTable(page, 'Accounts')
    await expectCells(accounts, 'ธนาคาร ก', { 'Interest Rate (%)': '1.50%', 'Interest / Year': '7,500.00' })
    await expectTotals(accounts, { Balance: '800,000.00', 'Interest / Year': '15,000.00' })
  })

  test('adds, edits and deletes an account; a rate over 100% is refused', async ({ page }) => {
    await page.goto('/savings')
    const accounts = resourceTable(page, 'Accounts')
    await accounts.add({ Bank: 'Bank New', Balance: '10000', 'Interest Rate (%)': '2.5' })
    await expectCells(accounts, 'Bank New', { 'Interest Rate (%)': '2.50%', 'Interest / Year': '250.00' })

    await accounts.edit('Bank New', { Balance: '20000' })
    await expectCells(accounts, 'Bank New', { 'Interest / Year': '500.00' })

    await accounts.startEdit('Bank New')
    await accounts.fill({ 'Interest Rate (%)': '101' })
    await accounts.save()
    await expect(accounts.error).toContainText('interest_bp must be between 0 and 10000')
    await accounts.cancel()

    await accounts.remove('Bank New')
    await expect(accounts.rows).toHaveCount(3)
  })
})

test.describe('Insurance', () => {
  test('numbers payments per company and totals premiums', async ({ page }) => {
    await page.goto('/insurance')
    const companies = resourceTable(page, 'Companies')
    await expectCells(companies, 'Insurer One', { Payments: '3', 'Total Paid': '75,000.00' })
    await expectCells(companies, 'Insurer Three', { Payments: '0', 'Total Paid': '0.00' })
    const log = resourceTable(page, 'Premium log')
    await expect(log.rows).toHaveCount(4)
    await expectCells(log, 'satang, no year', { Company: 'Insurer Two', 'Payment #': '1', Premium: '12,000.50' })
  })

  test('a recorded premium updates the company totals', async ({ page }) => {
    await page.goto('/insurance')
    const companies = resourceTable(page, 'Companies')
    const log = resourceTable(page, 'Premium log')

    await log.add({ Company: 'Insurer Three', Year: '2026', Premium: '1000', Note: 'e2e-premium' })
    await expectCells(log, 'e2e-premium', { 'Payment #': '1' })
    await expectCells(companies, 'Insurer Three', { Payments: '1', 'Total Paid': '1,000.00' })

    await log.edit('e2e-premium', { Premium: '1500' })
    await expectCells(companies, 'Insurer Three', { 'Total Paid': '1,500.00' })

    await log.startEdit('e2e-premium')
    await log.fill({ Premium: '0' })
    await log.save()
    await expect(log.error).toContainText('premium must be greater than 0')
    await log.cancel()

    await log.remove('e2e-premium')
    await expectCells(companies, 'Insurer Three', { Payments: '0' })
  })

  test('adds, edits and deletes a company; a duplicate name is refused', async ({ page }) => {
    await page.goto('/insurance')
    const companies = resourceTable(page, 'Companies')
    await companies.add({ Company: 'Insurer New', Note: 'e2e' })
    await companies.edit('Insurer New', { Note: 'edited' })
    await expectCells(companies, 'Insurer New', { Note: 'edited' })

    await companies.startAdd()
    await companies.fill({ Company: 'Insurer One' })
    await companies.save()
    await expect(companies.error).toContainText('already exists')
    await companies.cancel()

    await companies.remove('Insurer New')
    await expect(companies.rows).toHaveCount(3)
  })
})

test.describe('Salary & tax', () => {
  async function showYear(page: Page, year: number) {
    await page.locator('header.page-head').getByLabel('Year').fill(String(year))
  }

  test('the year filter shows only that year', async ({ page }) => {
    await page.goto('/salary')
    await showYear(page, 2025)
    const months = resourceTable(page, 'Monthly income 2025')
    await expect(months.rows).toHaveCount(12)
    await expectTotals(months, { Salary: '600,000.00', Bonus: '100,000.00' })
    await expectTotals(resourceTable(page, 'Deductions 2025'), { Amount: '94,000.00' })

    await showYear(page, 2026)
    await expect(resourceTable(page, 'Monthly income 2026').rows).toHaveCount(3)
    await expect(resourceTable(page, 'Deductions 2026').rows).toHaveCount(1)

    await showYear(page, 2031)
    await expect(resourceTable(page, 'Monthly income 2031').rows).toHaveCount(0)
    await expect(resourceTable(page, 'Monthly income 2031').section.locator('td.empty')).toBeVisible()
  })

  test('adds, edits and deletes a month in the selected year; a duplicate month is refused', async ({ page }) => {
    await page.goto('/salary')
    await showYear(page, 2031)
    const months = resourceTable(page, 'Monthly income 2031')

    await months.startAdd()
    await expect(months.editRow.getByLabel('Year', { exact: true })).toHaveValue('2031')
    await months.fill({ Month: 'Jan', Salary: '1000' })
    await months.save()
    await expectCells(months, 'Jan', { Salary: '1,000.00' })

    await months.edit('Jan', { Bonus: '500' })
    await expectTotals(months, { Bonus: '500.00' })

    await months.startAdd()
    await months.fill({ Month: 'Jan', Salary: '1' })
    await months.save()
    await expect(months.error).toContainText('already exists')
    await months.cancel()

    await months.remove('Jan')
    await expect(months.rows).toHaveCount(0)
  })

  test('adds, edits and deletes a deduction; a blank item is refused', async ({ page }) => {
    await page.goto('/salary')
    await showYear(page, 2031)
    const deductions = resourceTable(page, 'Deductions 2031')
    await deductions.add({ Item: 'e2e-deduction', Amount: '100' })
    await deductions.edit('e2e-deduction', { Amount: '250' })
    await expectCells(deductions, 'e2e-deduction', { Amount: '250.00' })

    await deductions.startAdd()
    await deductions.fill({ Item: ' ', Amount: '1' })
    await deductions.save()
    await expect(deductions.error).toContainText('item is required')
    await deductions.cancel()

    await deductions.remove('e2e-deduction')
    await expect(deductions.rows).toHaveCount(0)
  })
})

test.describe('Dividends', () => {
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  const money = (satang: number) => (satang / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  const pct = (ratio: number) => `${(ratio * 100).toFixed(2)}%`

  async function showYear(page: Page, year: number) {
    await page.locator('header.page-head').getByLabel('Year').fill(String(year))
  }
  const summary = (page: Page, year: number) =>
    page.locator('section.ledger').filter({ has: page.getByRole('heading', { name: `By asset ${year}`, exact: true }) })

  test('the yearly summary shows each asset, the months it paid and its yield', async ({ page }) => {
    await page.goto('/dividends')
    await showYear(page, 2025)
    const table = summary(page, 2025)
    const want = expected.dividends2025
    for (const s of Object.values(want.sources) as { name: string; payments: number; months: number[]; gross: number; withholdingTax: number; net: number; yieldPct: number }[]) {
      const row = table.locator('tbody tr').filter({ has: page.getByRole('cell', { name: s.name, exact: true }) })
      await expect(row.locator('td'), s.name).toHaveText([
        s.name,
        /.+/,
        String(s.payments),
        s.months.map((m) => months[m - 1]).join(', '),
        money(s.gross),
        money(s.withholdingTax),
        money(s.net),
        pct(s.yieldPct),
      ])
    }
    await expect(table.locator('tfoot td.num')).toHaveText([String(want.payments), money(want.gross), money(want.withholdingTax), money(want.net)])
    await expect(resourceTable(page, 'Payments 2025').rows).toHaveCount(want.payments)
  })

  test('a recorded payment shows up in the summary; tax above gross is refused', async ({ page }) => {
    await page.goto('/dividends')
    await showYear(page, 2025)
    const payments = resourceTable(page, 'Payments 2025')
    const table = summary(page, 2025)

    await payments.add({ Date: '2025-07-01', Source: 'Fund · Fund Bond B', Gross: '1000', 'Withholding Tax': '100', Note: 'e2e-dividend' })
    await expectCells(payments, 'e2e-dividend', { Net: '900.00' })
    const row = table.locator('tbody tr').filter({ hasText: 'Fund Bond B' })
    await expect(row.locator('td').nth(3)).toHaveText('Jul')

    await payments.edit('e2e-dividend', { Gross: '2000' })
    await expectCells(payments, 'e2e-dividend', { Net: '1,900.00' })
    await expect(row.locator('td').nth(6)).toHaveText('1,900.00')

    await payments.startEdit('e2e-dividend')
    await payments.fill({ 'Withholding Tax': '5000' })
    await payments.save()
    await expect(payments.error).toContainText('withholding_tax must be between 0 and the gross amount')
    await payments.cancel()

    await payments.remove('e2e-dividend')
    await expect(row).toHaveCount(0)
  })

  test('the year selector switches both tables', async ({ page }) => {
    await page.goto('/dividends')
    await showYear(page, 2026)
    await expect(resourceTable(page, 'Payments 2026').rows).toHaveCount(expected.dividends2026.payments)
    await expect(summary(page, 2026).locator('tfoot td.num').last()).toHaveText(money(expected.dividends2026.net))

    await showYear(page, 2031)
    await expect(summary(page, 2031).locator('td.empty')).toBeVisible()
    await expect(resourceTable(page, 'Payments 2031').rows).toHaveCount(0)
  })
})
