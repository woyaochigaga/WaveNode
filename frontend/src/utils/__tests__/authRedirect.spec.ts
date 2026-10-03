import { describe, expect, it } from 'vitest'
import { resolvePostLoginPath } from '@/utils/authRedirect'

describe('resolvePostLoginPath', () => {
  it('sends admins to the admin dashboard by default', () => {
    expect(resolvePostLoginPath(undefined, true)).toBe('/admin/dashboard')
  })

  it('sends regular users to the user dashboard by default', () => {
    expect(resolvePostLoginPath(undefined, false)).toBe('/dashboard')
  })

  it('keeps a valid explicit in-app redirect', () => {
    expect(resolvePostLoginPath('/admin/audit-logs?page=2', true)).toBe('/admin/audit-logs?page=2')
  })

  it.each(['https://evil.example', '//evil.example', 'dashboard', '/bad\npath'])(
    'rejects unsafe redirect %s',
    (redirect) => {
      expect(resolvePostLoginPath(redirect, true)).toBe('/admin/dashboard')
    }
  )
})
