import ResourceTable from '../components/ResourceTable'
import { fmtPct } from '../format'
import { useOptions, useReload } from '../hooks'

export default function Brokerage() {
  const [reload, bump] = useReload()
  const brokers = useOptions('brokers', 'name', reload)

  return (
    <>
      <header className="page-head">
        <h1>Brokerage Accounts</h1>
      </header>
      <ResourceTable
        path="brokers"
        addLabel="Add broker"
        title="Brokers"
        hint="อัปเดต Current Value เป็นระยะ ส่วน Deposits/Withdrawals คำนวณจากรายการด้านล่าง"
        reloadKey={reload}
        onChanged={bump}
        columns={[
          { key: 'name', label: 'Broker', type: 'text', required: true },
          { key: 'current_value', label: 'Current Value', type: 'money', total: true },
          { key: 'deposits', label: 'Deposits', type: 'money', computed: true, total: true },
          { key: 'withdrawals', label: 'Withdrawals', type: 'money', computed: true, total: true },
          { key: 'net_capital', label: 'Net Capital', type: 'money', computed: true, total: true },
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
      <ResourceTable
        path="broker-transactions"
        addLabel="Record deposit / withdrawal"
        title="Deposit / Withdraw log"
        reloadKey={reload}
        onChanged={bump}
        defaults={{ type: 'deposit' }}
        columns={[
          { key: 'txn_date', label: 'Date', type: 'date' },
          { key: 'broker_id', label: 'Broker', type: 'ref', required: true, options: brokers },
          {
            key: 'type',
            label: 'Type',
            type: 'select',
            required: true,
            options: [
              { value: 'deposit', label: 'Deposit' },
              { value: 'withdraw', label: 'Withdraw' },
            ],
          },
          { key: 'amount', label: 'Amount', type: 'money', required: true },
          { key: 'note', label: 'Note', type: 'text' },
        ]}
      />
    </>
  )
}
