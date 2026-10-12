import ResourceTable from '../components/ResourceTable'
import { fmtPct } from '../format'

export default function Pvd() {
  return (
    <>
      <header className="page-head">
        <h1>Provident Fund (PVD)</h1>
      </header>
      <ResourceTable
        path="pvd-years"
        addLabel="Add year"
        title="Yearly contributions"
        hint="1 แถว = 1 ปี กรอกยอดสะสมของปีนั้น ระบบรวมยอดทุกปีให้เอง"
        columns={[
          { key: 'year', label: 'Year', type: 'int', required: true },
          { key: 'self_contribution', label: 'Self Contrib.', type: 'money', total: true },
          { key: 'company_contribution', label: 'Company Contrib.', type: 'money', total: true },
          { key: 'self_profit', label: 'Self Profit', type: 'money', total: true },
          { key: 'company_profit', label: 'Company Profit', type: 'money', total: true },
          { key: 'total_contribution', label: 'Total Contrib.', type: 'money', computed: true, total: true },
          { key: 'total_profit', label: 'Total Profit', type: 'money', computed: true, total: true, tone: true },
          {
            key: 'return_pct',
            label: 'Return %',
            type: 'int',
            computed: true,
            format: (v) => fmtPct(v as number | null),
          },
          { key: 'value', label: 'Value', type: 'money', computed: true, total: true },
          { key: 'fund_allocation', label: 'Fund Allocation', type: 'text' },
        ]}
      />
    </>
  )
}
