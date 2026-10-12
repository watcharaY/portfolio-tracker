const thb = new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

export const fmtMoney = (satang: number | null | undefined) =>
  satang == null ? '' : thb.format(satang / 100)

export const fmtPct = (ratio: number | null | undefined, digits = 2) =>
  ratio == null ? '–' : `${(ratio * 100).toFixed(digits)}%`

export const fmtSignedMoney = (satang: number) => (satang > 0 ? '+' : '') + fmtMoney(satang)

export const toSatang = (thbText: string) => Math.round(Number(thbText.replace(/,/g, '')) * 100)

export const fmtBp = (bp: number | null | undefined) => (bp == null ? '' : `${(bp / 100).toFixed(2)}%`)
export const toBp = (pctText: string) => Math.round(Number(pctText.replace('%', '')) * 100)
