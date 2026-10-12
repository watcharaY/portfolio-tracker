import { useEffect, useState } from 'react'
import { api, type Row } from './api'

export function useOptions(path: string, labelKey: string, reloadKey = 0) {
  const [options, setOptions] = useState<{ value: number; label: string }[]>([])
  useEffect(() => {
    api<Row[]>(`/${path}`)
      .then((rows) => setOptions(rows.map((r) => ({ value: Number(r.id), label: String(r[labelKey]) }))))
      .catch(() => setOptions([]))
  }, [path, labelKey, reloadKey])
  return options
}

export function useReload(): [number, () => void] {
  const [n, setN] = useState(0)
  return [n, () => setN((x) => x + 1)]
}
