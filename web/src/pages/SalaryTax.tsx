import { useState } from 'react'
import ResourceTable from '../components/ResourceTable'

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

export default function SalaryTax() {
  const [year, setYear] = useState(new Date().getFullYear())

  return (
    <>
      <header className="page-head">
        <h1>Salary &amp; Tax</h1>
        <label className="inline">
          Year
          <input type="number" value={year} onChange={(e) => setYear(Number(e.target.value))} style={{ width: '6rem' }} />
        </label>
      </header>
      <ResourceTable
        path="salary-months"
        addLabel="Add month"
        title={`Monthly income ${year}`}
        filter={{ year }}
        defaults={{ year: String(year) }}
        columns={[
          { key: 'year', label: 'Year', type: 'int', required: true },
          { key: 'month', label: 'Month', type: 'ref', required: true, options: MONTHS.map((m, i) => ({ value: i + 1, label: m })) },
          { key: 'salary', label: 'Salary', type: 'money', total: true },
          { key: 'bonus', label: 'Bonus', type: 'money', total: true },
          { key: 'withholding_tax', label: 'Withholding Tax', type: 'money', total: true },
          { key: 'pvd', label: 'PVD', type: 'money', total: true },
          { key: 'sso', label: 'SSO', type: 'money', total: true },
        ]}
      />
      <ResourceTable
        path="tax-deductions"
        addLabel="Add deduction"
        title={`Deductions ${year}`}
        filter={{ year }}
        defaults={{ year: String(year) }}
        columns={[
          { key: 'year', label: 'Year', type: 'int', required: true },
          { key: 'item', label: 'Item', type: 'text', required: true },
          { key: 'amount', label: 'Amount', type: 'money', total: true },
        ]}
      />
    </>
  )
}
