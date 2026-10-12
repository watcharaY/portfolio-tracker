import ResourceTable from '../components/ResourceTable'
import { fmtPct } from '../format'

const CATEGORIES = ['Equity', 'Fixed Income', 'Money Market', 'Mixed', 'Gold', 'SSF', 'RMF', 'ThaiESG', 'Other']

export default function Funds() {
  return (
    <>
      <header className="page-head">
        <h1>Mutual Funds</h1>
      </header>
      <ResourceTable
        path="funds"
        addLabel="Add fund"
        title="Funds"
        hint="เพิ่มกองทุนใหม่ แล้วอัปเดต Market Value เป็นระยะ"
        columns={[
          { key: 'name', label: 'Fund', type: 'text', required: true },
          { key: 'category', label: 'Category', type: 'select', required: true, options: CATEGORIES.map((c) => ({ value: c, label: c })) },
          { key: 'cost', label: 'Cost', type: 'money', total: true },
          { key: 'market_value', label: 'Market Value', type: 'money', total: true },
          { key: 'unrealised_pl', label: 'Unrealised P/L', type: 'money', computed: true, total: true, tone: true },
          {
            key: 'pl_pct',
            label: 'P/L %',
            type: 'int',
            computed: true,
            format: (v) => fmtPct(v as number | null),
          },
          { key: 'note', label: 'Note', type: 'text' },
        ]}
      />
    </>
  )
}
