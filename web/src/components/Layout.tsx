import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth'

const tabs = [
  { to: '/', label: 'Overview', end: true },
  { to: '/brokerage', label: 'Brokerage' },
  { to: '/funds', label: 'Funds' },
  { to: '/pvd', label: 'Provident fund' },
  { to: '/savings', label: 'Savings' },
  { to: '/insurance', label: 'Insurance' },
  { to: '/salary', label: 'Salary & tax' },
]

export default function Layout() {
  const { user, logout } = useAuth()
  return (
    <>
      <header className="masthead">
        <div className="masthead-inner">
          <div className="brand">
            Portfolio <span>สมุดพอร์ต</span>
          </div>
          <nav className="tabs" aria-label="Sections">
            {tabs.map((t) => (
              <NavLink key={t.to} to={t.to} end={t.end}>
                {t.label}
              </NavLink>
            ))}
          </nav>
          <div className="account small">
            <span className="muted">{user?.username}</span>
            <button className="quiet" onClick={logout}>
              Log out
            </button>
          </div>
        </div>
      </header>
      <main className="page">
        <Outlet />
      </main>
    </>
  )
}
