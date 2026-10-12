import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { passwordRules } from '../password'

export default function ChangePassword() {
  const { user, changePassword, logout } = useAuth()
  const navigate = useNavigate()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')

  const checks = passwordRules.map((r) => ({ label: r.label, ok: r.ok(next) }))
  const matches = next !== '' && next === confirm
  const ready = checks.every((c) => c.ok) && matches

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (next !== confirm) return setError('รหัสผ่านใหม่ไม่ตรงกัน')
    try {
      await changePassword(current, next)
      navigate('/', { replace: true })
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <div className="auth-page">
      <form className="sheet auth-sheet" onSubmit={submit}>
        <h1>ตั้งรหัสผ่านใหม่</h1>
        {user?.mustChangePassword && (
          <p className="notice small">บัญชีนี้ยังใช้รหัสผ่านเริ่มต้นอยู่ ต้องเปลี่ยนก่อนใช้งาน</p>
        )}
        <label>
          Current password
          <input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" required />
        </label>
        <label>
          New password
          <input
            type="password"
            value={next}
            onChange={(e) => setNext(e.target.value)}
            autoComplete="new-password"
            aria-describedby="password-rules"
            required
          />
        </label>
        <ul id="password-rules" className="checklist small" aria-label="Password rules">
          {checks.map((c) => (
            <li key={c.label} className={c.ok ? 'ok' : ''}>
              {c.ok ? '✓' : '○'} {c.label}
            </li>
          ))}
          <li className="muted">○ ห้ามเป็นรหัสยอดนิยม ห้ามมีชื่อผู้ใช้ และห้ามมีลำดับอย่าง 1234 / abcd (ตรวจตอนบันทึก)</li>
        </ul>
        <label>
          Confirm new password
          <input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" required />
        </label>
        <ul className="checklist small">
          <li className={matches ? 'ok' : ''}>
            {matches ? '✓' : '○'} ยืนยันรหัสผ่านตรงกัน
          </li>
        </ul>
        {error && <p className="error">{error}</p>}
        <button className="primary block" disabled={!ready}>
          Save password
        </button>
        <button type="button" className="link" onClick={logout}>
          Log out
        </button>
      </form>
    </div>
  )
}
