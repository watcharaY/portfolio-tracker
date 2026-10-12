import { expect, test } from '@playwright/test'
import { readdirSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

// Fails when the app gains a route or a table that no spec exercises.
const read = (rel: string) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), 'utf8')
const specsDir = fileURLToPath(new URL('.', import.meta.url))
const specs = readdirSync(specsDir)
  .filter((f) => f.endsWith('.spec.ts') && f !== 'spec-coverage.spec.ts')
  .map((f) => readFileSync(`${specsDir}/${f}`, 'utf8'))
  .join('\n')

test('every route in App.tsx is visited by a spec', () => {
  const app = read('../../web/src/App.tsx')
  const routes = [...app.matchAll(/<Route[^>]*\spath="([^"*]+)"/g)].map((m) => '/' + m[1].replace(/^\//, ''))
  expect(routes.length).toBeGreaterThan(5)
  const missing = routes.filter((r) => !specs.includes(`'${r}'`) && !specs.includes(`${r}$`))
  expect(missing, 'routes without a spec').toEqual([])
})

test('every ResourceTable on every page is driven by a spec', () => {
  const pagesDir = fileURLToPath(new URL('../../web/src/pages/', import.meta.url))
  const titles = readdirSync(pagesDir).flatMap((f) => {
    const src = readFileSync(`${pagesDir}/${f}`, 'utf8')
    return [...src.matchAll(/<ResourceTable[\s\S]*?title=(?:"([^"]+)"|\{`([^`$]+))/g)].map((m) => (m[1] ?? m[2]).trim())
  })
  expect(titles.length).toBeGreaterThan(8)
  const missing = titles.filter((t) => !specs.includes(`resourceTable(page, '${t}`) && !specs.includes(`title: '${t}`))
  expect(missing, 'tables without a spec').toEqual([])
})
