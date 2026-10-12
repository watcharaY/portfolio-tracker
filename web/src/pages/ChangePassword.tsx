import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'

export default function ChangePassword() {
  const { user, changePassword, logout } = useAuth()
  const navigate = useNavigate()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')

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
          New password (≥ 8 ตัวอักษร)
          <input type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" minLength={8} required />
        </label>
        <label>
          Confirm new password
          <input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" required />
        </label>
        {error && <p className="error">{error}</p>}
        <button className="primary block">Save password</button>
        <button type="button" className="link" onClick={logout}>
          Log out
        </button>
      </form>
    </div>
  )
}
