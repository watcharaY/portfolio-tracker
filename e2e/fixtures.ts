import { createHash, randomBytes } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import type { BrowserContext } from '@playwright/test'
import pg from 'pg'

const testdata = (name: string) => fileURLToPath(new URL(`../backend/testdata/${name}`, import.meta.url))

export const expected = JSON.parse(readFileSync(testdata('e2e-expected.json'), 'utf8'))

const databaseURL =
  process.env.E2E_DATABASE_URL ??
  `postgres://portfolio:portfolio@localhost:${process.env.E2E_DB_PORT ?? '55432'}/portfolio_e2e`

async function withDB<T>(fn: (c: pg.Client) => Promise<T>): Promise<T> {
  const client = new pg.Client({ connectionString: databaseURL })
  await client.connect()
  try {
    return await fn(client)
  } finally {
    await client.end()
  }
}

// Reloads backend/testdata/e2e-seed.sql. It truncates every table, sessions included.
export function resetDB() {
  const seed = readFileSync(testdata('e2e-seed.sql'), 'utf8')
  return withDB((c) => c.query(seed))
}

// Signs a context in by writing a session row, the same shape the API stores:
// the cookie holds a random token and the table holds its SHA-256. This skips the
// login form, which only auth.spec.ts exercises, and its 10/min rate limit.
export async function signIn(context: BrowserContext, baseURL: string, username: string) {
  const token = randomBytes(32).toString('base64url')
  const hash = createHash('sha256').update(token).digest('hex')
  const { rowCount } = await withDB((c) =>
    c.query(
      `INSERT INTO sessions (token_hash, user_id, expires_at)
       SELECT $1, id, now() + interval '1 hour' FROM users WHERE username = $2`,
      [hash, username],
    ),
  )
  if (rowCount !== 1) throw new Error(`no seeded user ${username}`)
  await context.addCookies([{ name: 'pt_session', value: token, url: baseURL }])
}
