import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, type Dashboard as DashboardData } from '../api'
import AllocationDonut from '../components/AllocationDonut'
import { fmtMoney, fmtPct, fmtSignedMoney } from '../format'

const SERIES: Record<string, string> = {
  stocks: 'var(--series-1)',
  funds: 'var(--series-2)',
  pvd: 'var(--series-3)',
  savings: 'var(--series-4)',
  insurance: 'var(--series-5)',
}

const plClass = (n: number) => (n > 0 ? 'gain' : n < 0 ? 'loss' : '')

export default function Dashboard() {
  const [data, setData] = useState<DashboardData | null>(null)
  const [error, setError] = useState('')
  const [active, setActive] = useState<string | null>(null)

  useEffect(() => {
    api<DashboardData>('/dashboard').then(setData).catch((e: Error) => setError(e.message))
  }, [])

  if (error) return <p className="error">โหลดภาพรวมไม่สำเร็จ: {error}</p>
  if (!data) return <p className="muted">Loading…</p>

  const { total, classes } = data
  const visible = classes.filter((c) => c.value > 0)

  if (visible.length === 0) {
    return (
      <section className="hero">
        <h1 className="hero-line">สมุดพอร์ตยังว่างอยู่</h1>
        <p className="hero-sub">
          เริ่มจากบันทึกบัญชีที่มีอยู่ เช่น <Link to="/brokerage">โบรกเกอร์</Link> หรือ <Link to="/savings">บัญชีเงินฝาก</Link>{' '}
          แล้วภาพรวมจะคำนวณให้ที่หน้านี้
        </p>
      </section>
    )
  }

  const direction = total.pl > 0 ? 'มากกว่า' : total.pl < 0 ? 'น้อยกว่า' : 'เท่ากับ'
  const absPct = total.plPct == null ? null : Math.abs(total.plPct)
  return (
    <>
      <section className="hero" aria-label="Portfolio balance">
        <h1 className="hero-line">
          ทั้งพอร์ตตอนนี้มี <strong>{fmtMoney(total.value)}</strong> บาท
        </h1>
        <p className="hero-sub">
          {direction}เงินที่ใส่ไป
          {total.pl !== 0 && (
            <>
              {' '}
              <strong className={plClass(total.pl)}>
                {fmtMoney(Math.abs(total.pl))} บาท ({fmtPct(absPct)})
              </strong>
            </>
          )}{' '}
          จากต้นทุนรวม {fmtMoney(total.cost)} บาท
        </p>
        <p className="hero-note small muted">
          ดอกเบี้ยเงินฝากโดยประมาณ {fmtMoney(data.estSavingsInterestPerYear)} บาทต่อปี
        </p>
      </section>

      <section className="ledger" aria-labelledby="alloc-title">
        <div className="ledger-head">
          <div>
            <h2 id="alloc-title">Allocation</h2>
            <p className="small muted">Savings และ Insurance ไม่มีกำไร/ขาดทุน มูลค่าเท่ากับเงินที่ใส่ไป</p>
          </div>
        </div>
        <div className="allocation">
          <AllocationDonut rows={visible} total={total.value} colors={SERIES} active={active} onActive={setActive} />
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Asset class</th>
                  <th className="num">Value</th>
                  <th className="num">Cost / paid</th>
                  <th className="num">Unrealised P/L</th>
                  <th className="num">P/L %</th>
                  <th className="num">Share</th>
                </tr>
              </thead>
              <tbody onMouseLeave={() => setActive(null)}>
                {classes.map((c) => (
                  <tr
                    key={c.key}
                    className={active && active !== c.key ? 'dim' : ''}
                    onMouseEnter={() => c.value > 0 && setActive(c.key)}
                  >
                    <td>
                      <span className="swatch" style={{ background: SERIES[c.key] }} />
                      {c.label}
                    </td>
                    <td className="num">{fmtMoney(c.value)}</td>
                    <td className="num">{fmtMoney(c.cost)}</td>
                    <td className={'num ' + plClass(c.pl)}>{fmtSignedMoney(c.pl)}</td>
                    <td className="num">{fmtPct(c.plPct)}</td>
                    <td className="num">{fmtPct(c.allocationPct, 1)}</td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr>
                  <td>Total</td>
                  <td className="num">{fmtMoney(total.value)}</td>
                  <td className="num">{fmtMoney(total.cost)}</td>
                  <td className={'num ' + plClass(total.pl)}>{fmtSignedMoney(total.pl)}</td>
                  <td className="num">{fmtPct(total.plPct)}</td>
                  <td className="num">{fmtPct(total.allocationPct, 1)}</td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      </section>
    </>
  )
}
