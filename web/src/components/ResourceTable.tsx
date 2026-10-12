import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { api, del, post, put, type Row } from '../api'
import { fmtBp, fmtMoney, toBp, toSatang } from '../format'

export type ColumnType = 'text' | 'int' | 'money' | 'bp' | 'date' | 'select' | 'ref'

export interface Column {
  key: string
  label: string
  type: ColumnType
  computed?: boolean
  required?: boolean
  total?: boolean
  options?: { value: string | number; label: string }[]
  format?: (v: Row[string], row: Row) => string
  tone?: boolean
}

interface Props {
  path: string
  title: string
  hint?: string
  addLabel: string
  columns: Column[]
  filter?: Record<string, string | number>
  defaults?: Record<string, string>
  reloadKey?: number
  onChanged?: () => void
}

const numeric = (t: ColumnType) => t === 'int' || t === 'money' || t === 'bp'

function cellClass(col: Column, v: Row[string]): string {
  const cls = numeric(col.type) ? ['num'] : []
  if (col.tone && typeof v === 'number' && v !== 0) cls.push(v > 0 ? 'gain' : 'loss')
  return cls.join(' ')
}

function toInput(col: Column, v: Row[string]): string {
  if (v == null) return ''
  if (col.type === 'money') return (Number(v) / 100).toFixed(2)
  if (col.type === 'bp') return String(Number(v) / 100)
  return String(v)
}

function fromInput(col: Column, s: string): string | number | null {
  s = s.trim()
  if (s === '') return null
  if (col.type === 'money') return toSatang(s)
  if (col.type === 'bp') return toBp(s)
  if (col.type === 'int' || col.type === 'ref') return Number(s)
  return s
}

function display(col: Column, row: Row): string {
  const v = row[col.key]
  if (col.format) return col.format(v, row)
  if (v == null) return ''
  if (col.type === 'money') return fmtMoney(Number(v))
  if (col.type === 'bp') return fmtBp(Number(v))
  if (col.options) return col.options.find((o) => String(o.value) === String(v))?.label ?? String(v)
  return String(v)
}

export default function ResourceTable({ path, title, hint, addLabel, columns, filter, defaults, reloadKey, onChanged }: Props) {
  const [rows, setRows] = useState<Row[]>([])
  const [error, setError] = useState('')
  const [editing, setEditing] = useState<number | 'new' | null>(null)
  const [draft, setDraft] = useState<Record<string, string>>({})
  const editable = columns.filter((c) => !c.computed)
  const query = filter ? '?' + new URLSearchParams(Object.entries(filter).map(([k, v]) => [k, String(v)])) : ''

  const load = useCallback(() => {
    api<Row[]>(`/${path}${query}`)
      .then(setRows)
      .catch((e: Error) => setError(e.message))
  }, [path, query])

  useEffect(load, [load, reloadKey])

  const startNew = () => {
    setDraft({ ...Object.fromEntries(editable.map((c) => [c.key, ''])), ...defaults })
    setEditing('new')
    setError('')
  }
  const startEdit = (row: Row) => {
    setDraft(Object.fromEntries(editable.map((c) => [c.key, toInput(c, row[c.key])])))
    setEditing(Number(row.id))
    setError('')
  }

  const save = async (e: FormEvent) => {
    e.preventDefault()
    const body = Object.fromEntries(editable.map((c) => [c.key, fromInput(c, draft[c.key] ?? '')]))
    try {
      if (editing === 'new') await post(`/${path}`, body)
      else await put(`/${path}/${editing}`, body)
      setEditing(null)
      load()
      onChanged?.()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  const remove = async (row: Row) => {
    if (!confirm('ลบรายการนี้?')) return
    try {
      await del(`/${path}/${row.id}`)
      load()
      onChanged?.()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  const formId = `form-${path}`

  const input = (col: Column) => {
    const common = {
      form: formId,
      value: draft[col.key] ?? '',
      required: col.required,
      'aria-label': col.label,
      onChange: (e: { target: { value: string } }) => setDraft((d) => ({ ...d, [col.key]: e.target.value })),
    }
    if (col.options)
      return (
        <select {...common}>
          <option value="">—</option>
          {col.options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      )
    if (col.type === 'date') return <input type="date" {...common} />
    return <input inputMode={numeric(col.type) ? 'decimal' : undefined} {...common} />
  }

  const editRow = (key: string) => (
    <tr key={key} className="editing">
      {columns.map((c) => (
        <td key={c.key}>{c.computed ? <span className="muted">auto</span> : input(c)}</td>
      ))}
      <td className="actions">
        <button type="submit" form={formId} className="primary">
          Save
        </button>
        <button type="button" className="quiet" onClick={() => setEditing(null)}>
          Cancel
        </button>
      </td>
    </tr>
  )

  const hasTotals = columns.some((c) => c.total)

  return (
    <section className="ledger">
      <header className="ledger-head">
        <div>
          <h2>{title}</h2>
          {hint && <p className="muted small">{hint}</p>}
        </div>
        <button className="primary" onClick={startNew} disabled={editing !== null}>
          {addLabel}
        </button>
      </header>
      {error && <p className="error">{error}</p>}
      <form id={formId} onSubmit={save} />
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              {columns.map((c) => (
                <th key={c.key} className={numeric(c.type) ? 'num' : ''}>
                  {c.label}
                </th>
              ))}
              <th />
            </tr>
          </thead>
          <tbody>
            {rows.map((row) =>
              editing === row.id ? (
                editRow(String(row.id))
              ) : (
                <tr key={String(row.id)}>
                  {columns.map((c) => (
                    <td key={c.key} className={cellClass(c, row[c.key])}>
                      {display(c, row)}
                    </td>
                  ))}
                  <td className="actions">
                    <button className="quiet" onClick={() => startEdit(row)} disabled={editing !== null}>
                      Edit
                    </button>
                    <button className="quiet danger" onClick={() => remove(row)} disabled={editing !== null}>
                      Delete
                    </button>
                  </td>
                </tr>
              ),
            )}
            {editing === 'new' && editRow('new')}
            {rows.length === 0 && editing !== 'new' && (
              <tr>
                <td colSpan={columns.length + 1} className="empty">
                  ยังไม่มีรายการ กด “{addLabel}” เพื่อบันทึกรายการแรก
                </td>
              </tr>
            )}
          </tbody>
          {hasTotals && rows.length > 0 && (
            <tfoot>
              <tr>
                {columns.map((c, i) => (
                  <td key={c.key} className={numeric(c.type) ? 'num' : ''}>
                    {c.total ? fmtMoney(rows.reduce((s, r) => s + Number(r[c.key] ?? 0), 0)) : i === 0 ? 'Total' : ''}
                  </td>
                ))}
                <td />
              </tr>
            </tfoot>
          )}
        </table>
      </div>
    </section>
  )
}
