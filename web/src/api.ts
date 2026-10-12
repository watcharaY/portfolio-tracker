export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

export const AUTH_EXPIRED = 'auth:expired'
export const PASSWORD_CHANGE_REQUIRED = 'auth:password-change-required'

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`/api${path}`, {
    credentials: 'same-origin',
    ...init,
    headers: { 'Content-Type': 'application/json', ...init.headers },
  })
  if (res.status === 204) return undefined as T
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const message: string = body.error ?? res.statusText
    if (res.status === 401 && !path.startsWith('/auth/login')) window.dispatchEvent(new Event(AUTH_EXPIRED))
    if (res.status === 403 && message === 'PASSWORD_CHANGE_REQUIRED') window.dispatchEvent(new Event(PASSWORD_CHANGE_REQUIRED))
    throw new ApiError(res.status, message)
  }
  return body as T
}

export const post = <T>(path: string, data: unknown) => api<T>(path, { method: 'POST', body: JSON.stringify(data) })
export const put = <T>(path: string, data: unknown) => api<T>(path, { method: 'PUT', body: JSON.stringify(data) })
export const del = (path: string) => api<void>(path, { method: 'DELETE' })

export type Row = Record<string, string | number | null>

export interface User {
  id: number
  username: string
  mustChangePassword: boolean
}

export interface AssetRow {
  key: string
  label: string
  value: number
  cost: number
  pl: number
  plPct: number | null
  allocationPct: number | null
}

export interface Dashboard {
  total: AssetRow
  classes: AssetRow[]
  estSavingsInterestPerYear: number
}
