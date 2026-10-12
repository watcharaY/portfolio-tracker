// Mirrors the length and character rules of domain.ValidateNewPassword so the
// form can tick them off while typing. The server also refuses sequences,
// repeats, the username and leaked passwords, and says which on save.
export interface PasswordRule {
  label: string
  ok: (password: string) => boolean
}

const chars = (p: string) => [...p.normalize('NFC')].length

export const passwordRules: PasswordRule[] = [
  { label: '12–64 ตัวอักษร', ok: (p) => chars(p) >= 12 && chars(p) <= 64 && new TextEncoder().encode(p).length <= 72 },
  { label: 'ตัวพิมพ์เล็ก a–z', ok: (p) => /[a-z]/.test(p) },
  { label: 'ตัวพิมพ์ใหญ่ A–Z', ok: (p) => /[A-Z]/.test(p) },
  { label: 'ตัวเลข 0–9', ok: (p) => /[0-9]/.test(p) },
  { label: 'สัญลักษณ์หรือเว้นวรรค เช่น - ! @', ok: (p) => /[^\p{L}\p{N}]/u.test(p) },
]
