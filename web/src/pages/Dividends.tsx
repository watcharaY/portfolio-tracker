import { useEffect, useState } from 'react'
import { api } from '../api'
import ResourceTable from '../components/ResourceTable'
import { fmtMoney, fmtPct } from '../format'
import { useOptions, useReload } from '../hooks'

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

const SOURCE_KINDS = [
  { type: 'broker', label: 'Brokerage', path: 'brokers', name: 'name' },
  { type: 'fund', label: 'Fund', path: 'funds', name: 'name' },
  { type: 'savings', label: 'Savings', path: 'savings-accounts', name: 'bank' },
  { type: 'insurer', label: 'Insurance', path: 'insurers', name: 'company' },
] as const

const kindLabel = (type: string) => SOURCE_KINDS.find((k) => k.type === type)?.label ?? type

interface SourceYear {
  source: string
  source_type: string
  source_name: string
  payments: number
  months: number[]
  gross: number
  withholding_tax: number
  net: number
  yield_pct: number | null
}

interface DividendYear {
  year: number
  sources: SourceYear[]
  payments: number
  gross: number
  withholding_tax: number
  net: number
}

// Every asset that can pay out, as "type:id" options for the source column.
function useSourceOptions(reload: number) {
  const brokers = useOptions('brokers', 'name', reload)
  const funds = useOptions('funds', 'name', reload)
  const savings = useOptions('savings-accounts', 'bank', reload)
  const insurers = useOptions('insurers', 'company', reload)
  return [brokers, funds, savings, insurers].flatMap((opts, i) =>
    opts.map((o) => ({ value: `${SOURCE_KINDS[i].type}:${o.value}`, label: `${SOURCE_KINDS[i].label} · ${o.label}` })),
  )
}

export default function Dividends() {
  const [year, setYear] = useState(new Date().getFullYear())
  const [reload, bump] = useReload()
  const [summary, setSummary] = useState<DividendYear | null>(null)
  const [error, setError] = useState('')
  const sources = useSourceOptions(reload)

  useEffect(() => {
    api<DividendYear>(`/dividends/summary?year=${year}`)
      .then(setSummary)
      .catch((e: Error) => setError(e.message))
  }, [year, reload])

  return (
    <>
      <header className="page-head">
        <h1>Dividend Tracking</h1>
        <label className="inline">
          Year
          <input type="number" value={year} onChange={(e) => setYear(Number(e.target.value))} style={{ width: '6rem' }} />
        </label>
      </header>

      <section className="ledger" aria-labelledby="dividend-summary-title">
        <header className="ledger-head">
          <div>
            <h2 id="dividend-summary-title">{`By asset ${year}`}</h2>
            <p className="muted small">ปันผล ดอกเบี้ย และเงินคืนที่ได้รับจริงในปีนี้ เดือนที่จ่ายบอกรอบการจ่าย (รายเดือน ครึ่งปี รายปี)</p>
          </div>
        </header>
        {error && <p className="error">{error}</p>}
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Asset</th>
                <th>Type</th>
                <th className="num">Payments</th>
                <th>Months paid</th>
                <th className="num">Gross</th>
                <th className="num">Withholding Tax</th>
                <th className="num">Net</th>
                <th className="num">Yield</th>
              </tr>
            </thead>
            <tbody>
              {summary?.sources.map((s) => (
                <tr key={s.source}>
                  <td>{s.source_name}</td>
                  <td>{kindLabel(s.source_type)}</td>
                  <td className="num">{s.payments}</td>
                  <td>{s.months.map((m) => MONTHS[m - 1]).join(', ')}</td>
                  <td className="num">{fmtMoney(s.gross)}</td>
                  <td className="num">{fmtMoney(s.withholding_tax)}</td>
                  <td className="num">{fmtMoney(s.net)}</td>
                  <td className="num">{fmtPct(s.yield_pct)}</td>
                </tr>
              ))}
              {summary && summary.sources.length === 0 && (
                <tr>
                  <td colSpan={8} className="empty">
                    ยังไม่มีรายการในปี {year}
                  </td>
                </tr>
              )}
            </tbody>
            {summary && summary.sources.length > 0 && (
              <tfoot>
                <tr>
                  <td>Total</td>
                  <td />
                  <td className="num">{summary.payments}</td>
                  <td />
                  <td className="num">{fmtMoney(summary.gross)}</td>
                  <td className="num">{fmtMoney(summary.withholding_tax)}</td>
                  <td className="num">{fmtMoney(summary.net)}</td>
                  <td />
                </tr>
              </tfoot>
            )}
          </table>
        </div>
      </section>

      <ResourceTable
        path="dividends"
        addLabel="Record payment"
        title={`Payments ${year}`}
        hint="Source ดึงจากหน้า Brokerage, Funds, Savings และ Insurance (PVD ไม่จ่ายเป็นเงินสดจึงไม่มี)"
        filter={{ year }}
        reloadKey={reload}
        onChanged={bump}
        columns={[
          { key: 'paid_on', label: 'Date', type: 'date', required: true },
          { key: 'source', label: 'Source', type: 'select', required: true, options: sources },
          { key: 'gross', label: 'Gross', type: 'money', required: true, total: true },
          { key: 'withholding_tax', label: 'Withholding Tax', type: 'money', total: true },
          { key: 'net', label: 'Net', type: 'money', computed: true, total: true },
          { key: 'note', label: 'Note', type: 'text' },
        ]}
      />
    </>
  )
}
