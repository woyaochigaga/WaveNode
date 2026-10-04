import { describe, expect, it } from 'vitest'
import { formatPoints } from '@/utils/points'

describe('formatPoints', () => {
  it('keeps balance precision without adding a currency symbol', () => {
    expect(formatPoints(12.3)).toBe('12.30')
    expect(formatPoints(undefined)).toBe('0.00')
  })
})
