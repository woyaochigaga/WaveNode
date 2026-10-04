/** 格式化系统内部积分；积分沿用现有余额精度，不参与支付币种换算。 */
export function formatPoints(value: number | null | undefined): string {
  const amount = Number(value)
  if (!Number.isFinite(amount)) return '0.00'
  return amount.toFixed(2)
}
