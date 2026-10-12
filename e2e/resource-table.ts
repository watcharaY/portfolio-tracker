import { expect, type Locator, type Page } from '@playwright/test'

// Drives one ResourceTable section by its heading. Inputs in the edit row carry
// aria-label = column label, and selects are filled by option label, so a test
// talks about cells the way the page shows them.
export type Values = Record<string, string>

export function resourceTable(page: Page, title: string) {
  const section = page.locator('section.ledger').filter({ has: page.getByRole('heading', { name: title, exact: true }) })
  const editRow = section.locator('tr.editing')
  const rows = section.locator('tbody tr:not(.editing)').filter({ hasNot: page.locator('td.empty') })

  async function columnIndex(label: string) {
    const headers = await section.locator('thead th').allInnerTexts()
    const i = headers.findIndex((h) => h.trim() === label)
    if (i < 0) throw new Error(`no column "${label}" in "${title}" (have ${headers.join(', ')})`)
    return i
  }

  async function fill(values: Values) {
    for (const [label, value] of Object.entries(values)) {
      const field = editRow.getByLabel(label, { exact: true })
      if ((await field.evaluate((el) => el.tagName)) === 'SELECT') await field.selectOption({ label: value })
      else await field.fill(value)
    }
  }

  return {
    section,
    rows,
    editRow,
    error: section.locator('p.error'),
    row: (text: string) => rows.filter({ hasText: text }),

    async cell(rowText: string, label: string): Promise<Locator> {
      return rows.filter({ hasText: rowText }).locator('td').nth(await columnIndex(label))
    },
    async total(label: string): Promise<Locator> {
      return section.locator('tfoot td').nth(await columnIndex(label))
    },

    async startAdd() {
      await section.locator('header button.primary').click()
      await expect(editRow).toBeVisible()
    },
    async startEdit(rowText: string) {
      await rows.filter({ hasText: rowText }).getByRole('button', { name: 'Edit' }).click()
      await expect(editRow).toBeVisible()
    },
    fill,
    async save() {
      await editRow.getByRole('button', { name: 'Save' }).click()
    },
    async cancel() {
      await editRow.getByRole('button', { name: 'Cancel' }).click()
      await expect(editRow).toHaveCount(0)
    },

    async add(values: Values) {
      await this.startAdd()
      await fill(values)
      await this.save()
      await expect(editRow).toHaveCount(0)
    },
    async edit(rowText: string, values: Values) {
      await this.startEdit(rowText)
      await fill(values)
      await this.save()
      await expect(editRow).toHaveCount(0)
    },
    async remove(rowText: string) {
      page.once('dialog', (d) => d.accept())
      await rows.filter({ hasText: rowText }).getByRole('button', { name: 'Delete' }).click()
      await expect(rows.filter({ hasText: rowText })).toHaveCount(0)
    },
  }
}
