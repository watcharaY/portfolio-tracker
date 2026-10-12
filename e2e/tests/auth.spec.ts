import { expect, test, type Page } from '@playwright/test'
import { expected, resetDB } from '../fixtures'

const { ready, forcedChange } = expected.users as Record<string, { username: string; password: string }>

async function login(page: Page, username: string, password: string) {
  await page.goto('/login')
  await page.getByLabel('Username').fill(username)
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: 'Sign in' }).click()
}

test.beforeEach(resetDB)

test('signed-out visitors are sent to the login page', async ({ page }) => {
  await page.goto('/funds')
  await expect(page).toHaveURL(/\/login$/)
})

test('a wrong password shows an error and stays on login', async ({ page }) => {
  await login(page, ready.username, 'not-the-password')
  await expect(page.locator('.error')).toBeVisible()
  await expect(page).toHaveURL(/\/login$/)
})

test('sign in lands on the overview, log out returns to login', async ({ page }) => {
  await login(page, ready.username, ready.password)
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('navigation', { name: 'Sections' })).toBeVisible()

  await page.getByRole('button', { name: 'Log out' }).click()
  await expect(page).toHaveURL(/\/login$/)
  await page.goto('/')
  await expect(page).toHaveURL(/\/login$/)
})

test('the seeded admin must change the default password before using the app', async ({ page }) => {
  await login(page, forcedChange.username, forcedChange.password)
  await expect(page).toHaveURL(/\/change-password$/)

  await page.goto('/funds')
  await expect(page).toHaveURL(/\/change-password$/)

  const newPassword = 'Another-Pass-77'
  await page.getByLabel('Current password').fill(forcedChange.password)
  await page.getByLabel(/^New password/).fill(newPassword)
  await page.getByLabel('Confirm new password').fill(newPassword)
  await page.getByRole('button', { name: 'Save password' }).click()
  await expect(page).toHaveURL(/\/$/)

  await page.getByRole('button', { name: 'Log out' }).click()
  await login(page, forcedChange.username, forcedChange.password)
  await expect(page.locator('.error')).toBeVisible()
  await login(page, forcedChange.username, newPassword)
  await expect(page).toHaveURL(/\/$/)
})
