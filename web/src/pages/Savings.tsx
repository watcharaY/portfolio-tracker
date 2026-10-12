import ResourceTable from '../components/ResourceTable'

export default function Savings() {
  return (
    <>
      <header className="page-head">
        <h1>Savings Accounts</h1>
      </header>
      <ResourceTable
        path="savings-accounts"
        addLabel="Add account"
        title="Accounts"
        hint="อัตราดอกเบี้ยกรอกเป็น % เช่น 2.5"
        columns={[
          { key: 'bank', label: 'Bank', type: 'text', required: true },
          { key: 'balance', label: 'Balance', type: 'money', total: true },
          { key: 'interest_bp', label: 'Interest Rate (%)', type: 'bp' },
          { key: 'interest_per_year', label: 'Interest / Year', type: 'money', computed: true, total: true },
          { key: 'note', label: 'Note', type: 'text' },
        ]}
      />
    </>
  )
}
