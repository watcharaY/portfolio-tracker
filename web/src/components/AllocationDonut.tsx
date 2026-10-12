import type { AssetRow } from '../api'
import { fmtMoney, fmtPct } from '../format'

interface Props {
  rows: AssetRow[]
  total: number
  colors: Record<string, string>
  active: string | null
  onActive: (key: string | null) => void
}

const SIZE = 220
const R_OUT = 100
const R_IN = 70
const C = SIZE / 2

function point(r: number, angle: number) {
  return [C + r * Math.sin(angle), C - r * Math.cos(angle)]
}

function arcPath(start: number, end: number, rOut = R_OUT): string {
  // A single full ring cannot be drawn as one arc; split it in two.
  if (end - start >= Math.PI * 2 - 1e-6) {
    return arcPath(0, Math.PI, rOut) + arcPath(Math.PI, Math.PI * 2, rOut)
  }
  const large = end - start > Math.PI ? 1 : 0
  const [x0, y0] = point(rOut, start)
  const [x1, y1] = point(rOut, end)
  const [x2, y2] = point(R_IN, end)
  const [x3, y3] = point(R_IN, start)
  return `M${x0},${y0} A${rOut},${rOut} 0 ${large} 1 ${x1},${y1} L${x2},${y2} A${R_IN},${R_IN} 0 ${large} 0 ${x3},${y3} Z`
}

export default function AllocationDonut({ rows, total, colors, active, onActive }: Props) {
  let angle = 0
  const slices = rows.map((r) => {
    const sweep = total > 0 ? (r.value / total) * Math.PI * 2 : 0
    const s = { row: r, start: angle, end: angle + sweep }
    angle += sweep
    return s
  })
  const hovered = rows.find((r) => r.key === active)

  return (
    <svg
      viewBox={`0 0 ${SIZE} ${SIZE}`}
      className="donut"
      role="img"
      aria-label={'Allocation: ' + rows.map((r) => `${r.label} ${fmtPct(r.allocationPct, 1)}`).join(', ')}
      onMouseLeave={() => onActive(null)}
    >
      <g className="donut-ring">
        {slices.map(({ row, start, end }) => (
          <path
            key={row.key}
            d={arcPath(start, end, active === row.key ? R_OUT + 5 : R_OUT)}
            fill={colors[row.key]}
            className={'donut-seg' + (active && active !== row.key ? ' dim' : '')}
            tabIndex={0}
            aria-label={`${row.label}: ${fmtMoney(row.value)} THB, ${fmtPct(row.allocationPct, 1)}`}
            onMouseEnter={() => onActive(row.key)}
            onFocus={() => onActive(row.key)}
            onBlur={() => onActive(null)}
          />
        ))}
      </g>
      <text x={C} y={C - 14} className="donut-label">
        {hovered ? hovered.label : 'Total'}
      </text>
      <text x={C} y={C + 8} className="donut-value">
        {fmtMoney(hovered ? hovered.value : total)}
      </text>
      <text x={C} y={C + 28} className="donut-label">
        {hovered ? fmtPct(hovered.allocationPct, 1) : 'THB'}
      </text>
    </svg>
  )
}
