import ResourceTable from '../components/ResourceTable'
import { useOptions, useReload } from '../hooks'

export default function Insurance() {
  const [reload, bump] = useReload()
  const insurers = useOptions('insurers', 'company', reload)

  return (
    <>
      <header className="page-head">
        <h1>Insurance Premiums</h1>
      </header>
      <ResourceTable
        path="insurers"
        addLabel="Add insurer"
        title="Companies"
        reloadKey={reload}
        onChanged={bump}
        columns={[
          { key: 'company', label: 'Company', type: 'text', required: true },
          { key: 'payments', label: 'Payments', type: 'int', computed: true },
          { key: 'total_paid', label: 'Total Paid', type: 'money', computed: true, total: true },
          { key: 'note', label: 'Note', type: 'text' },
        ]}
      />
      <ResourceTable
        path="insurance-payments"
        addLabel="Record premium"
        title="Premium log"
        hint="งวดที่ (Payment #) นับให้อัตโนมัติตามบริษัท"
        reloadKey={reload}
        onChanged={bump}
        columns={[
          { key: 'insurer_id', label: 'Company', type: 'ref', required: true, options: insurers },
          { key: 'payment_no', label: 'Payment #', type: 'int', computed: true },
          { key: 'year', label: 'Year', type: 'int' },
          { key: 'premium', label: 'Premium', type: 'money', required: true, total: true },
          { key: 'note', label: 'Note', type: 'text' },
        ]}
      />
    </>
  )
}
